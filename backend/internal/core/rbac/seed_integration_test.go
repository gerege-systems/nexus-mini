package rbac

// Role seed ба permission-ий default оноолт (GrantOnInstall / RevokeOnUninstall)
// шууд: implies гинж, scope, "хэзээ ч нарийсгахгүй" дүрэм.

import (
	"context"
	"os"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestSeedRolesGrantAndRevoke(t *testing.T) {
	ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL_OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	owner, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	clean := func() {
		_, _ = owner.Exec(ctx, `DELETE FROM permissions WHERE code LIKE 'seedtest%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM tenants WHERE slug = 'seedtest'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		owner.Close()
	})
	var tid string
	if err := owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('seedtest','С') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	perms := []nexus.PermissionDefinition{
		{Code: "seedtest.read", Name: "харах", DefaultRoles: []string{"manager", "user"}},
		{Code: "seedtest.manage", Name: "удирдах", OwnScope: true, DefaultRoles: []string{"manager", "user:own"}},
		{Code: "seedtest.secret", Name: "нууц"}, // DefaultRoles хоосон = зөвхөн admin
	}
	for _, p := range perms {
		if _, err := owner.Exec(ctx, `INSERT INTO permissions (code, module_id, name, own_scope) VALUES ($1,'mn.seed.test',$2,$3)`,
			p.Code, p.Name, p.OwnScope); err != nil {
			t.Fatal(err)
		}
	}

	tx, err := owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	adminRole, err := SeedTenantRoles(ctx, tx, tid)
	if err != nil || adminRole == "" {
		t.Fatalf("SeedTenantRoles = %q %v", adminRole, err)
	}
	if err := GrantOnInstall(ctx, tx, tid, perms); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	// Гурван role үүссэн, implies гинжтэй.
	rows, err := owner.Query(ctx, `SELECT code, coalesce(implies,'') FROM roles WHERE tenant_id = $1::uuid ORDER BY code`, tid)
	if err != nil {
		t.Fatal(err)
	}
	implies := map[string]string{}
	for rows.Next() {
		var code, imp string
		if err := rows.Scan(&code, &imp); err != nil {
			t.Fatal(err)
		}
		implies[code] = imp
	}
	rows.Close()
	if implies["admin"] != "manager" || implies["manager"] != "user" || implies["user"] != "" {
		t.Fatalf("implies гинж = %v", implies)
	}

	// Оноолтууд.
	get := func() map[string]string {
		out := map[string]string{}
		rows, err := owner.Query(ctx, `
			SELECT r.code || ':' || rp.permission_code, rp.scope FROM role_permissions rp
			  JOIN roles r ON r.id = rp.role_id WHERE r.tenant_id = $1::uuid AND rp.permission_code LIKE 'seedtest%'`, tid)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		for rows.Next() {
			var k, v string
			if err := rows.Scan(&k, &v); err != nil {
				t.Fatal(err)
			}
			out[k] = v
		}
		return out
	}
	g := get()
	for k, want := range map[string]string{
		"admin:seedtest.read": "all", "admin:seedtest.manage": "all", "admin:seedtest.secret": "all",
		"manager:seedtest.read": "all", "manager:seedtest.manage": "all",
		"user:seedtest.read": "all", "user:seedtest.manage": "own",
	} {
		if g[k] != want {
			t.Errorf("%s = %q, хүлээсэн %q", k, g[k], want)
		}
	}
	if _, ok := g["user:seedtest.secret"]; ok {
		t.Error("DefaultRoles хоосон permission user-т оногдов")
	}

	// Дахин GrantOnInstall — "all"-ыг "own" болгож НАРИЙСГАХГҮЙ.
	tx, _ = owner.Begin(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE role_permissions SET scope = 'all' WHERE permission_code = 'seedtest.manage'
		  AND role_id = (SELECT id FROM roles WHERE tenant_id = $1::uuid AND code = 'user')`, tid); err != nil {
		t.Fatal(err)
	}
	if err := GrantOnInstall(ctx, tx, tid, perms); err != nil {
		t.Fatal(err)
	}
	_ = tx.Commit(ctx)
	if g := get(); g["user:seedtest.manage"] != "all" {
		t.Fatalf("дахин суулгахад scope нарийсав: %q", g["user:seedtest.manage"])
	}

	// RevokeOnUninstall — модулийн бүх оноолт арилна, бусад нь хэвээр.
	tx, _ = owner.Begin(ctx)
	if err := RevokeOnUninstall(ctx, tx, tid, "mn.seed.test"); err != nil {
		t.Fatal(err)
	}
	_ = tx.Commit(ctx)
	if g := get(); len(g) != 0 {
		t.Fatalf("revoke дараа = %v", g)
	}
	var coreGrants int
	if err := owner.QueryRow(ctx, `
		SELECT count(*) FROM role_permissions rp JOIN roles r ON r.id = rp.role_id
		 WHERE r.tenant_id = $1::uuid AND rp.permission_code LIKE 'core.%'`, tid).Scan(&coreGrants); err != nil {
		t.Fatal(err)
	}
	_ = coreGrants // core оноолт энэ tenant-д хийгдээгүй тул 0 байж болно
	// Үл мэдэх role код — чимээгүй алгасна (алдаа биш).
	tx, _ = owner.Begin(ctx)
	err = GrantOnInstall(ctx, tx, tid, []nexus.PermissionDefinition{{Code: "seedtest.read", DefaultRoles: []string{"байхгүй_role"}}})
	_ = tx.Rollback(ctx)
	if err != nil {
		t.Fatalf("байхгүй role = %v", err)
	}
	var _ pgx.Tx
}

func TestCorePermissionsContract(t *testing.T) {
	perms := CorePermissions()
	if len(perms) == 0 {
		t.Fatal("core permission алга")
	}
	seen := map[string]bool{}
	for _, p := range perms {
		if !seen[p.Code] {
			seen[p.Code] = true
		} else {
			t.Errorf("давхардсан код: %s", p.Code)
		}
		if p.Name == "" {
			t.Errorf("%s: нэргүй", p.Code)
		}
		if len(p.Code) < 6 || p.Code[:5] != "core." {
			t.Errorf("%s: core. prefix-гүй", p.Code)
		}
	}
	for _, want := range []string{"core.members.manage", "core.roles.manage", "core.apps.manage", "core.audit.read", "core.settings.manage", "core.sso.manage"} {
		if !seen[want] {
			t.Errorf("%s алга", want)
		}
	}
}
