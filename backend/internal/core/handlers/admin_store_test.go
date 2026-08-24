package handlers_test

// Платформын админ (tenant төлөв, устгалын хүлээлт, impersonation), app store
// (суулгах/асаах/түүх), байгууллагын профайл, SSO клиент — бүгд бодит DB дээр.

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestTenantStateSuspendReadOnlyAndSessions(t *testing.T) {
	h := newHarness(t)
	owner := h.signup(t, "state")
	tid := owner.tenantID(t)
	adm := h.signup(t, "state-admin")
	h.makePlatformAdmin(t, adm.userID)
	adm = h.login(t, "htest-state-admin@x.mn", "password-12", "")

	// Зөвхөн-унших: GET зөвшөөрөгдөнө, бичих 503.
	if w := adm.do(t, http.MethodPut, "/api/admin/tenants/"+tid+"/state",
		map[string]any{"suspended": false, "read_only": true}); w.Code != 200 {
		t.Fatalf("read_only тохируулах = %d: %s", w.Code, w.Body.String())
	}
	h.state.Invalidate(tid)
	if w := owner.do(t, http.MethodGet, "/api/members", nil); w.Code != 200 {
		t.Fatalf("read-only үед GET = %d", w.Code)
	}
	if w := owner.do(t, http.MethodPost, "/api/roles", map[string]any{"code": "x1", "name": "X"}); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("read-only үед бичих = %d (503 хүлээсэн)", w.Code)
	}
	// /api/me нь RequireUser тул ажиллана; tenant_state харагдана.
	me := owner.json(t, http.MethodGet, "/api/me", nil)
	st, _ := me["tenant_state"].(map[string]any)
	if st == nil || st["read_only"] != true {
		t.Fatalf("tenant_state = %v", me["tenant_state"])
	}

	// Түдгэлзүүлэх: session-ууд шууд устана, tenant route 403.
	if w := adm.do(t, http.MethodPut, "/api/admin/tenants/"+tid+"/state",
		map[string]any{"suspended": true, "reason": "тест", "read_only": false}); w.Code != 200 {
		t.Fatalf("suspend = %d: %s", w.Code, w.Body.String())
	}
	h.state.Invalidate(tid)
	if w := owner.do(t, http.MethodGet, "/api/members", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("suspend-ийн дараа хуучин session = %d (401 хүлээсэн — session устсан)", w.Code)
	}
	owner2 := h.login(t, "htest-state@x.mn", "password-12", tid)
	if w := owner2.do(t, http.MethodGet, "/api/menu", nil); w.Code != http.StatusForbidden {
		t.Fatalf("түдгэлзүүлсэн tenant = %d (403 хүлээсэн)", w.Code)
	}
	// Хаагдсан ч /api/me ажиллаж шалтгааныг өгнө (frontend-ийн хаагдсан дэлгэц).
	me = owner2.json(t, http.MethodGet, "/api/me", nil)
	st, _ = me["tenant_state"].(map[string]any)
	if st == nil || st["suspended"] != true || st["reason"] != "тест" {
		t.Fatalf("хаагдсан үеийн tenant_state = %v", me["tenant_state"])
	}
	// Буцаах.
	if w := adm.do(t, http.MethodPut, "/api/admin/tenants/"+tid+"/state",
		map[string]any{"suspended": false, "read_only": false}); w.Code != 200 {
		t.Fatal("сэргээх амжилтгүй")
	}
	h.state.Invalidate(tid)
	owner3 := h.login(t, "htest-state@x.mn", "password-12", tid)
	if w := owner3.do(t, http.MethodGet, "/api/menu", nil); w.Code != 200 {
		t.Fatalf("сэргээсний дараа = %d", w.Code)
	}
}

