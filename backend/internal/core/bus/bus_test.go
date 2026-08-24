package bus

// LISTEN/NOTIFY: нэг процессын Publish нөгөө процессын Subscribe-д хүрнэ.

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPublishReachesSubscriber(t *testing.T) {
	url := os.Getenv("NEXUS_TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	// Хоёр тусдаа Bus = хоёр процессын дүр.
	listener, publisher := New(pool), New(pool)
	got := make(chan string, 4)
	listener.Subscribe(func(p string) { got <- p })
	runCtx, stop := context.WithCancel(ctx)
	defer stop()
	go listener.Run(runCtx)
	time.Sleep(700 * time.Millisecond) // LISTEN тогтох хүртэл

	publisher.Publish("grants:11111111-1111-1111-1111-111111111111")
	select {
	case p := <-got:
		if p != "grants:11111111-1111-1111-1111-111111111111" {
			t.Fatalf("payload = %q", p)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("мэдэгдэл хүрсэнгүй")
	}
	// Хоёр дахь мэдэгдэл ч ирнэ (холболт амьд).
	publisher.Publish("gate:22222222-2222-2222-2222-222222222222")
	select {
	case p := <-got:
		if p[:5] != "gate:" {
			t.Fatalf("payload = %q", p)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("хоёр дахь мэдэгдэл хүрсэнгүй")
	}
	// ctx цуцлагдахад Run буцна.
	stop()
	time.Sleep(300 * time.Millisecond)
}
