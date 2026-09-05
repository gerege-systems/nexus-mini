package handlers_test

// Handler-үүдийн integration харнесс: бодит DB, бодит session cookie, бодит
// permission middleware (serve.go-той ижил утга). Тест бүр өөрийн tenant
// үүсгэж, төгсгөлд нь цэвэрлэнэ.

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/appstore"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	coredb "github.com/gerege-systems/nexus-mini/backend/internal/core/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/handlers"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/ssoclient"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/tenantstate"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type harness struct {
	router                   http.Handler
	app, authP, admin, owner *pgxpool.Pool
	svc                      *auth.Service
	perms                    *rbac.Store
	state                    *tenantstate.Store
	ssoClient                *ssoclient.Client
	sso                      *handlers.SSO
	setup                    *handlers.Setup
	authHandler              *handlers.Auth
	prefix                   string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	appURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	authURL, adminURL := os.Getenv("NEXUS_TEST_DATABASE_URL_AUTH"), os.Getenv("NEXUS_TEST_DATABASE_URL_ADMIN")
	if appURL == "" || ownerURL == "" || authURL == "" || adminURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL / _OWNER / _AUTH / _ADMIN шаардлагатай")
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
	h := &harness{app: open(appURL), authP: open(authURL), admin: open(adminURL), owner: open(ownerURL), prefix: "htest-"}
	clean := func() {
		_, _ = h.owner.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'htest-%'`)
		_, _ = h.owner.Exec(ctx, `DELETE FROM users WHERE email LIKE 'htest-%'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		h.app.Close()
		h.authP.Close()
		h.admin.Close()
		h.owner.Close()
	})

	tdb := coredb.NewTenantDB(h.app)
	h.svc = auth.NewService(h.authP, false)
	h.perms = rbac.NewStore(tdb)
	h.state = tenantstate.New(h.authP)
	h.svc.SetTenantState(func(ctx context.Context, tid string) (bool, bool, error) {
		st, err := h.state.Get(ctx, tid)
		return st.Suspended, st.ReadOnly, err
	})
	rec := audit.NewRecorder(tdb)
	installer := appstore.NewInstaller(tdb, h.perms)
	gate := appstore.NewGate(tdb)
	authH := &handlers.Auth{Pool: h.authP, DB: tdb, Svc: h.svc, Audit: rec, Perms: h.perms, State: h.state, Issuer: "https://portal.mn/api/oauth2"}
	rbacH := &handlers.RBACH{DB: tdb, Pool: h.authP, Perms: h.perms, Audit: rec}
	adminH := &handlers.Admin{Pool: h.admin, Svc: h.svc, Rec: rec, State: h.state, PortalURL: "https://portal.mn"}
	storeH := &handlers.Store{DB: tdb, Installer: installer, Gate: gate, Audit: rec}
	miscH := &handlers.Misc{DB: tdb, Perms: h.perms}
	// SSO (relying party) — тестэд provider-ийг тест бүр өөрөө тохируулна.
	h.ssoClient = ssoclient.New(nil)
	ssoH := handlers.NewSSO(h.ssoClient, authH, "https://portal.mn", false, false, "тест-нууц")
	h.sso = ssoH
	h.authHandler = authH

	r := chi.NewRouter()
	r.Post("/api/signup", authH.Signup)
	h.setup = handlers.NewSetup(authH, "https://portal.mn")
	r.Get("/api/setup/status", h.setup.Status)
	r.Post("/api/setup/complete", h.setup.Complete)
	r.Post("/api/login", authH.Login)
	r.Post("/api/logout", authH.Logout)
	r.Post("/api/auth/handover", authH.Handover)
	r.Get("/api/auth/sso/providers", ssoH.Providers)
	r.Get("/api/auth/sso/{key}/start", ssoH.Start)
	r.Get("/api/auth/sso/{key}/callback", ssoH.Callback)
	r.Group(func(g chi.Router) {
		g.Use(h.svc.RequireUser)
		g.Get("/api/me", authH.Me)
		g.Put("/api/me", authH.UpdateProfile)
		g.Post("/api/me/password", authH.ChangePassword)
		g.Post("/api/session/tenant", authH.SelectTenant)
		g.Post("/api/tenants", authH.CreateTenant)
	})
	r.Group(func(g chi.Router) {
		g.Use(h.svc.RequireTenant)
		g.Get("/api/menu", miscH.Menu)
		g.With(nexus.RequirePermission(h.perms, "core.audit.read")).Get("/api/audit", miscH.Audit)
		g.With(nexus.RequirePermission(h.perms, "core.audit.read")).Get("/api/audit/verify", miscH.AuditVerify)
		g.Get("/api/tenant/profile", authH.TenantProfile)
		g.With(nexus.RequirePermission(h.perms, "core.settings.manage")).Put("/api/tenant/profile", authH.UpdateTenantProfile)
		g.With(nexus.RequirePermission(h.perms, "core.sso.manage")).Get("/api/sso-clients", authH.SSOClients)
		g.With(nexus.RequirePermission(h.perms, "core.sso.manage")).Post("/api/sso-clients", authH.CreateSSOClient)
		g.With(nexus.RequirePermission(h.perms, "core.sso.manage")).Put("/api/sso-clients/{id}", authH.UpdateSSOClient)
		g.With(nexus.RequirePermission(h.perms, "core.sso.manage")).Delete("/api/sso-clients/{id}", authH.DeleteSSOClient)
		g.With(nexus.RequirePermission(h.perms, "core.members.manage")).Get("/api/members", rbacH.Members)
		g.With(nexus.RequirePermission(h.perms, "core.members.manage")).Get("/api/members/lookup", rbacH.LookupMember)
		g.With(nexus.RequirePermission(h.perms, "core.members.manage")).Post("/api/members", rbacH.AddMember)
		g.With(nexus.RequirePermission(h.perms, "core.members.manage")).Put("/api/members/{id}/roles", rbacH.SetMemberRoles)
		g.With(nexus.RequirePermission(h.perms, "core.members.manage")).Delete("/api/members/{id}", rbacH.RemoveMember)
		g.Get("/api/permissions", rbacH.Permissions)
		g.With(nexus.RequirePermission(h.perms, "core.roles.manage")).Get("/api/roles", rbacH.Roles)
		g.With(nexus.RequirePermission(h.perms, "core.roles.manage")).Post("/api/roles", rbacH.CreateRole)
		g.With(nexus.RequirePermission(h.perms, "core.roles.manage")).Put("/api/roles/{id}/grants", rbacH.SetGrants)
		g.Get("/api/store/apps", storeH.List)
		g.Get("/api/store/apps/{id}/history", storeH.History)
		g.With(nexus.RequirePermission(h.perms, "core.apps.manage")).Post("/api/store/apps/{id}/install", storeH.Install)
		g.With(nexus.RequirePermission(h.perms, "core.apps.manage")).Post("/api/store/apps/{id}/disable", storeH.SetStatus("disabled"))
		g.With(nexus.RequirePermission(h.perms, "core.apps.manage")).Post("/api/store/apps/{id}/enable", storeH.SetStatus("enabled"))
	})
	r.Group(func(g chi.Router) {
		g.Use(h.svc.RequirePlatformAdmin)
		g.Get("/api/admin/overview", adminH.Overview)
		g.Get("/api/admin/users", adminH.Users)
		g.Get("/api/admin/apps", adminH.Apps)
		g.Get("/api/admin/audit", adminH.Audit)
		g.Get("/api/admin/tenants", adminH.Tenants)
		g.Get("/api/admin/tenants/{id}/members", adminH.TenantMembers)
		g.Put("/api/admin/tenants/{id}/state", adminH.SetTenantState)
		g.Post("/api/admin/tenants/{id}/delete", adminH.ScheduleDeletion)
		g.Post("/api/admin/tenants/{id}/delete/cancel", adminH.CancelDeletion)
		g.Post("/api/admin/impersonate", adminH.Impersonate)
	})
	h.router = r
	return h
}

