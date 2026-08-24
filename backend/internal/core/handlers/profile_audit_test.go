package handlers_test

// Профайл, нууц үг солих (impersonation хамгаалалт, бусад session-ийг унтраах),
// audit endpoint-ууд (эрх, гинжийн баталгаа), цэс.

import (
	"context"
	"net/http"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
)

func TestProfileUpdateAndPasswordChange(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "prof2")

	// Нэр солих.
	if w := s.do(t, http.MethodPut, "/api/me", map[string]string{"name": "Шинэ Нэр"}); w.Code != 200 {
		t.Fatalf("нэр солих = %d: %s", w.Code, w.Body.String())
	}
	me := s.json(t, http.MethodGet, "/api/me", nil)
	if me["user"].(map[string]any)["name"] != "Шинэ Нэр" {
		t.Fatalf("нэр хадгалагдсангүй: %v", me["user"])
	}
	// Хоосон нэр — 400.
	if w := s.do(t, http.MethodPut, "/api/me", map[string]string{"name": "  "}); w.Code != http.StatusBadRequest {
		t.Fatalf("хоосон нэр = %d", w.Code)
	}

	// Хоёр дахь session нээгээд нууц үг солиход тэр нь унтарна.
	other := h.login(t, "htest-prof2@x.mn", "password-12", "")
	if w := other.do(t, http.MethodGet, "/api/me", nil); w.Code != 200 {
		t.Fatalf("хоёр дахь session = %d", w.Code)
	}
	// Буруу одоогийн нууц үг.
	if w := s.do(t, http.MethodPost, "/api/me/password",
		map[string]string{"current_password": "буруу", "new_password": "шинэ-нууц-12"}); w.Code != http.StatusForbidden {
		t.Fatalf("буруу одоогийн нууц үг = %d (403 хүлээсэн)", w.Code)
	}
	// Богино шинэ нууц үг — ТЭМДЭГТЭЭР тоологдоно (кирилл 6 тэмдэгт = 12 байт
	// байсан ч 8-аас бага тул татгалзана).
	for _, short := range []string{"богино", "abcdefg", "аб"} {
		if w := s.do(t, http.MethodPost, "/api/me/password",
			map[string]string{"current_password": "password-12", "new_password": short}); w.Code != http.StatusBadRequest {
			t.Fatalf("богино нууц үг %q = %d", short, w.Code)
		}
	}
	// Зөв солилт.
	if w := s.do(t, http.MethodPost, "/api/me/password",
		map[string]string{"current_password": "password-12", "new_password": "шинэ-нууц-12"}); w.Code != 200 {
		t.Fatalf("нууц үг солих = %d: %s", w.Code, w.Body.String())
	}
	// Бусад session унтарсан, өөрийнх ажиллана.
	if w := other.do(t, http.MethodGet, "/api/me", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("бусад session = %d (401 хүлээсэн)", w.Code)
	}
	if w := s.do(t, http.MethodGet, "/api/me", nil); w.Code != 200 {
		t.Fatalf("өөрийн session = %d", w.Code)
	}
	// Шинэ нууц үгээр нэвтэрнэ, хуучнаар нэвтрэхгүй.
	if w := h.do(t, nil, http.MethodPost, "/api/login",
		map[string]string{"email": "htest-prof2@x.mn", "password": "password-12"}); w.Code == 200 {
		t.Fatal("хуучин нууц үгээр нэвтэрлээ")
	}
	_ = h.login(t, "htest-prof2@x.mn", "шинэ-нууц-12", "")
}

