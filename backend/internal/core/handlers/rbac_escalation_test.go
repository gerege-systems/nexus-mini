package handlers_test

// RBAC-ийн хилүүд: эрх дээшлүүлэх бүх мэдэгдэж буй зам хаагдсан эсэх.
// Эдгээр нь бодит аудитаар илэрсэн алдаанууд — регресс болж давтагдахгүй.

import (
	"net/http"
	"testing"
)

// hrSession — core.members.manage-тэй (гэхдээ админ БИШ) хэрэглэгч бэлдэнэ.
func hrSession(t *testing.T, h *harness, owner *session, tenantID, email string) *session {
	t.Helper()
	if w := owner.do(t, http.MethodPost, "/api/roles", map[string]any{"code": "hr", "name": "HR"}); w.Code >= 400 {
		t.Fatalf("role үүсгэх = %d: %s", w.Code, w.Body.String())
	}
	roles := owner.json(t, http.MethodGet, "/api/roles", nil)
	var hrID string
	for _, r := range roles["roles"].([]any) {
		m := r.(map[string]any)
		if m["code"] == "hr" {
			hrID = m["id"].(string)
		}
	}
	if hrID == "" {
		t.Fatal("hr role олдсонгүй")
	}
	if w := owner.do(t, http.MethodPut, "/api/roles/"+hrID+"/grants",
		map[string]any{"grants": map[string]string{"core.members.manage": "all"}}); w.Code != 200 {
		t.Fatalf("grants = %d: %s", w.Code, w.Body.String())
	}
	if w := owner.do(t, http.MethodPost, "/api/members",
		map[string]any{"email": email, "name": "HR", "password": "password-12", "roles": []string{"hr"}}); w.Code >= 400 {
		t.Fatalf("гишүүн нэмэх = %d: %s", w.Code, w.Body.String())
	}
	return h.login(t, email, "password-12", tenantID)
}