// session — нэвтэрсэн хэрэглэгчийн cookie агуулагч.
type session struct {
	h      *harness
	cookie string
	userID string
}

// signup — шинэ хэрэглэгч + байгууллага үүсгэж session буцаана.
func (h *harness) signup(t *testing.T, slug string) *session {
	t.Helper()
	body := map[string]string{"name": "Т " + slug, "email": "htest-" + slug + "@x.mn", "password": "password-12",
		"tenant_name": "Т " + slug, "tenant_slug": "htest-" + slug}
	w := h.do(t, nil, http.MethodPost, "/api/signup", body)
	if w.Code >= 400 {
		t.Fatalf("signup(%s) = %d: %s", slug, w.Code, w.Body.String())
	}
	var out struct {
		UserID string `json:"user_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	s := &session{h: h, userID: out.UserID}
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			s.cookie = c.Value
		}
	}
	if s.cookie == "" {
		t.Fatal("signup-д cookie алга")
	}
	return s
}

// login — байгаа хэрэглэгчээр нэвтэрч (tenant сонгож) session буцаана.
func (h *harness) login(t *testing.T, email, password, tenantID string) *session {
	t.Helper()
	w := h.do(t, nil, http.MethodPost, "/api/login", map[string]string{"email": email, "password": password})
	if w.Code != 200 {
		t.Fatalf("login(%s) = %d: %s", email, w.Code, w.Body.String())
	}
	s := &session{h: h}
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			s.cookie = c.Value
		}
	}
	if tenantID != "" {
		if w := s.do(t, http.MethodPost, "/api/session/tenant", map[string]string{"tenant_id": tenantID}); w.Code != 200 {
			t.Fatalf("tenant сонголт = %d: %s", w.Code, w.Body.String())
		}
	}
	me := s.json(t, http.MethodGet, "/api/me", nil)
	if u, ok := me["user"].(map[string]any); ok {
		s.userID, _ = u["id"].(string)
	}
	return s
}

func (h *harness) do(t *testing.T, cookie *string, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = bytes.NewBuffer(nil)
	}
	r := httptest.NewRequest(method, target, buf)
	r.Header.Set("Content-Type", "application/json")
	if cookie != nil && *cookie != "" {
		r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: *cookie})
	}
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, r)
	return w
}

func (s *session) do(t *testing.T, method, target string, body any) *httptest.ResponseRecorder {
	t.Helper()
	return s.h.do(t, &s.cookie, method, target, body)
}

func (s *session) json(t *testing.T, method, target string, body any) map[string]any {
	t.Helper()
	w := s.do(t, method, target, body)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

// ssoStart — SSO нэвтрэлт эхлүүлж (state, cookie) буцаана.
func (h *harness) ssoStart(t *testing.T, idp *fakeIDP) (state string, cookie *http.Cookie) {
	t.Helper()
	w := h.do(t, nil, http.MethodGet, "/api/auth/sso/sso/start", nil)
	if w.Code != http.StatusFound {
		t.Fatalf("sso start = %d", w.Code)
	}
	u, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	idp.nonce = u.Query().Get("nonce")
	for _, c := range w.Result().Cookies() {
		if c.Name == "nexus_sso" {
			cookie = c
		}
	}
	return u.Query().Get("state"), cookie
}

func (h *harness) ssoCallback(t *testing.T, cookie *http.Cookie, query string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, "/api/auth/sso/sso/callback?"+query, nil)
	if cookie != nil {
		r.AddCookie(cookie)
	}
	return recordRequest(h, r)
}

// formPost — x-www-form-urlencoded хүсэлт (handover гэх мэт).
func formPost(t *testing.T, target, body string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func recordRequest(h *harness, r *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.router.ServeHTTP(w, r)
	return w
}

// tenantID — session-ий идэвхтэй байгууллага.
func (s *session) tenantID(t *testing.T) string {
	t.Helper()
	me := s.json(t, http.MethodGet, "/api/me", nil)
	id, _ := me["tenant_id"].(string)
	if id == "" {
		t.Fatal("tenant сонгогдоогүй")
	}
	return id
}

// makePlatformAdmin — тухайн хэрэглэгчийг платформын админ болгоно.
func (h *harness) makePlatformAdmin(t *testing.T, userID string) {
	t.Helper()
	if _, err := h.owner.Exec(context.Background(), `UPDATE users SET platform_admin = true WHERE id = $1::uuid`, userID); err != nil {
		t.Fatal(err)
	}
}
