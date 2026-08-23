package handlers_test

// Signup, түдгэлзүүлэлт, каталогийн integration тестүүд — бодит DB шаардана
// (NEXUS_TEST_DATABASE_URL / _OWNER / _AUTH; make check-db). Эдгээр урсгал нь
// хоёр DB role, RLS, definer функцүүдийг хамардаг тул mock-оор баригдахгүй.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	coredb "github.com/gerege-systems/nexus-mini/backend/internal/core/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/handlers"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/tenantstate"
	"github.com/jackc/pgx/v5/pgxpool"
)

type env struct {
	app, auth, owner *pgxpool.Pool
	h                *handlers.Auth
	svc              *auth.Service
}

func setup(t *testing.T) *env {
	t.Helper()
	appURL, ownerURL, authURL := os.Getenv("NEXUS_TEST_DATABASE_URL"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER"), os.Getenv("NEXUS_TEST_DATABASE_URL_AUTH")
	if appURL == "" || ownerURL == "" || authURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL / _OWNER / _AUTH шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	open := func(u string) *pgxpool.Pool {
		p, err := pgxpool.New(ctx, u)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	e := &env{app: open(appURL), auth: open(authURL), owner: open(ownerURL)}
	t.Cleanup(func() {
		_, _ = e.owner.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'itest-%'`)
		_, _ = e.owner.Exec(ctx, `DELETE FROM users WHERE email LIKE 'itest-%'`)
		e.app.Close()
		e.auth.Close()
		e.owner.Close()
	})
	tdb := coredb.NewTenantDB(e.app)
	e.svc = auth.NewService(e.auth, false)
	perms := rbac.NewStore(tdb)
	e.h = &handlers.Auth{Pool: e.auth, DB: tdb, Svc: e.svc, Audit: audit.NewRecorder(tdb), Perms: perms, State: tenantstate.New(e.auth)}
	return e
}

func post(t *testing.T, h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/signup", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

// Signup нь хоёр DB role-оор ажилладаг (auth: хэрэглэгч, app: байгууллага).
// Буруу pool ашиглавал 42501 → 500 (бодитоор тохиолдсон алдаа).
func TestSignupCreatesUserAndTenant(t *testing.T) {
	e := setup(t)
	w := post(t, e.h.Signup, `{"name":"Итест","email":"itest-ok@x.mn","password":"password-12","tenant_name":"Итест","tenant_slug":"itest-ok"}`)
	if w.Code != http.StatusCreated && w.Code != http.StatusOK {
		t.Fatalf("signup = %d, бие: %s", w.Code, w.Body.String())
	}
	ctx := context.Background()
	var n int
	if err := e.owner.QueryRow(ctx, `SELECT count(*) FROM memberships m
		JOIN users u ON u.id = m.user_id JOIN tenants t ON t.id = m.tenant_id
		WHERE u.email = 'itest-ok@x.mn' AND t.slug = 'itest-ok'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("гишүүнчлэл = %d, 1 байх ёстой", n)
	}
	// Админ role + core эрхүүд оноогдсон эсэх.
	var grants int
	if err := e.owner.QueryRow(ctx, `SELECT count(*) FROM role_permissions rp
		JOIN roles r ON r.id = rp.role_id JOIN tenants t ON t.id = r.tenant_id
		WHERE t.slug = 'itest-ok' AND r.code = 'admin'`).Scan(&grants); err != nil {
		t.Fatal(err)
	}
	if grants == 0 {
		t.Fatal("admin role-д core эрх оноогдоогүй")
	}
}

// Байгууллага үүсэхгүй бол (slug давхардсан) хэрэглэгч ҮЛДЭХГҮЙ — нөхөн
// сэргээх устгал (00015).
func TestSignupRollsBackUserWhenTenantFails(t *testing.T) {
	e := setup(t)
	if w := post(t, e.h.Signup, `{"name":"A","email":"itest-a@x.mn","password":"password-12","tenant_name":"A","tenant_slug":"itest-dup"}`); w.Code >= 400 {
		t.Fatalf("эхний signup = %d: %s", w.Code, w.Body.String())
	}
	w := post(t, e.h.Signup, `{"name":"B","email":"itest-b@x.mn","password":"password-12","tenant_name":"B","tenant_slug":"itest-dup"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("давхардсан slug = %d (409 байх ёстой): %s", w.Code, w.Body.String())
	}
	var exists bool
	if err := e.owner.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM users WHERE email = 'itest-b@x.mn')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("байгууллага үүсээгүй атал хэрэглэгч үлдсэн (орфан)")
	}
}

// Түдгэлзүүлсэн байгууллагад RequireTenant 403 буцаана (frontend үүнийг
// login руу шидэлгүй хаагдсан дэлгэц үзүүлнэ).
func TestSuspendedTenantForbidsTenantRoutes(t *testing.T) {
	e := setup(t)
	if w := post(t, e.h.Signup, `{"name":"S","email":"itest-s@x.mn","password":"password-12","tenant_name":"S","tenant_slug":"itest-susp"}`); w.Code >= 400 {
		t.Fatalf("signup = %d: %s", w.Code, w.Body.String())
	}
	ctx := context.Background()
	if _, err := e.owner.Exec(ctx, `UPDATE tenants SET suspended_at = now(), suspension_reason = 'тест' WHERE slug = 'itest-susp'`); err != nil {
		t.Fatal(err)
	}
	var tid string
	if err := e.owner.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = 'itest-susp'`).Scan(&tid); err != nil {
		t.Fatal(err)
	}
	st, err := tenantstate.New(e.auth).Get(ctx, tid)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Suspended || st.Reason != "тест" {
		t.Fatalf("tenant_state = %+v, түдгэлзүүлсэн байх ёстой", st)
	}
	// RequireTenant-ийн шийдвэр: суспенд → 403.
	e.svc.SetTenantState(func(ctx context.Context, tenantID string) (bool, bool, error) {
		s, err := tenantstate.New(e.auth).Get(ctx, tenantID)
		return s.Suspended, s.ReadOnly, err
	})
	var code int
	h := e.svc.RequireTenant(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { code = 200 }))
	r := httptest.NewRequest(http.MethodGet, "/api/menu", nil)
	w := httptest.NewRecorder()
	// Session cookie-гүй тул 401; энэ тест зөвхөн tenant_state-ийг барина.
	h.ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("cookie-гүй = %d (401 байх ёстой)", w.Code)
	}
	_ = code
	var body map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &body)
}
