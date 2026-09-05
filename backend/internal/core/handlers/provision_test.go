package handlers_test

// Админ түр нууц үгтэй үүсгэсэн данс: солих хүртэл tenant-ийн route 403
// (password_change_required), /api/me флаг өгнө; сольсны дараа хэвийн.

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

func TestProvisionedMemberMustChangePassword(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	owner := h.signup(t, "prov")
	ownerMe := owner.json(t, http.MethodGet, "/api/me", nil)
	if ownerMe["must_change_password"] != false {
		t.Fatalf("signup хэрэглэгчид флаг = %v", ownerMe["must_change_password"])
	}
	tenantID, _ := ownerMe["tenant_id"].(string)

	const email, temp, fresh = "htest-prov-user@x.mn", "temp-pass-12", "fresh-pass-123"
	t.Cleanup(func() { _, _ = h.owner.Exec(ctx, `DELETE FROM users WHERE email = $1`, email) })
	if w := owner.do(t, http.MethodPost, "/api/members",
		map[string]any{"email": email, "name": "Түр", "password": temp, "roles": []string{"user"}}); w.Code >= 300 {
		t.Fatalf("add member = %d: %s", w.Code, w.Body.String())
	}

	s := h.login(t, email, temp, tenantID)
	me := s.json(t, http.MethodGet, "/api/me", nil)
	if me["must_change_password"] != true {
		t.Fatalf("түр нууц үгтэй дансанд флаг = %v", me["must_change_password"])
	}
	if w := s.do(t, http.MethodGet, "/api/menu", nil); w.Code != http.StatusForbidden ||
		!strings.Contains(w.Body.String(), "password_change_required") {
		t.Fatalf("солиогүй байхад /api/menu = %d: %s", w.Code, w.Body.String())
	}
	// Буруу одоогийн нууц үгээр солигдохгүй.
	if w := s.do(t, http.MethodPost, "/api/me/password",
		map[string]string{"current_password": "буруу-буруу-1", "new_password": fresh}); w.Code != http.StatusForbidden {
		t.Fatalf("буруу одоогийн нууц үг = %d", w.Code)
	}
	if w := s.do(t, http.MethodPost, "/api/me/password",
		map[string]string{"current_password": temp, "new_password": fresh}); w.Code != 200 {
		t.Fatalf("солих = %d: %s", w.Code, w.Body.String())
	}
	me = s.json(t, http.MethodGet, "/api/me", nil)
	if me["must_change_password"] != false {
		t.Fatalf("сольсны дараа флаг = %v", me["must_change_password"])
	}
	if w := s.do(t, http.MethodGet, "/api/menu", nil); w.Code != 200 {
		t.Fatalf("сольсны дараа /api/menu = %d: %s", w.Code, w.Body.String())
	}
	// Түр нууц үг цаашид хэрэггүй (админ мэддэг байсан).
	if w := h.do(t, nil, http.MethodPost, "/api/login", map[string]string{"email": email, "password": temp}); w.Code == 200 {
		t.Fatal("хуучин түр нууц үгээр нэвтэрлээ")
	}
}