func TestImpersonatedSessionCannotChangeProfile(t *testing.T) {
	h := newHarness(t)
	target := h.signup(t, "impprof")
	tid := target.tenantID(t)
	adm := h.signup(t, "impprof-admin")
	h.makePlatformAdmin(t, adm.userID)
	adm = h.login(t, "htest-impprof-admin@x.mn", "password-12", "")

	out := adm.json(t, http.MethodPost, "/api/admin/impersonate",
		map[string]string{"tenant_id": tid, "user_id": target.userID})
	token := out["token"].(string)

	// Handover endpoint (form POST) → impersonated session cookie.
	r := formPost(t, "/api/auth/handover", "token="+token)
	w := recordRequest(h, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("handover = %d: %s", w.Code, w.Body.String())
	}
	var cookie string
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			cookie = c.Value
		}
	}
	if cookie == "" {
		t.Fatal("impersonated cookie алга")
	}
	imp := &session{h: h, cookie: cookie}
	me := imp.json(t, http.MethodGet, "/api/me", nil)
	if me["impersonated_by"] != adm.userID {
		t.Fatalf("impersonated_by = %v", me["impersonated_by"])
	}
	// Профайл/нууц үг өөрчлөх хориотой.
	if w := imp.do(t, http.MethodPut, "/api/me", map[string]string{"name": "Hack"}); w.Code != http.StatusForbidden {
		t.Fatalf("impersonated нэр солих = %d (403 хүлээсэн)", w.Code)
	}
	if w := imp.do(t, http.MethodPost, "/api/me/password",
		map[string]string{"current_password": "password-12", "new_password": "hack-12345"}); w.Code != http.StatusForbidden {
		t.Fatalf("impersonated нууц үг = %d (403 хүлээсэн)", w.Code)
	}
	// Токен нэг л удаа.
	if w := recordRequest(h, formPost(t, "/api/auth/handover", "token="+token)); w.Code != http.StatusUnauthorized {
		t.Fatalf("handover дахин = %d (401 хүлээсэн)", w.Code)
	}
	// Токенгүй.
	if w := recordRequest(h, formPost(t, "/api/auth/handover", "")); w.Code != http.StatusBadRequest {
		t.Fatalf("токенгүй handover = %d", w.Code)
	}
}

func TestAuditEndpoints(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "audit")
	// Signup + role үүсгэх → audit мөрүүд.
	if w := s.do(t, http.MethodPost, "/api/roles", map[string]any{"code": "aud1", "name": "Аудит"}); w.Code >= 400 {
		t.Fatalf("role = %d", w.Code)
	}
	out := s.json(t, http.MethodGet, "/api/audit?limit=10", nil)
	entries, _ := out["entries"].([]any)
	if len(entries) == 0 {
		t.Fatalf("audit хоосон: %v", out)
	}
	first := entries[0].(map[string]any)
	for _, k := range []string{"id", "action", "hash", "occurred_at"} {
		if first[k] == nil {
			t.Fatalf("audit мөрөнд %s алга: %v", k, first)
		}
	}
	// Гинж бүрэн.
	v := s.json(t, http.MethodGet, "/api/audit/verify", nil)
	if v["intact"] != true {
		t.Fatalf("гинж = %v", v)
	}
	// Эрхгүй хэрэглэгч (core.audit.read байхгүй) — 403.
	hr := hrSession(t, h, s, s.tenantID(t), "htest-audit-hr@x.mn")
	if w := hr.do(t, http.MethodGet, "/api/audit", nil); w.Code != http.StatusForbidden {
		t.Fatalf("эрхгүй audit = %d (403 хүлээсэн)", w.Code)
	}
	// Цэс — суулгасан аппгүй бол хоосон.
	menu := s.json(t, http.MethodGet, "/api/menu", nil)
	if menu["apps"] == nil {
		t.Fatalf("цэс = %v", menu)
	}
}

func TestLogoutClearsSession(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "logout")
	if w := s.do(t, http.MethodPost, "/api/logout", nil); w.Code != 200 {
		t.Fatalf("logout = %d", w.Code)
	}
	if w := s.do(t, http.MethodGet, "/api/me", nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("logout дараа = %d (401 хүлээсэн)", w.Code)
	}
	// Session-гүй logout ч алдаагүй.
	if w := h.do(t, nil, http.MethodPost, "/api/logout", nil); w.Code != 200 {
		t.Fatalf("cookie-гүй logout = %d", w.Code)
	}
	_ = context.Background()
}
