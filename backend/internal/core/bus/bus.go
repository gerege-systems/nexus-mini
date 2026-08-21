// Package bus — процесс хоорондын cache invalidation, Postgres LISTEN/NOTIFY-
// ээр. Нэмэлт хамаарал (Redis) шаардахгүй: DB аль хэдийн бий, ачаалал нь
// хэдэн байт/сек. Олон API процесс (Docker scale, systemd хоёр instance)
// ажиллахад нэг дээр хасагдсан эрх нөгөө дээр 30с cache-д үлдэхгүй.
//
// Баталгаа: at-most-once delivery (холболт тасарсан зуурын мэдэгдэл
// алдагдана) — тиймээс cache-ийн TTL хэвээр үлдэнэ; bus нь хоцролтыг
// ≤TTL-ээс ~0 болгож байгаа хэрэг, зөв байдлын суурь биш.
package bus

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const channel = "nexus_invalidate"

type Bus struct {
	pool     *pgxpool.Pool
	mu       sync.Mutex
	handlers []func(payload string)
}

func New(pool *pgxpool.Pool) *Bus { return &Bus{pool: pool} }

// Publish — бусад процесст мэдэгдэнэ (өөртөө ч ирнэ — давхар локал
// invalidate хямд тул шүүхгүй). Алдаа нь үндсэн үйлдлийг унагаахгүй.
func (b *Bus) Publish(payload string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if _, err := b.pool.Exec(ctx, `SELECT pg_notify($1, $2)`, channel, payload); err != nil {
		log.Printf("bus publish: %v", err)
	}
}

func (b *Bus) Subscribe(fn func(payload string)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = append(b.handlers, fn)
}

// Run — тусдаа холболтоор LISTEN хийж, тасарвал экспоненциал хүлээлттэй
// дахин холбогдоно. ctx цуцлагдтал ажиллана (goroutine-д дууд).
func (b *Bus) Run(ctx context.Context) {
	backoff := time.Second
	for ctx.Err() == nil {
		if err := b.listenOnce(ctx); err != nil && ctx.Err() == nil {
			log.Printf("bus listen: %v (дахин %s)", err, backoff)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}
		backoff = time.Second
	}
}

func (b *Bus) listenOnce(ctx context.Context) error {
	cfg := b.pool.Config().ConnConfig.Copy()
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() {
		cctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = conn.Close(cctx)
	}()
	if _, err := conn.Exec(ctx, "LISTEN "+channel); err != nil {
		return err
	}
	for {
		n, err := conn.WaitForNotification(ctx)
		if err != nil {
			return err
		}
		b.mu.Lock()
		hs := append([]func(string){}, b.handlers...)
		b.mu.Unlock()
		for _, h := range hs {
			h(n.Payload)
		}
	}
}
