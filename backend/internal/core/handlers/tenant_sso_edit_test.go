package handlers_test

// Нэмэлт байгууллага үүсгэх (CreateTenant) ба SSO клиент засах — үлдсэн
// хамрагдаагүй handler-ууд.

import (
	"net/http"
	"testing"
)

func TestCreateAdditionalTenant(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "multi")
	first := s.tenantID(t)

	out := s.json(t, http.MethodPost, "/api/tenants", map[string]string{"name": "Хоёр дахь", "slug": "htest-multi2"})
	second, _ := out["tenant_id"].(string)
	if second == "" || second == first {
		t.Fatalf("шинэ байгууллага = %v", out)
	}
	// Хоёулаа /api/me-д харагдана.
	me := s.json(t, http.MethodGet, "/api/me", nil)
	if len(me["tenants"].([]any)) != 2 {
		t.Fatalf("байгууллагууд = %v", me["tenants"])
	}
	// Шинэ рүү шилжээд админ эрхтэй эсэхийг шалгана.
	if w := s.do(t, http.MethodPost, "/api/session/tenant", map[string]string{"tenant_id": second}); w.Code != 200 {
		t.Fatalf("шилжих = %d", w.Code)
	}
	me = s.json(t, http.MethodGet, "/api/me", nil)
	perms, _ := me["permissions"].(map[string]any)
	if perms["core.members.manage"] != "all" {
		t.Fatalf("шинэ байгууллагад админ эрх алга: %v", perms)
	}
	// Давхардсан slug.
	if w := s.do(t, http.MethodPost, "/api/tenants", map[string]string{"name": "Дахин", "slug": "htest-multi2"}); w.Code != http.StatusConflict {
		t.Fatalf("давхардсан slug = %d (409 хүлээсэн)", w.Code)
	}
	// Хоосон талбар.
	for _, body := range []map[string]string{{"name": "", "slug": "x"}, {"name": "X", "slug": ""}} {
		if w := s.do(t, http.MethodPost, "/api/tenants", body); w.Code != http.StatusBadRequest {
			t.Fatalf("хоосон талбар %v = %d", body, w.Code)
		}
	}
}

func TestUpdateSSOClient(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "ssoedit")
	out := s.json(t, http.MethodPost, "/api/sso-clients", map[string]any{
		"name": "Эх", "redirect_uris": []string{"https://a.mn/cb"}, "scopes": "openid"})
	id := out["id"].(string)

	if w := s.do(t, http.MethodPut, "/api/sso-clients/"+id, map[string]any{
		"name": "Шинэчилсэн", "redirect_uris": []string{"https://a.mn/cb", "https://b.mn/cb"},
		"post_logout_uris": []string{"https://a.mn/"}, "scopes": "openid profile email"}); w.Code != 200 {
		t.Fatalf("засах = %d: %s", w.Code, w.Body.String())
	}
	list := s.json(t, http.MethodGet, "/api/sso-clients", nil)
	c := list["clients"].([]any)[0].(map[string]any)
	if c["name"] != "Шинэчилсэн" || len(c["redirect_uris"].([]any)) != 2 || c["scopes"] != "openid profile email" {
		t.Fatalf("шинэчлэлт = %v", c)
	}
	// Буруу оролт, буруу id, өөр байгууллагын клиент.
	if w := s.do(t, http.MethodPut, "/api/sso-clients/"+id, map[string]any{
		"name": "X", "redirect_uris": []string{"http://evil.mn/cb"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу redirect = %d", w.Code)
	}
	if w := s.do(t, http.MethodPut, "/api/sso-clients/тийм-биш", map[string]any{
		"name": "X", "redirect_uris": []string{"https://a.mn/cb"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу uuid = %d", w.Code)
	}
	if w := s.do(t, http.MethodPut, "/api/sso-clients/00000000-0000-0000-0000-000000000009", map[string]any{
		"name": "X", "redirect_uris": []string{"https://a.mn/cb"}}); w.Code != http.StatusNotFound {
		t.Fatalf("байхгүй клиент = %d", w.Code)
	}
	other := h.signup(t, "ssoedit2")
	if w := other.do(t, http.MethodPut, "/api/sso-clients/"+id, map[string]any{
		"name": "Hack", "redirect_uris": []string{"https://a.mn/cb"}}); w.Code != http.StatusNotFound {
		t.Fatalf("өөр байгууллагын клиент = %d (404 хүлээсэн)", w.Code)
	}
}
