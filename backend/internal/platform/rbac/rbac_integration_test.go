package rbac

import (
	"context"
	"os"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/platform/db"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration — бодит PG шаардана (db package-ийн тесттэй ижил env).
func TestUserGrantsIntegration(t *testing.T) {
	appURL := os.Getenv("NEXUS_TEST_DATABASE_URL")
	ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if appURL == "" || ownerURL == "" {
		t.Skip("NEXUS_TEST_DATABASE_URL(_OWNER) тохируулаагүй — integration тест алгаслаа")
	}
	ctx := context.Background()
	oc, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = oc.Exec(ctx, `DELETE FROM permissions WHERE code LIKE 'rbactest.%'`)
		_, _ = oc.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'rbactest-%'`)
		_, _ = oc.Exec(ctx, `DELETE FROM users WHERE email LIKE 'rbactest-%'`)
		oc.Close(ctx)
	})

	// Seed: tenant, user (admin ⊃ manager ⊃ user гинжний manager role-тэй),
	// permissions: p1 user-т own, manager-т all; p2 зөвхөн user-т.
	var t1, t2, u1 string
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(oc.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('rbactest-a', 'A') RETURNING id`).Scan(&t1))
	must(oc.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('rbactest-b', 'B') RETURNING id`).Scan(&t2))
	must(oc.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('rbactest-u@x.mn', 'x', 'U') RETURNING id`).Scan(&u1))
	var memberID string
	must(oc.QueryRow(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1, $2) RETURNING id`, t1, u1).Scan(&memberID))

	roleIDs := map[string]string{}
	for _, r := range [][2]string{{"user", ""}, {"manager", "user"}} {
		var id string
		var implies any
		if r[1] != "" {
			implies = r[1]
		}
		must(oc.QueryRow(ctx,
			`INSERT INTO roles (tenant_id, code, name, implies) VALUES ($1, $2, $2, $3) RETURNING id`,
			t1, r[0], implies).Scan(&id))
		roleIDs[r[0]] = id
	}
	must2 := func(_ pgx.Rows, err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	_ = must2
	for _, p := range []string{"rbactest.p1", "rbactest.p2"} {
		if _, err := oc.Exec(ctx,
			`INSERT INTO permissions (code, module_id, name) VALUES ($1, 'rbactest', $1)`, p); err != nil {
			t.Fatal(err)
		}
	}
	grants := [][3]string{
		{"user", "rbactest.p1", "own"},
		{"manager", "rbactest.p1", "all"},
		{"user", "rbactest.p2", "own"},
	}
	for _, g := range grants {
		if _, err := oc.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_code, scope) VALUES ($1, $2, $3)`,
			roleIDs[g[0]], g[1], g[2]); err != nil {
			t.Fatal(err)
		}
	}
	// Хэрэглэгч зөвхөн manager role-тэй — user-ийн эрхийг implies-ээр өвлөнө.
	if _, err := oc.Exec(ctx,
		`INSERT INTO membership_roles (membership_id, role_id) VALUES ($1, $2)`,
		memberID, roleIDs["manager"]); err != nil {
		t.Fatal(err)
	}

	pool, err := pgxpool.New(ctx, appURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := NewStore(db.NewTenantDB(pool))

	got, err := store.UserGrants(ctx, t1, u1)
	if err != nil {
		t.Fatal(err)
	}
	// p1: manager(all) + user(own) → all ялна; p2: implies-ээр own ирнэ.
	if g := got["rbactest.p1"]; !g.Allowed || g.Scope != nexus.ScopeAll {
		t.Fatalf("p1: %+v (all байх ёстой — өргөн scope ялна)", g)
	}
	if g := got["rbactest.p2"]; !g.Allowed || g.Scope != nexus.ScopeOwn {
		t.Fatalf("p2: %+v (implies гинжээр own ирэх ёстой)", g)
	}

	// Invalidate: өөр tenant-ийг цэвэрлэхэд энэ tenant-ийн кэш үлдэнэ.
	store.Invalidate(t2)
	store.mu.Lock()
	_, cached := store.cache[t1+":"+u1]
	store.mu.Unlock()
	if !cached {
		t.Fatal("өөр tenant-ийн Invalidate энэ tenant-ийн кэшийг устгав")
	}
	store.Invalidate(t1)
	store.mu.Lock()
	_, cached = store.cache[t1+":"+u1]
	store.mu.Unlock()
	if cached {
		t.Fatal("Invalidate өөрийн tenant-ийн кэшийг устгасангүй")
	}
}
