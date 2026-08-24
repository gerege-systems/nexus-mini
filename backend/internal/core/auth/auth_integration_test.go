package auth_test

// Session, дансны түгжээ, idle timeout, tenant сонголт, impersonation
// handover — цөмийн нэвтрэлтийн бүх баталгаа бодит DB дээр.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/jackc/pgx/v5/pgxpool"
)

type env struct {
	svc          *auth.Service
	authP, owner *pgxpool.Pool
	userID, tid  string
	email        string
}

func setup(t *testing.T) *env {
	t.Helper()
	authURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_AUTH"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if authURL == "" || ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL_AUTH / _OWNER шаардлагатай")
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
	e := &env{authP: open(authURL), owner: open(ownerURL), email: "atest@x.mn"}
	clean := func() {
		_, _ = e.owner.Exec(ctx, `DELETE FROM tenants WHERE slug LIKE 'atest%'`)
		_, _ = e.owner.Exec(ctx, `DELETE FROM users WHERE email LIKE 'atest%'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		e.authP.Close()
		e.owner.Close()
	})
	hash, err := password.Hash("password-12")
	if err != nil {
		t.Fatal(err)
	}
	if err := e.owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ($1,$2,'А') RETURNING id`,
		e.email, hash).Scan(&e.userID); err != nil {
		t.Fatal(err)
	}
	if err := e.owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('atest','А') RETURNING id`).Scan(&e.tid); err != nil {
		t.Fatal(err)
	}
	if _, err := e.owner.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2)`, e.tid, e.userID); err != nil {
		t.Fatal(err)
	}
	e.svc = auth.NewService(e.authP, false)
	return e
}

func (e *env) start(t *testing.T) (sid, cookie string) {
	t.Helper()
	w := httptest.NewRecorder()
	sid, err := e.svc.StartSession(context.Background(), w, e.userID)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			cookie = c.Value
		}
	}
	if cookie == "" {
		t.Fatal("cookie алга")
	}
	return sid, cookie
}

func req(cookie string) *http.Request {
	r := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	if cookie != "" {
		r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookie})
	}
	return r
}

