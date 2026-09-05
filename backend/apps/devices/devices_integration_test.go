package devices_test

// Жишээ модулийн бүтэн урсгал: CRUD, "өөрийн" scope-ийн шүүлт, tenant
// тусгаарлалт, validation. Модуль бичигчдэд загвар болно.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/apps/devices"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	coredb "github.com/gerege-systems/nexus-mini/backend/internal/core/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/migrate"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fx struct {
	router        http.Handler
	owner         *pgxpool.Pool
	tid           string
	adminU, userU string
}

func setup(t *testing.T) *fx {
	t.Helper()
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
	// Модулийн хүснэгтүүд — бинарийн migrate-ээс хамаарахгүй, шинэ тест DB-д ч ажиллана.
	if err := migrate.Run(ownerURL, t.Logf, devices.New()); err != nil {
		t.Fatal(err)
	}
	f := &fx{owner: owner}
	clean := func() {
		_, _ = owner.Exec(ctx, `DELETE FROM devices WHERE tenant_id IN (SELECT id FROM tenants WHERE slug LIKE 'dtest%')`)
		_, _ = owner.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'dtest%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM users WHERE email LIKE 'dtest%'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		app.Close()
		owner.Close()
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('dtest','Д') RETURNING id`).Scan(&f.tid))
	must(owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('dtest-admin@x.mn','x','Админ') RETURNING id`).Scan(&f.adminU))
	must(owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('dtest-user@x.mn','x','Хэрэглэгч') RETURNING id`).Scan(&f.userU))
	for _, u := range []string{f.adminU, f.userU} {
		_, err := owner.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2)`, f.tid, u)
		must(err)
	}
	// Модулийн permission каталогт + role оноолт.
	m := devices.New()
	for _, p := range m.Permissions() {
		_, err := owner.Exec(ctx, `
			INSERT INTO permissions (code, module_id, name, own_scope) VALUES ($1,$2,$3,$4)
			ON CONFLICT (code) DO UPDATE SET own_scope = excluded.own_scope`, p.Code, m.ID(), p.Name, p.OwnScope)
		must(err)
	}
	tdb := coredb.NewTenantDB(app)
	perms := rbac.NewStore(tdb)
	deps := nexus.Deps{DB: tdb, Perms: perms, Audit: audit.NewRecorder(tdb)}

	r := chi.NewRouter()
	r.Route("/api/apps/devices", func(sub chi.Router) {
		// serve.go-той ижил: identity + tenant аль хэдийн тогтсон гэж үзнэ.
		sub.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				uid := req.Header.Get("X-Test-User")
				ctx := identity.With(req.Context(), f.tid, uid)
				ctx = nexus.WithIdentity(ctx, f.tid, uid)
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
		m.RegisterRoutes(sub, deps)
	})
	f.router = r
	return f
}

