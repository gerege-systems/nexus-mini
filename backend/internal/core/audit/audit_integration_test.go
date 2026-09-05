package audit_test

// Audit гинж: бичилт, hash chain, гар хүрэхээс хамгаалалт (append-only),
// impersonated тэмдэглэгээ.

import (
	"context"
	"os"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	coredb "github.com/gerege-systems/nexus-mini/backend/internal/core/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAuditChainAndAppendOnly(t *testing.T) {
	appURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if appURL == "" || ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL / _OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	app, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	clean := func() {
		_, _ = owner.Exec(ctx, `DELETE FROM audit_log WHERE tenant_id IN (SELECT id FROM tenants WHERE slug='autest')`)
		_, _ = owner.Exec(ctx, `DELETE FROM tenants WHERE slug = 'autest'`)
		_, _ = owner.Exec(ctx, `DELETE FROM users WHERE email = 'autest@x.mn'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		app.Close()
		owner.Close()
	})
	var tid, uid string
	if err := owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('autest','А') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('autest@x.mn','x','А') RETURNING id`).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2)`, tid, uid); err != nil {
		t.Fatal(err)
	}

	tdb := coredb.NewTenantDB(app)
	rec := audit.NewRecorder(tdb)
	base := identity.With(ctx, tid, uid)
	rec.RecordAs(base, tid, uid, "test.one", "объект-1", map[string]any{"a": 1})
	rec.RecordAs(base, tid, uid, "test.two", "объект-2", nil)
	// Impersonated context → details-д impersonated_by орно.
	imp := identity.WithImpersonator(base, uid)
	rec.RecordAs(imp, tid, uid, "test.three", "объект-3", nil)

	var n int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM audit_log WHERE tenant_id = $1::uuid`, tid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("бичлэгийн тоо = %d", n)
	}
	var impBy *string
	if err := owner.QueryRow(ctx, `SELECT details->>'impersonated_by' FROM audit_log WHERE tenant_id = $1::uuid AND action = 'test.three'`, tid).Scan(&impBy); err != nil {
		t.Fatal(err)
	}
	if impBy == nil || *impBy != uid {
		t.Fatalf("impersonated_by = %v", impBy)
	}
	// Гинж бүрэн.
	// audit_verify нь эвдэрсэн мөрийн id, эсвэл NULL (бүрэн) буцаана.
	// 00018-аас хойш зөвхөн өөрийн tenant-ийн context-оос (эсвэл платформ)
	// дуудагдана — апп role-оор, app.tenant_id тохируулсан гүйлгээнд.
	var brokenAt *int64
	verify := func(asTenant string) error {
		tx, err := app.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if _, err := tx.Exec(ctx, `SELECT set_config('app.tenant_id', $1, true)`, asTenant); err != nil {
			return err
		}
		return tx.QueryRow(ctx, `SELECT audit_verify($1::uuid)`, tid).Scan(&brokenAt)
	}
	if err := verify(tid); err != nil {
		t.Fatal(err)
	}
	if brokenAt != nil {
		t.Fatalf("гинж #%d дээр тасарсан", *brokenAt)
	}
	// Өөр tenant-ийн гинжийг шалгаж болохгүй.
	if err := verify("00000000-0000-0000-0000-000000000001"); err == nil {
		t.Fatal("өөр tenant-ийн audit_verify зөвшөөрөгдөв")
	}
	// Append-only: UPDATE/DELETE хориотой (owner-оор ч).
	if _, err := owner.Exec(ctx, `UPDATE audit_log SET action = 'hack' WHERE tenant_id = $1::uuid`, tid); err == nil {
		t.Fatal("audit_log засагдав")
	}
	if _, err := owner.Exec(ctx, `DELETE FROM audit_log WHERE tenant_id = $1::uuid AND action = 'test.one'`); err == nil {
		t.Fatal("audit_log-оос устгагдав")
	}
	// Цэвэрлэхийн тулд триггерийг түр хасна (тест хийсэн мөрөө авах).
	if _, err := owner.Exec(ctx, `ALTER TABLE audit_log DISABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
	_, _ = owner.Exec(ctx, `DELETE FROM audit_log WHERE tenant_id = $1::uuid`, tid)
	if _, err := owner.Exec(ctx, `ALTER TABLE audit_log ENABLE TRIGGER USER`); err != nil {
		t.Fatal(err)
	}
}