func TestSessionLifecycle(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	sid, cookie := e.start(t)

	p, ok := e.svc.Resolve(ctx, req(cookie))
	if !ok || p.UserID != e.userID || p.TenantID != "" || p.ImpersonatedBy != "" {
		t.Fatalf("Resolve = %+v %v", p, ok)
	}
	// Tenant сонголт (гишүүн).
	if ok, err := e.svc.SetTenant(ctx, sid, e.tid); err != nil || !ok {
		t.Fatalf("SetTenant = %v %v", ok, err)
	}
	if p, _ := e.svc.Resolve(ctx, req(cookie)); p.TenantID != e.tid {
		t.Fatalf("tenant тогтоогүй: %+v", p)
	}
	// Гишүүн биш tenant — татгалзана.
	var other string
	if err := e.owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('atest2','Б') RETURNING id`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if ok, _ := e.svc.SetTenant(ctx, sid, other); ok {
		t.Fatal("гишүүн бус tenant сонгогдов")
	}
	// Гишүүнчлэл хасагдвал tenant автоматаар унтарна (lookup дотор шалгагдана).
	if _, err := e.owner.Exec(ctx, `DELETE FROM memberships WHERE tenant_id = $1::uuid AND user_id = $2::uuid`, e.tid, e.userID); err != nil {
		t.Fatal(err)
	}
	if p, _ := e.svc.Resolve(ctx, req(cookie)); p.TenantID != "" {
		t.Fatalf("хасагдсан гишүүн tenant хэвээр: %+v", p)
	}
	// Гарах — session устна.
	w := httptest.NewRecorder()
	e.svc.EndSession(ctx, w, req(cookie))
	if _, ok := e.svc.Resolve(ctx, req(cookie)); ok {
		t.Fatal("EndSession-ий дараа session амьд")
	}
	// Байхгүй cookie.
	if _, ok := e.svc.Resolve(ctx, req("хуурмаг")); ok {
		t.Fatal("хуурамч токен нэвтэрлээ")
	}
	if _, ok := e.svc.Resolve(ctx, req("")); ok {
		t.Fatal("cookie-гүй нэвтэрлээ")
	}
}

func TestSessionIdleTimeoutAndExpiry(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	_, cookie := e.start(t)
	// 2 цагийн өмнөх хэрэглээ → 90 мин idle-ээс хэтэрсэн.
	if _, err := e.owner.Exec(ctx, `UPDATE sessions SET last_seen_at = now() - interval '2 hours' WHERE user_id = $1::uuid`, e.userID); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.svc.Resolve(ctx, req(cookie)); ok {
		t.Fatal("idle timeout ажиллаагүй")
	}
	// Хугацаа дууссан session.
	_, cookie2 := e.start(t)
	if _, err := e.owner.Exec(ctx, `UPDATE sessions SET expires_at = now() - interval '1 minute' WHERE user_id = $1::uuid AND expires_at > now()`, e.userID); err != nil {
		t.Fatal(err)
	}
	if _, ok := e.svc.Resolve(ctx, req(cookie2)); ok {
		t.Fatal("хугацаа дууссан session амьд")
	}
}

func TestAccountLockout(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	// 4 удаа буруу — түгжигдэхгүй.
	for i := 0; i < 4; i++ {
		locked, err := e.svc.LoginResult(ctx, e.email, false)
		if err != nil {
			t.Fatal(err)
		}
		if locked {
			t.Fatalf("%d дэх оролдлогод түгжигдэв", i+1)
		}
	}
	if until, _ := e.svc.Lockout(ctx, e.email); until != nil {
		t.Fatal("4 оролдлогын дараа түгжээтэй")
	}
	// 5 дахь — түгжинэ.
	locked, err := e.svc.LoginResult(ctx, e.email, false)
	if err != nil || !locked {
		t.Fatalf("5 дахь оролдлого: locked=%v err=%v", locked, err)
	}
	until, err := e.svc.Lockout(ctx, e.email)
	if err != nil || until == nil || until.Before(time.Now()) {
		t.Fatalf("Lockout = %v %v", until, err)
	}
	// Амжилттай нэвтрэлт тоолуурыг тэглэнэ.
	if _, err := e.owner.Exec(ctx, `SELECT auth_login_result($1::varchar(255), true)`, e.email); err != nil {
		t.Fatal(err)
	}
	if until, _ := e.svc.Lockout(ctx, e.email); until != nil {
		t.Fatal("амжилтын дараа түгжээтэй хэвээр")
	}
	// Байхгүй имэйл — түгжээгүй, алдаагүй.
	if until, err := e.svc.Lockout(ctx, "atest-байхгүй@x.mn"); err != nil || until != nil {
		t.Fatalf("байхгүй имэйл: %v %v", until, err)
	}
}

func TestHandoverAndTenantSessionRevoke(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	// Админ хэрэглэгч.
	var adminID string
	if err := e.owner.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, platform_admin) VALUES ('atest-admin@x.mn','x','Админ',true) RETURNING id`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	// Handover: зөвхөн гишүүн, зөвхөн платформ админаас.
	tok, err := e.svc.CreateHandover(ctx, adminID, e.userID, e.tid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.svc.CreateHandover(ctx, e.userID, e.userID, e.tid); err == nil {
		t.Fatal("админ бус хүн handover үүсгэв")
	}
	if _, err := e.svc.CreateHandover(ctx, adminID, adminID, e.tid); err == nil {
		t.Fatal("платформ админыг impersonate хийв")
	}
	// Consume → impersonated session.
	w := httptest.NewRecorder()
	hv, err := e.svc.ConsumeHandover(ctx, w, tok)
	if err != nil || hv == nil {
		t.Fatalf("ConsumeHandover = %v %v", hv, err)
	}
	var cookie string
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			cookie = c.Value
		}
	}
	p, ok := e.svc.Resolve(ctx, req(cookie))
	if !ok || p.UserID != e.userID || p.ImpersonatedBy != adminID || p.TenantID != e.tid {
		t.Fatalf("impersonated session = %+v", p)
	}
	// Нэг л удаа.
	if hv2, _ := e.svc.ConsumeHandover(ctx, httptest.NewRecorder(), tok); hv2 != nil {
		t.Fatal("handover дахин хэрэглэгдэв")
	}
	// Хугацаа дууссан token.
	tok2, err := e.svc.CreateHandover(ctx, adminID, e.userID, e.tid)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := e.owner.Exec(ctx, `UPDATE handover_tokens SET expires_at = now() - interval '1 minute'`); err != nil {
		t.Fatal(err)
	}
	if hv3, _ := e.svc.ConsumeHandover(ctx, httptest.NewRecorder(), tok2); hv3 != nil {
		t.Fatal("хугацаа дууссан handover ажиллав")
	}
	// Tenant-ийн session-ууд бөөнөөр устана.
	if _, err := e.svc.SetTenant(ctx, mustSession(t, e), e.tid); err != nil {
		t.Fatal(err)
	}
	n, err := e.svc.RevokeTenantSessions(ctx, e.tid)
	if err != nil || n < 1 {
		t.Fatalf("RevokeTenantSessions = %d %v", n, err)
	}
	if _, ok := e.svc.Resolve(ctx, req(cookie)); ok {
		t.Fatal("revoke-ийн дараа session амьд")
	}
}

func mustSession(t *testing.T, e *env) string {
	t.Helper()
	sid, _ := e.start(t)
	return sid
}

func TestIsMember(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	ok, err := e.svc.IsMember(ctx, e.tid, e.userID)
	if err != nil || !ok {
		t.Fatalf("гишүүн = %v %v", ok, err)
	}
	var other string
	if err := e.owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('atest3','В') RETURNING id`).Scan(&other); err != nil {
		t.Fatal(err)
	}
	if ok, _ := e.svc.IsMember(ctx, other, e.userID); ok {
		t.Fatal("гишүүн бус нь true")
	}
}
