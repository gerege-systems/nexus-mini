package db

// TenantDB: GUC-ууд гүйлгээ бүрд тохирч, буцаж цэвэрлэгдэнэ; identity-гүй
// хүсэлт юу ч харахгүй (RLS fail-closed).

import (
	"context"
	"os"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestTenantDBSetsAndResetsGUCs(t *testing.T) {
	appURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if appURL == "" || ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL / _OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}

	owner, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}

	clean := func() { _, _ = owner.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'gtest%'`) }
	clean()
	t.Cleanup(func() {
		clean() // цэвэрлэлт pool хаагдахаас ӨМНӨ (defer Close нь Cleanup-ээс өмнө ажилладаг)
		pool.Close()
		owner.Close()
	})
	var tid, uid string
	if err := owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('gtest','Г') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&uid); err != nil {
		t.Fatal(err)
	}

	tdb := NewTenantDB(pool)
	// identity-тэй: GUC-ууд тохирно.
	var gotTenant, gotUser string
	if err := tdb.QueryRow(identity.With(ctx, tid, uid),
		`SELECT coalesce(current_setting('app.tenant_id', true), ''), coalesce(current_setting('app.user_id', true), '')`).
		Scan(&gotTenant, &gotUser); err != nil {
		t.Fatal(err)
	}
	if gotTenant != tid || gotUser != uid {
		t.Fatalf("GUC = %q / %q", gotTenant, gotUser)
	}
	// identity-гүй: GUC хоосон → RLS-ээр юу ч харагдахгүй.
	var n int
	if err := tdb.QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("identity-гүй үед %d tenant харагдав (0 байх ёстой)", n)
	}
	// Гүйлгээ: доторх бичилт RLS-д захирагдана, дараа нь GUC цэвэрлэгдэнэ.
	err = tdb.Tx(identity.With(ctx, tid, uid), func(tx pgx.Tx) error {
		var inTx string
		if err := tx.QueryRow(ctx, `SELECT current_setting('app.tenant_id', true)`).Scan(&inTx); err != nil {
			return err
		}
		if inTx != tid {
			t.Fatalf("гүйлгээн доторх GUC = %q", inTx)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := tdb.QueryRow(ctx, `SELECT count(*) FROM tenants`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("гүйлгээний дараа GUC үлдсэн (холболт бохирдсон)")
	}
	// Гүйлгээ алдаа буцаавал rollback.
	sentinel := context.Canceled
	err = tdb.Tx(identity.With(ctx, tid, uid), func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `INSERT INTO tenants (slug, name) VALUES ('gtest-rollback','Р')`); err != nil {
			return err
		}
		return sentinel
	})
	if err != sentinel {
		t.Fatalf("Tx алдаа = %v", err)
	}
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM tenants WHERE slug = 'gtest-rollback'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("rollback ажиллаагүй")
	}
}