func TestTenantDeletionSchedule(t *testing.T) {
	h := newHarness(t)
	owner := h.signup(t, "del")
	tid := owner.tenantID(t)
	adm := h.signup(t, "del-admin")
	h.makePlatformAdmin(t, adm.userID)
	adm = h.login(t, "htest-del-admin@x.mn", "password-12", "")

	if w := adm.do(t, http.MethodPost, "/api/admin/tenants/"+tid+"/delete", nil); w.Code != 200 {
		t.Fatalf("устгалд товлох = %d: %s", w.Code, w.Body.String())
	}
	// Жагсаалтад deletion_at + suspended.
	list := adm.json(t, http.MethodGet, "/api/admin/tenants", nil)
	var found map[string]any
	for _, x := range list["tenants"].([]any) {
		m := x.(map[string]any)
		if m["id"] == tid {
			found = m
		}
	}
	if found == nil || found["deletion_at"] == nil || found["suspended"] != true {
		t.Fatalf("товлосны дараа = %v", found)
	}
	// Цуцлах.
	if w := adm.do(t, http.MethodPost, "/api/admin/tenants/"+tid+"/delete/cancel", nil); w.Code != 200 {
		t.Fatalf("цуцлах = %d", w.Code)
	}
	list = adm.json(t, http.MethodGet, "/api/admin/tenants", nil)
	for _, x := range list["tenants"].([]any) {
		m := x.(map[string]any)
		if m["id"] == tid && (m["deletion_at"] != nil || m["suspended"] != false) {
			t.Fatalf("цуцлалтын дараа = %v", m)
		}
	}
	// Буруу uuid.
	if w := adm.do(t, http.MethodPost, "/api/admin/tenants/тийм-биш/delete", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу uuid = %d", w.Code)
	}
	// Жирийн хэрэглэгч админ API-д хүрэхгүй.
	if w := owner.do(t, http.MethodGet, "/api/admin/tenants", nil); w.Code != http.StatusForbidden && w.Code != http.StatusUnauthorized {
		t.Fatalf("жирийн хэрэглэгч админ API = %d", w.Code)
	}
}

func TestImpersonationHandover(t *testing.T) {
	h := newHarness(t)
	owner := h.signup(t, "imp")
	tid := owner.tenantID(t)
	adm := h.signup(t, "imp-admin")
	h.makePlatformAdmin(t, adm.userID)
	adm = h.login(t, "htest-imp-admin@x.mn", "password-12", "")

	// Гишүүдийн жагсаалт.
	members := adm.json(t, http.MethodGet, "/api/admin/tenants/"+tid+"/members", nil)
	list := members["members"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["email"] != "htest-imp@x.mn" {
		t.Fatalf("гишүүд = %v", list)
	}
	// Handover token.
	out := adm.json(t, http.MethodPost, "/api/admin/impersonate",
		map[string]string{"tenant_id": tid, "user_id": owner.userID})
	token, _ := out["token"].(string)
	if token == "" || !strings.HasSuffix(out["url"].(string), "/api/auth/handover") {
		t.Fatalf("impersonate = %v", out)
	}
	// Платформын админыг impersonate хийхгүй.
	if w := adm.do(t, http.MethodPost, "/api/admin/impersonate",
		map[string]string{"tenant_id": tid, "user_id": adm.userID}); w.Code != http.StatusBadRequest {
		t.Fatalf("админыг impersonate = %d (400 хүлээсэн)", w.Code)
	}
	// Буруу uuid.
	if w := adm.do(t, http.MethodPost, "/api/admin/impersonate",
		map[string]string{"tenant_id": "x", "user_id": owner.userID}); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу uuid = %d", w.Code)
	}
	// Токеныг нэг л удаа хэрэглэнэ (auth.Service дээр шууд).
	ctx := context.Background()
	rec := &fakeWriter{header: http.Header{}}
	hv, err := h.svc.ConsumeHandover(ctx, rec, token)
	if err != nil || hv == nil || hv.UserID != owner.userID || hv.AdminID != adm.userID {
		t.Fatalf("ConsumeHandover = %v %v", hv, err)
	}
	if hv2, err := h.svc.ConsumeHandover(ctx, rec, token); err != nil || hv2 != nil {
		t.Fatalf("дахин хэрэглэлт = %v %v", hv2, err)
	}
	_ = url.Values{}
}

type fakeWriter struct {
	header http.Header
	code   int
}

func (f *fakeWriter) Header() http.Header         { return f.header }
func (f *fakeWriter) Write(b []byte) (int, error) { return len(b), nil }
func (f *fakeWriter) WriteHeader(c int)           { f.code = c }

