package migrate

// Миграц: давтан ажиллуулахад алдаагүй (idempotent) бөгөөд цөмийн болон
// модулиудын хувилбар тэмдэглэгдсэн байна.

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/apps/devices"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunAllIsIdempotent(t *testing.T) {
	ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL_OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	var logged []string
	if err := Run(ownerURL, func(f string, a ...any) { logged = append(logged, f) }, devices.New()); err != nil {
		t.Fatalf("нэг дэх ажиллалт: %v", err)
	}
	// Хоёр дахь удаа — өөрчлөлтгүй, алдаагүй.
	if err := Run(ownerURL, func(f string, a ...any) {}, devices.New()); err != nil {
		t.Fatalf("хоёр дахь ажиллалт: %v", err)
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var version int64
	if err := pool.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version < 15 {
		t.Fatalf("цөмийн миграцын хувилбар = %d", version)
	}
	// Модуль бүр өөрийн goose хүснэгттэй (цөм + devices).
	var tables int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM information_schema.tables WHERE table_name LIKE 'goose_%'`).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables < 2 {
		t.Fatalf("goose хүснэгтийн тоо = %d (цөм + модулиуд)", tables)
	}
	if len(logged) == 0 || !strings.Contains(strings.Join(logged, " "), "миграц") {
		t.Fatalf("лог = %v", logged)
	}
}

func TestRunAllRejectsBadURL(t *testing.T) {
	if err := RunAll("postgres://байхгүй:5432/x?connect_timeout=1", func(string, ...any) {}); err == nil {
		t.Fatal("буруу URL дээр алдаа гарсангүй")
	}
}
