package handlers_test

// Анхны тохиргооны шидтэн: платформ админгүй үед л зэвсэглэнэ, токенгүй/буруу
// токенд 404, зөв токеноор админ + байгууллага үүсээд хаалга хаагдана, DB
// хоёр дахь админыг өөрөө татгалзана.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSetupWizard(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	var admins int
	if err := h.owner.QueryRow(ctx, `SELECT count(*) FROM users WHERE platform_admin`).Scan(&admins); err != nil {
		t.Fatal(err)
	}
	if admins != 0 {
		t.Fatalf("тест DB-д %d платформ админ байна — шидтэний тест хоосон DB шаардана", admins)
	}
	const email, slug = "htest-setup-admin@x.mn", "htest-setup"
	t.Cleanup(func() {
		_, _ = h.owner.Exec(ctx, `DELETE FROM tenants WHERE slug = $1`, slug)
		_, _ = h.owner.Exec(ctx, `DELETE FROM users WHERE email = $1`, email)
	})

	status := func() map[string]any {
		w := h.do(t, nil, http.MethodGet, "/api/setup/status", nil)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return out
	}
	complete := func(token string, body any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(http.MethodPost, "/api/setup/complete", strings.NewReader(string(b)))
		r.Header.Set("Content-Type", "application/json")
		if token != "" {
			r.Header.Set("X-Setup-Token", token)
		}
		return recordRequest(h, r)
	}
	good := map[string]any{
		"admin":        map[string]string{"email": email, "name": "Анхны админ", "password": "setup-pass-12"},
		"organisation": map[string]string{"name": "Тест ХХК", "slug": slug},
	}

	// Зэвсэглээгүй: шаардлагатай ч armed=false, токен ч байхгүй тул 404.
	if st := status(); st["required"] != true || st["armed"] != false {
		t.Fatalf("зэвсэглээгүй status = %v", st)
	}
	if w := complete("", good); w.Code != http.StatusNotFound {
		t.Fatalf("токенгүй = %d", w.Code)
	}

	h.setup.Arm(ctx)
	token := h.setup.Token()
	if len(token) != 64 {
		t.Fatalf("токен = %q", token)
	}
	if st := status(); st["armed"] != true {
		t.Fatalf("зэвсэглэсэн status = %v", st)
	}
	if w := complete("буруу", good); w.Code != http.StatusNotFound {
		t.Fatalf("буруу токен = %d (404 хүлээв)", w.Code)
	}
	bad := map[string]any{"admin": map[string]string{"email": email, "name": "x", "password": "short"}}
	if w := complete(token, bad); w.Code != http.StatusBadRequest {
		t.Fatalf("сул нууц үг = %d", w.Code)
	}
	w := complete(token, good)
	if w.Code != http.StatusCreated {
		t.Fatalf("complete = %d: %s", w.Code, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out["tenant_id"] == nil || out["tenant_error"] != nil {
		t.Fatalf("байгууллага үүссэнгүй: %v", out)
	}
	var isAdmin bool
	if err := h.owner.QueryRow(ctx, `SELECT platform_admin FROM users WHERE email = $1`, email).Scan(&isAdmin); err != nil || !isAdmin {
		t.Fatalf("platform_admin = %v %v", isAdmin, err)
	}
	// Хаалга хаагдсан: токен хүчингүй, status required=false, DB ч хоёр дахийг татгалзана.
	if w := complete(token, good); w.Code != http.StatusNotFound {
		t.Fatalf("дахин complete = %d", w.Code)
	}
	if st := status(); st["required"] != false || st["armed"] != false {
		t.Fatalf("дууссаны дараа status = %v", st)
	}
	h.setup.Arm(ctx) // админтай үед юу ч хийхгүй
	if h.setup.Token() != "" {
		t.Fatal("админ байхад дахин зэвсэглэв")
	}
	var second *string
	if err := h.authP.QueryRow(ctx, `SELECT auth_setup_admin('htest-setup-2@x.mn', 'x', 'y')`).Scan(&second); err != nil || second != nil {
		t.Fatalf("DB хоёр дахь админ = %v %v", second, err)
	}
	// Шинэ админ нэвтэрч байгууллагаа хардаг.
	s := h.login(t, email, "setup-pass-12", out["tenant_id"].(string))
	me := s.json(t, http.MethodGet, "/api/me", nil)
	if me["user"].(map[string]any)["platform_admin"] != true || me["tenant_id"] != out["tenant_id"] {
		t.Fatalf("me = %v", me)
	}
}