func TestStoreInstallAndHistory(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "store")
	// Каталог: тест бинарид модуль бүртгэгдээгүй тул суулгах боломжгүй апп.
	if w := s.do(t, http.MethodPost, "/api/store/apps/mn.байхгүй.апп/install", nil); w.Code != http.StatusNotFound {
		t.Fatalf("байхгүй апп = %d (404 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// Store жагсаалт ажиллана.
	if w := s.do(t, http.MethodGet, "/api/store/apps", nil); w.Code != 200 {
		t.Fatalf("store жагсаалт = %d", w.Code)
	}
	// Түүх — хоосон ч 200.
	out := s.json(t, http.MethodGet, "/api/store/apps/mn.байхгүй.апп/history", nil)
	if out["releases"] == nil || out["events"] == nil {
		t.Fatalf("түүх = %v", out)
	}
}

func TestTenantProfileAndSSOClients(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "prof")
	// Профайл унших (гишүүн бүр).
	p := s.json(t, http.MethodGet, "/api/tenant/profile", nil)
	if p["slug"] != "htest-prof" {
		t.Fatalf("профайл = %v", p)
	}
	// Бичих.
	if w := s.do(t, http.MethodPut, "/api/tenant/profile", map[string]any{
		"name": "Шинэ нэр", "legal_name": "Шинэ ХХК", "registration_number": "1234567",
		"email": "info@x.mn"}); w.Code != 200 {
		t.Fatalf("профайл засах = %d: %s", w.Code, w.Body.String())
	}
	p = s.json(t, http.MethodGet, "/api/tenant/profile", nil)
	if p["name"] != "Шинэ нэр" || p["legal_name"] != "Шинэ ХХК" {
		t.Fatalf("хадгалагдсан профайл = %v", p)
	}
	// Буруу имэйл, хоосон нэр.
	for _, body := range []map[string]any{{"name": "X", "email": "буруу"}, {"name": ""}} {
		if w := s.do(t, http.MethodPut, "/api/tenant/profile", body); w.Code != http.StatusBadRequest {
			t.Fatalf("буруу профайл %v = %d", body, w.Code)
		}
	}

	// SSO клиент: admin-д core.sso.manage бий.
	out := s.json(t, http.MethodPost, "/api/sso-clients", map[string]any{
		"name": "Тест ERP", "redirect_uris": []string{"https://erp.mn/cb"}, "scopes": "openid profile"})
	if out["client_id"] == nil || out["client_secret"] == nil {
		t.Fatalf("клиент үүсгэх = %v", out)
	}
	id := out["id"].(string)
	// Public клиент — secret байхгүй.
	pub := s.json(t, http.MethodPost, "/api/sso-clients", map[string]any{
		"name": "SPA", "public": true, "redirect_uris": []string{"http://localhost:3000/cb"}, "scopes": "openid"})
	if pub["client_secret"] != "" && pub["client_secret"] != nil {
		t.Fatalf("public клиентэд secret гарав: %v", pub["client_secret"])
	}
	// Буруу redirect (http, localhost биш) / хоосон жагсаалт / үл мэдэх scope.
	for _, body := range []map[string]any{
		{"name": "X", "redirect_uris": []string{"http://evil.mn/cb"}},
		{"name": "X", "redirect_uris": []string{}},
		{"name": "X", "redirect_uris": []string{"https://a.mn/cb"}, "scopes": "openid үл-мэдэх"},
		{"name": "X", "redirect_uris": []string{"https://a.mn/cb#fragment"}},
	} {
		if w := s.do(t, http.MethodPost, "/api/sso-clients", body); w.Code != http.StatusBadRequest {
			t.Fatalf("буруу клиент %v = %d", body, w.Code)
		}
	}
	// Жагсаалт + устгах.
	list := s.json(t, http.MethodGet, "/api/sso-clients", nil)
	if len(list["clients"].([]any)) != 2 || list["issuer"] != "https://portal.mn/api/oauth2" {
		t.Fatalf("клиентийн жагсаалт = %v", list)
	}
	if w := s.do(t, http.MethodDelete, "/api/sso-clients/"+id, nil); w.Code != 200 {
		t.Fatalf("устгах = %d", w.Code)
	}
	if w := s.do(t, http.MethodDelete, "/api/sso-clients/"+id, nil); w.Code != http.StatusNotFound {
		t.Fatalf("дахин устгах = %d (404 хүлээсэн)", w.Code)
	}
}
