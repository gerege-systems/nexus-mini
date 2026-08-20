// Package db — холболтын pool-ууд ба nexus.DB-ийн хэрэгжүүлэлт.
//
// Хоёр pool: App (nexus_app, RLS үйлчилнэ) ба Admin (nexus_admin,
// nexus_platform гишүүн тул бодлогууд платформ гэж таньдаг). Аппын tenant
// урсгал бүхэлдээ App pool-оор явж, дуудлага бүрийн өмнө холболт дээр
// app.tenant_id / app.user_id тохируулагдана — RLS-ийг тойрох зам код
// талд байхгүй.
package db

import (
	"context"
	"fmt"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Pools struct {
	App   *pgxpool.Pool
	Admin *pgxpool.Pool
}

func Connect(ctx context.Context, appURL, adminURL string) (*Pools, error) {
	app, err := pgxpool.New(ctx, appURL)
	if err != nil {
		return nil, fmt.Errorf("app pool: %w", err)
	}
	admin, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		app.Close()
		return nil, fmt.Errorf("admin pool: %w", err)
	}
	if err := app.Ping(ctx); err != nil {
		app.Close()
		admin.Close()
		return nil, fmt.Errorf("app ping: %w", err)
	}
	return &Pools{App: app, Admin: admin}, nil
}

func (p *Pools) Close() {
	p.App.Close()
	p.Admin.Close()
}

// TenantDB нь nexus.DB-ийн хэрэгжүүлэлт. Холболтыг pool-оос авч RLS
// context-оо тохируулаад, ажил дуусмагц цэвэрлэж буцаана.
type TenantDB struct {
	pool *pgxpool.Pool
}

func NewTenantDB(pool *pgxpool.Pool) *TenantDB { return &TenantDB{pool: pool} }

var _ nexus.DB = (*TenantDB)(nil)

// acquire нь context-ийн tenant/user-ийг холболт дээр session түвшинд
// тохируулна. release нь цэвэрлэж pool-д буцаана — бохир холболт дараагийн
// хүсэлтэд өөр tenant-ийн эрхээр очих ёсгүй.
func (d *TenantDB) acquire(ctx context.Context) (*pgxpool.Conn, func(), error) {
	conn, err := d.pool.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	_, err = conn.Exec(ctx,
		`SELECT set_config('app.tenant_id', $1::text, false),
		        set_config('app.user_id',   $2::text, false)`,
		identity.TenantID(ctx), identity.UserID(ctx))
	if err != nil {
		conn.Release()
		return nil, nil, err
	}
	release := func() {
		// context цуцлагдсан байж болох тул цэвэрлэгээг өөрийн context-оор.
		_, rerr := conn.Exec(context.Background(),
			`SELECT set_config('app.tenant_id', '', false),
			        set_config('app.user_id',   '', false)`)
		if rerr != nil {
			// Цэвэрлэж чадаагүй холболтыг pool-д буцаахгүй — хаяна.
			conn.Conn().Close(context.Background())
		}
		conn.Release()
	}
	return conn, release, nil
}

func (d *TenantDB) Tx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	conn, release, err := d.acquire(ctx)
	if err != nil {
		return err
	}
	defer release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (d *TenantDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	conn, release, err := d.acquire(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		release()
		return nil, err
	}
	return &releasingRows{Rows: rows, release: release}, nil
}

func (d *TenantDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	rows, err := d.Query(ctx, sql, args...)
	if err != nil {
		return errRow{err}
	}
	return &firstRow{rows: rows}
}

func (d *TenantDB) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	conn, release, err := d.acquire(ctx)
	if err != nil {
		return pgconn.CommandTag{}, err
	}
	defer release()
	return conn.Exec(ctx, sql, args...)
}

// releasingRows — Close дээр холболтоо цэвэрлэж буцаадаг pgx.Rows.
type releasingRows struct {
	pgx.Rows
	release  func()
	released bool
}

func (r *releasingRows) Close() {
	r.Rows.Close()
	if !r.released {
		r.released = true
		r.release()
	}
}

type errRow struct{ err error }

func (e errRow) Scan(...any) error { return e.err }

// firstRow — pgx.Row семантик: эхний мөрийг Scan хийгээд хаана.
type firstRow struct{ rows pgx.Rows }

func (f *firstRow) Scan(dest ...any) error {
	defer f.rows.Close()
	if !f.rows.Next() {
		if err := f.rows.Err(); err != nil {
			return err
		}
		return pgx.ErrNoRows
	}
	if err := f.rows.Scan(dest...); err != nil {
		return err
	}
	return f.rows.Err()
}