func (f *fx) grant(t *testing.T, roleCode, userID string, grants map[string]string) {
	t.Helper()
	ctx := context.Background()
	var roleID, memberID string
	if err := f.owner.QueryRow(ctx, `INSERT INTO roles (tenant_id, code, name) VALUES ($1,$2,$2) RETURNING id`, f.tid, roleCode).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	if err := f.owner.QueryRow(ctx, `SELECT id FROM memberships WHERE tenant_id = $1::uuid AND user_id = $2::uuid`, f.tid, userID).Scan(&memberID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.owner.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1,$2)`, memberID, roleID); err != nil {
		t.Fatal(err)
	}
	for code, scope := range grants {
		if _, err := f.owner.Exec(ctx, `INSERT INTO role_permissions (role_id, permission_code, scope) VALUES ($1,$2,$3)`, roleID, code, scope); err != nil {
			t.Fatal(err)
		}
	}
}

func (f *fx) req(t *testing.T, user, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, target, &buf)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Test-User", user)
	w := httptest.NewRecorder()
	f.router.ServeHTTP(w, r)
	return w
}

func TestDevicesCRUDAndOwnScope(t *testing.T) {
	f := setup(t)
	f.grant(t, "d_admin", f.adminU, map[string]string{"devices.read": "all", "devices.manage": "all"})
	f.grant(t, "d_user", f.userU, map[string]string{"devices.read": "own", "devices.manage": "own"})

	// Админ бүртгэнэ.
	w := f.req(t, f.adminU, http.MethodPost, "/api/apps/devices/", map[string]any{"name": "Dell", "serial": "S-1", "kind": "laptop"})
	if w.Code != http.StatusCreated {
		t.Fatalf("create = %d: %s", w.Code, w.Body.String())
	}
	var created map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	adminDevice := created["id"]

	// Хэрэглэгч өөрийнхөө бүртгэнэ.
	w = f.req(t, f.userU, http.MethodPost, "/api/apps/devices/", map[string]any{"name": "HP", "serial": "S-2"})
	if w.Code != http.StatusCreated {
		t.Fatalf("хэрэглэгчийн create = %d: %s", w.Code, w.Body.String())
	}
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	userDevice := created["id"]

	// Жагсаалт: админ 2, хэрэглэгч зөвхөн өөрийнх (own scope).
	count := func(user string) int {
		w := f.req(t, user, http.MethodGet, "/api/apps/devices/", nil)
		var out struct {
			Devices []map[string]any `json:"devices"`
			Scope   string           `json:"scope"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return len(out.Devices)
	}
	if n := count(f.adminU); n != 2 {
		t.Fatalf("админд харагдах = %d (2 хүлээсэн)", n)
	}
	if n := count(f.userU); n != 1 {
		t.Fatalf("own scope-той хэрэглэгчид = %d (1 хүлээсэн)", n)
	}
	// Хэрэглэгч бусдын мөрийг засаж/устгаж чадахгүй.
	if w := f.req(t, f.userU, http.MethodPut, "/api/apps/devices/"+adminDevice,
		map[string]any{"name": "Hack", "serial": "S-1"}); w.Code != http.StatusNotFound {
		t.Fatalf("бусдын мөр засах = %d (404 хүлээсэн)", w.Code)
	}
	if w := f.req(t, f.userU, http.MethodDelete, "/api/apps/devices/"+adminDevice, nil); w.Code != http.StatusNotFound {
		t.Fatalf("бусдын мөр устгах = %d (404 хүлээсэн)", w.Code)
	}
	// Өөрийнхөө засна.
	if w := f.req(t, f.userU, http.MethodPut, "/api/apps/devices/"+userDevice,
		map[string]any{"name": "HP шинэ", "serial": "S-2", "status": "repair"}); w.Code != 200 {
		t.Fatalf("өөрийн мөр засах = %d: %s", w.Code, w.Body.String())
	}
	// Validation: нэргүй, буруу статус, урт нэр, давхардсан сериал, буруу uuid.
	bad := []map[string]any{
		{"name": "", "serial": "S-3"},
		{"name": "X", "serial": ""},
		{"name": "X", "serial": "S-3", "status": "үл-мэдэх"},
		{"name": string(make([]byte, 200)), "serial": "S-4"},
	}
	for i, b := range bad {
		if w := f.req(t, f.adminU, http.MethodPost, "/api/apps/devices/", b); w.Code != http.StatusBadRequest {
			t.Errorf("буруу оролт #%d = %d (400 хүлээсэн)", i, w.Code)
		}
	}
	if w := f.req(t, f.adminU, http.MethodPost, "/api/apps/devices/", map[string]any{"name": "Дахин", "serial": "S-1"}); w.Code != http.StatusConflict {
		t.Fatalf("давхардсан сериал = %d (409 хүлээсэн)", w.Code)
	}
	if w := f.req(t, f.adminU, http.MethodDelete, "/api/apps/devices/тийм-биш", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу uuid = %d (400 хүлээсэн)", w.Code)
	}
	// Эрхгүй хэрэглэгч (role оноогоогүй) — 403.
	ctx := context.Background()
	var strangerID string
	if err := f.owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('dtest-none@x.mn','x','Гадны') RETURNING id`).Scan(&strangerID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.owner.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2)`, f.tid, strangerID); err != nil {
		t.Fatal(err)
	}
	if w := f.req(t, strangerID, http.MethodGet, "/api/apps/devices/", nil); w.Code != http.StatusForbidden {
		t.Fatalf("эрхгүй хэрэглэгч = %d (403 хүлээсэн)", w.Code)
	}
	// Устгах.
	if w := f.req(t, f.adminU, http.MethodDelete, "/api/apps/devices/"+adminDevice, nil); w.Code != 200 {
		t.Fatalf("устгах = %d", w.Code)
	}
	if n := count(f.adminU); n != 1 {
		t.Fatalf("устгасны дараа = %d", n)
	}
}
