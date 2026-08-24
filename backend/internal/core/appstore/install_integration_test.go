package appstore

// Апп суулгах бүтэн зам: хамаарлын дараалал, permission-ий default оноолт,
// суулгалтын үйл явдал, gate (суулгаагүй апп 403), enable/disable.

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	coredb "github.com/gerege-systems/nexus-mini/backend/internal/core/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// testModule — бодит модулийн оронд (миграцгүй) SDK-ийн гэрээг хангасан хоёр
// модуль: хамаарлын дарааллыг шалгахад хэрэгтэй.
type testModule struct {
	id, short string
	deps      []nexus.Dependency
	perms     []nexus.PermissionDefinition
}

func (m *testModule) ID() string                                { return m.id }
func (m *testModule) ShortID() string                           { return m.short }
func (m *testModule) Name() string                              { return m.short }
func (m *testModule) Version() string                           { return "1.0.0" }
func (m *testModule) Dependencies() []nexus.Dependency          { return m.deps }
func (m *testModule) Permissions() []nexus.PermissionDefinition { return m.perms }
func (m *testModule) Menus() []nexus.MenuDefinition             { return nil }
func (m *testModule) Migrations() fs.FS                         { return nil }
func (m *testModule) RegisterRoutes(r chi.Router, d nexus.Deps) {
	r.Get("/", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
}

func TestInstallGrantsAndGate(t *testing.T) {
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
		_, _ = owner.Exec(ctx, `DELETE FROM installation_events WHERE app_id LIKE 'mn.itest2.%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM app_installations WHERE app_id LIKE 'mn.itest2.%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM app_releases WHERE app_id LIKE 'mn.itest2.%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM role_permissions WHERE permission_code LIKE 'itest2%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM permissions WHERE code LIKE 'itest2%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM apps WHERE id LIKE 'mn.itest2.%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM tenants WHERE slug = 'itest2'`)
		_, _ = owner.Exec(ctx, `DELETE FROM users WHERE email = 'itest2@x.mn'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		app.Close()
		owner.Close()
	})

	base := &testModule{id: "mn.itest2.base", short: "itest2base",
		perms: []nexus.PermissionDefinition{{Code: "itest2base.read", Name: "харах", DefaultRoles: []string{"manager", "user"}}}}
	dep := &testModule{id: "mn.itest2.dep", short: "itest2dep",
		deps:  []nexus.Dependency{{ID: base.ID()}},
		perms: []nexus.PermissionDefinition{{Code: "itest2dep.manage", Name: "удирдах", OwnScope: true, DefaultRoles: []string{"manager", "user:own"}}}}
	nexus.Register(base)
	nexus.Register(dep)

	// Хамаарлын дараалал: base эхэлнэ.
	mods := map[string]nexus.Module{base.ID(): base, dep.ID(): dep}
	order, err := ResolveOrder(mods, dep)
	if err != nil || len(order) != 2 || order[0].ID() != base.ID() {
		t.Fatalf("ResolveOrder = %v %v", order, err)
	}

	// Fixtures: tenant + admin/manager/user role-ууд + permission каталог.
	var tid, uid string
	if err := owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('itest2','И') RETURNING id`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('itest2@x.mn','x','И') RETURNING id`).Scan(&uid); err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2)`, tid, uid); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"admin", "manager", "user"} {
		if _, err := owner.Exec(ctx, `INSERT INTO roles (tenant_id, code, name) VALUES ($1,$2,$2)`, tid, code); err != nil {
			t.Fatal(err)
		}
	}
	for _, m := range []*testModule{base, dep} {
		if _, err := owner.Exec(ctx, `INSERT INTO apps (id, short_id, name, version, compiled) VALUES ($1,$2,$2,'1.0.0',true)`, m.ID(), m.ShortID()); err != nil {
			t.Fatal(err)
		}
		for _, p := range m.Permissions() {
			if _, err := owner.Exec(ctx, `INSERT INTO permissions (code, module_id, name, own_scope) VALUES ($1,$2,$3,$4)`,
				p.Code, m.ID(), p.Name, p.OwnScope); err != nil {
				t.Fatal(err)
			}
		}
	}

	tdb := coredb.NewTenantDB(app)
	perms := rbac.NewStore(tdb)
	inst := NewInstaller(tdb, perms)
	tctx := identity.With(ctx, tid, uid)
	nctx := nexus.WithIdentity(tctx, tid, uid)

	if err := inst.Install(nctx, dep.ID()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	// Хоёулаа суусан (хамаарал автоматаар).
	var n int
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM app_installations WHERE tenant_id = $1::uuid`, tid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("суулгалтын тоо = %d", n)
	}
	// Default оноолт: admin бүгдийг all, manager/user тунхагласнаар.
	type g struct{ role, code, scope string }
	rows, err := owner.Query(ctx, `
		SELECT r.code, rp.permission_code, rp.scope FROM role_permissions rp
		  JOIN roles r ON r.id = rp.role_id
		 WHERE r.tenant_id = $1::uuid AND rp.permission_code LIKE 'itest2%' ORDER BY 1,2`, tid)
	if err != nil {
		t.Fatal(err)
	}
	got := map[g]bool{}
	for rows.Next() {
		var x g
		if err := rows.Scan(&x.role, &x.code, &x.scope); err != nil {
			t.Fatal(err)
		}
		got[x] = true
	}
	rows.Close()
	for _, want := range []g{
		{"admin", "itest2base.read", "all"}, {"admin", "itest2dep.manage", "all"},
		{"manager", "itest2base.read", "all"}, {"manager", "itest2dep.manage", "all"},
		{"user", "itest2base.read", "all"}, {"user", "itest2dep.manage", "own"},
	} {
		if !got[want] {
			t.Errorf("оноолт алга: %+v", want)
		}
	}
	if got[g{"user", "itest2base.read", "own"}] {
		t.Error("own_scope биш permission-д own оноогдов")
	}
	// Суулгалтын үйл явдал бичигдсэн.
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM installation_events WHERE tenant_id = $1::uuid AND action = 'install'`, tid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("install үйл явдал = %d", n)
	}

	// Gate: суусан апп нээлттэй, суулгаагүй нь 403.
	gate := NewGate(tdb)
	handler := func(appID string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(nexus.WithIdentity(identity.With(r.Context(), tid, uid), tid, uid))
		w := httptest.NewRecorder()
		gate.Middleware(appID)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})).ServeHTTP(w, r)
		return w
	}
	if w := handler(dep.ID()); w.Code != http.StatusOK {
		t.Fatalf("суусан апп gate = %d", w.Code)
	}
	if w := handler("mn.itest2.байхгүй"); w.Code != http.StatusForbidden {
		t.Fatalf("суулгаагүй апп gate = %d (403 хүлээсэн)", w.Code)
	}
	// Унтраах → gate хаана.
	if err := inst.SetStatus(nctx, dep.ID(), "disabled"); err != nil {
		t.Fatal(err)
	}
	gate.InvalidateLocal(tid)
	if w := handler(dep.ID()); w.Code != http.StatusForbidden {
		t.Fatalf("унтраасан апп gate = %d (403 хүлээсэн)", w.Code)
	}
	// Асаах → буцаж нээгдэнэ, үйл явдал бичигдэнэ.
	if err := inst.SetStatus(nctx, dep.ID(), "enabled"); err != nil {
		t.Fatal(err)
	}
	gate.InvalidateLocal(tid)
	if w := handler(dep.ID()); w.Code != http.StatusOK {
		t.Fatalf("асаасан апп gate = %d", w.Code)
	}
	if err := owner.QueryRow(ctx, `SELECT count(*) FROM installation_events WHERE tenant_id = $1::uuid AND action IN ('enable','disable')`, tid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("enable/disable үйл явдал = %d", n)
	}
	// Байхгүй апп.
	if err := inst.Install(nctx, "mn.itest2.байхгүй"); err != ErrNotFound {
		t.Fatalf("байхгүй апп = %v", err)
	}
	if err := inst.SetStatus(nctx, "mn.itest2.байхгүй", "disabled"); err != ErrNotFound {
		t.Fatalf("байхгүй суулгалт = %v", err)
	}
}
