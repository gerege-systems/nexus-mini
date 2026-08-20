package db

import (
	"context"
	"os"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration тест — бодит Postgres шаардана (миграц хийгдсэн байх):
//
//	NEXUS_TEST_DATABASE_URL=postgres://nexus_app:...  \
//	NEXUS_TEST_DATABASE_URL_OWNER=postgres://nexus_owner:...  go test ./internal/core/db/
func testEnv(t *testing.T) (appURL, ownerURL string) {
	t.Helper()
	appURL = os.Getenv("NEXUS_TEST_DATABASE_URL")
	ownerURL = os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if appURL == "" || ownerURL == "" {
		t.Skip("NEXUS_TEST_DATABASE_URL(_OWNER) тохируулаагүй — integration тест алгаслаа")
	}
	return
}

// seedTwoTenants — owner-ээр хоёр tenant + T1-д гишүүн хэрэглэгч, role
// үүсгээд цэвэрлэгээг t.Cleanup-д бүртгэнэ. Бүх мөр 'rlstest-' prefix-тэй.
func seedTwoTenants(t *testing.T, ownerURL string) (t1, t2, u1 string) {
	t.Helper()
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = conn.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'rlstest-%'`)
		_, _ = conn.Exec(ctx, `DELETE FROM users WHERE email LIKE 'rlstest-%'`)
		conn.Close(ctx)
	})
	if err := conn.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ('rlstest-a', 'A') RETURNING id`).Scan(&t1); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx,
		`INSERT INTO tenants (slug, name) VALUES ('rlstest-b', 'B') RETURNING id`).Scan(&t2); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name) VALUES ('rlstest-u1@x.mn', 'x', 'U1') RETURNING id`).Scan(&u1); err != nil {
		t.Fatal(err)
	}
	if _, err := conn.Exec(ctx,
		`INSERT INTO memberships (tenant_id, user_id) VALUES ($1, $2)`, t1, u1); err != nil {
		t.Fatal(err)
	}
	for _, tid := range []string{t1, t2} {
		if _, err := conn.Exec(ctx,
			`INSERT INTO roles (tenant_id, code, name) VALUES ($1, 'rlstest_role', 'x')`, tid); err != nil {
			t.Fatal(err)
		}
	}
	return
}

func TestRLSIsolation(t *testing.T) {
	appURL, ownerURL := testEnv(t)
	t1, t2, u1 := seedTwoTenants(t, ownerURL)

	pool, err := pgxpool.New(context.Background(), appURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tdb := NewTenantDB(pool)

	// T1-ийн context-оос: өөрийн role харагдана, T2-ийнх огт үгүй.
	ctx := identity.With(context.Background(), t1, u1)
	var n int
	if err := tdb.QueryRow(ctx,
		`SELECT count(*) FROM roles WHERE code = 'rlstest_role'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("T1 нь %d role харав (1 байх ёстой — зөвхөн өөрийнх)", n)
	}
	if err := tdb.QueryRow(ctx,
		`SELECT count(*) FROM roles WHERE tenant_id = $1::uuid`, t2).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("T1 нь T2-ийн %d role харав — RLS цоорхой!", n)
	}
	// tenants: гишүүнчлэлгүй tenant харагдахгүй.
	if err := tdb.QueryRow(ctx,
		`SELECT count(*) FROM tenants WHERE id = $1::uuid`, t2).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("T1-ийн хэрэглэгч T2 tenant-ийг харав")
	}
}

func TestGUCResetAfterRelease(t *testing.T) {
	appURL, ownerURL := testEnv(t)
	t1, _, u1 := seedTwoTenants(t, ownerURL)

	// Нэг холболттой pool — release-ийн дараах ижил холболтыг шалгана.
	cfg, err := pgxpool.ParseConfig(appURL)
	if err != nil {
		t.Fatal(err)
	}
	cfg.MaxConns = 1
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	tdb := NewTenantDB(pool)

	ctx := identity.With(context.Background(), t1, u1)
	var one int
	if err := tdb.QueryRow(ctx, `SELECT 1`).Scan(&one); err != nil {
		t.Fatal(err)
	}

	// Одоо identity-гүй raw acquire: GUC хоосон байх ЁСТОЙ.
	conn, err := pool.Acquire(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	var guc string
	if err := conn.QueryRow(context.Background(),
		`SELECT coalesce(current_setting('app.tenant_id', true), '')`).Scan(&guc); err != nil {
		t.Fatal(err)
	}
	if guc != "" {
		t.Fatalf("release-ийн дараа GUC цэвэрлэгдээгүй: %q", guc)
	}
}
