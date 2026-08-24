package handlers_test

// Платформ админы уншилтын endpoint-ууд, permission каталог, устгалын sweep,
// OIDC-ийн end_session — үлдсэн хамрагдаагүй замууд.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/handlers"
)

func TestAdminReadEndpoints(t *testing.T) {
	h := newHarness(t)
	owner := h.signup(t, "adminro")
	tid := owner.tenantID(t)
	adm := h.signup(t, "adminro-admin")
	h.makePlatformAdmin(t, adm.userID)
	adm = h.login(t, "htest-adminro-admin@x.mn", "password-12", "")

	ov := adm.json(t, http.MethodGet, "/api/admin/overview", nil)
	for _, k := range []string{"tenants", "users", "apps"} {
		if ov[k] == nil {
			t.Fatalf("overview-д %s алга: %v", k, ov)
		}
	}
	users := adm.json(t, http.MethodGet, "/api/admin/users", nil)
	list, _ := users["users"].([]any)
	if len(list) == 0 {
		t.Fatalf("админы хэрэглэгчид = %v", users)
	}
	var found bool
	for _, u := range list {
		m := u.(map[string]any)
		if m["email"] == "htest-adminro@x.mn" {
			found = true
			if m["tenants"] == nil || m["created_at"] == nil {
				t.Fatalf("хэрэглэгчийн мөр дутуу: %v", m)
			}
		}
	}
	if !found {
		t.Fatal("шинэ хэрэглэгч админы жагсаалтад алга")
	}
	apps := adm.json(t, http.MethodGet, "/api/admin/apps", nil)
	if apps["apps"] == nil {
		t.Fatalf("админы аппууд = %v", apps)
	}
	audit := adm.json(t, http.MethodGet, "/api/admin/audit", nil)
	if audit["entries"] == nil {
		t.Fatalf("админы audit = %v", audit)
	}
	// Жирийн хэрэглэгчид хаалттай.
	for _, path := range []string{"/api/admin/overview", "/api/admin/users", "/api/admin/apps", "/api/admin/audit"} {
		if w := owner.do(t, http.MethodGet, path, nil); w.Code != http.StatusForbidden {
			t.Fatalf("%s жирийн хэрэглэгчид = %d (403 хүлээсэн)", path, w.Code)
		}
	}
	_ = tid
}

func TestPermissionCatalogEndpoint(t *testing.T) {
	h := newHarness(t)
	s := h.signup(t, "perms")
	out := s.json(t, http.MethodGet, "/api/permissions", nil)
	list, _ := out["permissions"].([]any)
	if len(list) == 0 {
		t.Fatalf("permission каталог хоосон: %v", out)
	}
	var hasCore bool
	for _, p := range list {
		m := p.(map[string]any)
		if m["code"] == "core.members.manage" {
			hasCore = true
			if m["name"] == nil {
				t.Fatalf("permission мөр дутуу: %v", m)
			}
		}
	}
	if !hasCore {
		t.Fatal("core.members.manage каталогт алга")
	}
}

func TestSweepDeletionsRemovesExpiredTenants(t *testing.T) {
	h := newHarness(t)
	owner := h.signup(t, "sweep")
	tid := owner.tenantID(t)
	ctx := context.Background()
	// Хугацаа нь өнгөрсөн устгал.
	if _, err := h.owner.Exec(ctx, `UPDATE tenants SET deletion_scheduled_at = now() - interval '1 hour' WHERE id = $1::uuid`, tid); err != nil {
		t.Fatal(err)
	}
	n, err := handlers.SweepDeletions(ctx, h.admin)
	if err != nil {
		t.Fatalf("SweepDeletions: %v", err)
	}
	if n < 1 {
		t.Fatalf("устгасан тоо = %d", n)
	}
	var left int
	if err := h.owner.QueryRow(ctx, `SELECT count(*) FROM tenants WHERE id = $1::uuid`, tid).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatal("байгууллага устгагдсангүй")
	}
	// Гишүүнчлэл ч cascade-аар устсан.
	if err := h.owner.QueryRow(ctx, `SELECT count(*) FROM memberships WHERE tenant_id = $1::uuid`, tid).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatal("гишүүнчлэл үлдсэн")
	}
	// Товлоогүй байгууллагад хүрэхгүй.
	other := h.signup(t, "sweep-keep")
	otherTid := other.tenantID(t)
	if _, err := handlers.SweepDeletions(ctx, h.admin); err != nil {
		t.Fatal(err)
	}
	if err := h.owner.QueryRow(ctx, `SELECT count(*) FROM tenants WHERE id = $1::uuid`, otherTid).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 1 {
		t.Fatal("товлоогүй байгууллага устав")
	}
	_ = httptest.NewRecorder()
}