func TestMembersManagerCannotEscalate(t *testing.T) {
	h := newHarness(t)
	admin := h.signup(t, "esc")
	tid := admin.tenantID(t)
	hr := hrSession(t, h, admin, tid, "htest-esc-hr@x.mn")

	// 1. Өөрийгөө admin болгох (шинэ гишүүнээр) — role нь өөрт байхгүй эрх олгоно.
	if w := hr.do(t, http.MethodPost, "/api/members",
		map[string]any{"email": "htest-esc-new@x.mn", "name": "Ш", "password": "password-12", "roles": []string{"admin"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("admin role оноох = %d (400 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// 2. Байгаа гишүүний (админы) role-ийг AddMember-ээр дарж бичих → 409.
	if w := hr.do(t, http.MethodPost, "/api/members",
		map[string]any{"email": "htest-esc@x.mn", "roles": []string{"user"}}); w.Code != http.StatusConflict {
		t.Fatalf("байгаа гишүүнийг AddMember = %d (409 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// 3. Админы role-ийг буулгах.
	members := hr.json(t, http.MethodGet, "/api/members", nil)
	var adminMembership string
	for _, m := range members["members"].([]any) {
		mm := m.(map[string]any)
		if mm["email"] == "htest-esc@x.mn" {
			adminMembership = mm["membership_id"].(string)
		}
	}
	if adminMembership == "" {
		t.Fatal("админы гишүүнчлэл олдсонгүй")
	}
	if w := hr.do(t, http.MethodPut, "/api/members/"+adminMembership+"/roles",
		map[string]any{"roles": []string{"user"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("админыг буулгах = %d (400 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// 4. Админыг хасах.
	if w := hr.do(t, http.MethodDelete, "/api/members/"+adminMembership, nil); w.Code != http.StatusBadRequest {
		t.Fatalf("админыг хасах = %d (400 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// 5. Roles API-д хандах эрхгүй (core.roles.manage байхгүй).
	if w := hr.do(t, http.MethodGet, "/api/roles", nil); w.Code != http.StatusForbidden {
		t.Fatalf("roles = %d (403 хүлээсэн)", w.Code)
	}
	// 6. Буруу uuid → 400 (500 биш).
	if w := hr.do(t, http.MethodDelete, "/api/members/тийм-биш", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу uuid = %d", w.Code)
	}
}

func TestSetGrantsRules(t *testing.T) {
	h := newHarness(t)
	admin := h.signup(t, "grants")
	tid := admin.tenantID(t)
	roles := admin.json(t, http.MethodGet, "/api/roles", nil)
	var adminID, managerID string
	for _, r := range roles["roles"].([]any) {
		m := r.(map[string]any)
		switch m["code"] {
		case "admin":
			adminID = m["id"].(string)
		case "manager":
			managerID = m["id"].(string)
		}
	}
	// admin role-ийн оноолт гараар засагдахгүй.
	if w := admin.do(t, http.MethodPut, "/api/roles/"+adminID+"/grants",
		map[string]any{"grants": map[string]string{"core.audit.read": "all"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("admin role засах = %d (400 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// own_scope биш permission-д "own" өгөх.
	if w := admin.do(t, http.MethodPut, "/api/roles/"+managerID+"/grants",
		map[string]any{"grants": map[string]string{"core.audit.read": "own"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("own scope = %d (400 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// Байхгүй permission код.
	if w := admin.do(t, http.MethodPut, "/api/roles/"+managerID+"/grants",
		map[string]any{"grants": map[string]string{"үл.мэдэх": "all"}}); w.Code != http.StatusBadRequest {
		t.Fatalf("үл мэдэх код = %d (400 хүлээсэн)", w.Code)
	}
	// Зөв: админ өөрийн эзэмшдэг эрхийг manager-т өгнө.
	if w := admin.do(t, http.MethodPut, "/api/roles/"+managerID+"/grants",
		map[string]any{"grants": map[string]string{"core.members.manage": "all"}}); w.Code != 200 {
		t.Fatalf("зөв оноолт = %d: %s", w.Code, w.Body.String())
	}
	// Буруу role код.
	if w := admin.do(t, http.MethodPost, "/api/roles", map[string]any{"code": "Буруу Код", "name": "x"}); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу role код = %d", w.Code)
	}
	// implies=admin-аар өвлөх — админ өөрөө хийж болно.
	if w := admin.do(t, http.MethodPost, "/api/roles", map[string]any{"code": "sup", "name": "Sup", "implies": "admin"}); w.Code >= 400 {
		t.Fatalf("админ implies=admin = %d: %s", w.Code, w.Body.String())
	}
	_ = tid
}

func TestLastAdminProtection(t *testing.T) {
	h := newHarness(t)
	admin := h.signup(t, "lastadmin")
	members := admin.json(t, http.MethodGet, "/api/members", nil)
	me := members["members"].([]any)[0].(map[string]any)
	mid := me["membership_id"].(string)
	// Өөрийгөө user болгох → сүүлчийн админ алга болно → 409.
	if w := admin.do(t, http.MethodPut, "/api/members/"+mid+"/roles", map[string]any{"roles": []string{"user"}}); w.Code != http.StatusConflict {
		t.Fatalf("сүүлчийн админ = %d (409 хүлээсэн): %s", w.Code, w.Body.String())
	}
	// Өөрийгөө хасах ч болохгүй (RemoveMember нь user_id <> өөрөө).
	if w := admin.do(t, http.MethodDelete, "/api/members/"+mid, nil); w.Code != http.StatusNotFound {
		t.Fatalf("өөрийгөө хасах = %d (404 хүлээсэн)", w.Code)
	}
}

func TestMemberLookupIsTenantScoped(t *testing.T) {
	h := newHarness(t)
	a := h.signup(t, "lookupa")
	b := h.signup(t, "lookupb")
	_ = b
	// Өөр байгууллагын хэрэглэгч: exists=true (платформ дээр бий) ч member=false.
	out := a.json(t, http.MethodGet, "/api/members/lookup?email=htest-lookupb@x.mn", nil)
	if out["exists"] != true || out["member"] != false {
		t.Fatalf("өөр tenant-ийн хэрэглэгч: %v", out)
	}
	// Өөрийн гишүүн.
	out = a.json(t, http.MethodGet, "/api/members/lookup?email=htest-lookupa@x.mn", nil)
	if out["exists"] != true || out["member"] != true {
		t.Fatalf("өөрийн гишүүн: %v", out)
	}
	// Байхгүй.
	out = a.json(t, http.MethodGet, "/api/members/lookup?email=htest-үгүй@x.mn", nil)
	if out["exists"] != false {
		t.Fatalf("байхгүй хэрэглэгч: %v", out)
	}
	// Буруу имэйл → 400.
	if w := a.do(t, http.MethodGet, "/api/members/lookup?email=буруу", nil); w.Code != http.StatusBadRequest {
		t.Fatalf("буруу имэйл = %d", w.Code)
	}
}

func TestTenantIsolationBetweenSessions(t *testing.T) {
	h := newHarness(t)
	a := h.signup(t, "isoa")
	b := h.signup(t, "isob")
	bTenant := b.tenantID(t)
	// A нь B-ийн tenant-ийг сонгож чадахгүй (гишүүн биш).
	if w := a.do(t, http.MethodPost, "/api/session/tenant", map[string]string{"tenant_id": bTenant}); w.Code < 400 {
		t.Fatalf("өөр tenant сонгогдов = %d", w.Code)
	}
	// A-гийн гишүүдийн жагсаалтад зөвхөн өөрийнх нь хүн.
	members := a.json(t, http.MethodGet, "/api/members", nil)
	list := members["members"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["email"] != "htest-isoa@x.mn" {
		t.Fatalf("гишүүдийн тусгаарлалт эвдэрсэн: %v", list)
	}
}
