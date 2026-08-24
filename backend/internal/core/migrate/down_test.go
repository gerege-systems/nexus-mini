package migrate

// Миграцын Down зам: цөмийн хамгийн сүүлийн миграцыг буцааж, дахин
// хэрэглэхэд алдаагүй байх (goose Down бичигдээгүй бол энд илэрнэ).

import (
	"context"
	"database/sql"
	"os"
	"testing"

	coredb "github.com/gerege-systems/nexus-mini/backend/db"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

func TestCoreMigrationsDownUpRoundTrip(t *testing.T) {
	ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL_OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	if os.Getenv("NEXUS_TEST_ALLOW_DOWN") == "" {
		t.Skip("Down тест өгөгдөл устгадаг — NEXUS_TEST_ALLOW_DOWN=1 өгвөл ажиллана")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var before int64
	if err := pool.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version`).Scan(&before); err != nil {
		t.Fatal(err)
	}
	sqlDB, err := sql.Open("pgx", ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	goose.SetBaseFS(coredb.Migrations)
	goose.SetTableName("goose_db_version")
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Down(sqlDB, "migrations"); err != nil {
		t.Fatalf("Down (%d → %d): %v", before, before-1, err)
	}
	if err := goose.Up(sqlDB, "migrations"); err != nil {
		t.Fatalf("Up буцаах: %v", err)
	}
	var after int64
	if err := pool.QueryRow(ctx, `SELECT max(version_id) FROM goose_db_version`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("хувилбар сэргээгдсэнгүй: %d → %d", before, after)
	}
}
