package handlers

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/tenantstate"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Admin — платформын админ панелийн API. Admin pool (nexus_admin,
// nexus_platform гишүүн)-оор ажилладаг тул RLS бодлогууд платформ гэж
// таньж бүх tenant-ийг харуулна. Route-ууд RequirePlatformAdmin-ий цаана.
type Admin struct {
	Pool      *pgxpool.Pool // admin pool
	Svc       *auth.Service
	Rec       *audit.Recorder
	State     *tenantstate.Store
	PortalURL string
}

// GET /api/admin/overview — тоон үзүүлэлтүүд.
func (h *Admin) Overview(w http.ResponseWriter, r *http.Request) {
	var tenants, users, apps, installs int
	err := h.Pool.QueryRow(r.Context(), `
		SELECT (SELECT count(*) FROM tenants),
		       (SELECT count(*) FROM users),
		       (SELECT count(*) FROM apps WHERE compiled),
		       (SELECT count(*) FROM app_installations WHERE status = 'enabled')`).
		Scan(&tenants, &users, &apps, &installs)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "overview failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]int{
		"tenants": tenants, "users": users, "apps": apps, "installations": installs,
	})
}

// GET /api/admin/tenants
func (h *Admin) Tenants(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT t.id, t.slug, t.name, t.created_at,
		       t.suspended_at IS NOT NULL, t.suspension_reason, t.read_only,
		       (SELECT count(*) FROM memberships m WHERE m.tenant_id = t.id),
		       (SELECT count(*) FROM app_installations i
		         WHERE i.tenant_id = t.id AND i.status = 'enabled')
		  FROM tenants t ORDER BY t.created_at DESC`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenants query failed")
		return
	}
	defer rows.Close()
	type row struct {
		ID        string    `json:"id"`
		Slug      string    `json:"slug"`
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
		Suspended bool      `json:"suspended"`
		Reason    string    `json:"reason"`
		ReadOnly  bool      `json:"read_only"`
		Members   int       `json:"members"`
		Apps      int       `json:"apps"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Slug, &x.Name, &x.CreatedAt, &x.Suspended, &x.Reason, &x.ReadOnly, &x.Members, &x.Apps); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenants query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tenants": out})
}

// GET /api/admin/tenants/{id}/members — impersonation сонголтод.
func (h *Admin) TenantMembers(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT u.id, u.name, u.email, u.platform_admin,
		       coalesce(array_agg(ro.code ORDER BY ro.code) FILTER (WHERE ro.code IS NOT NULL), '{}')
		  FROM memberships m
		  JOIN users u ON u.id = m.user_id
		  LEFT JOIN membership_roles mr ON mr.membership_id = m.id
		  LEFT JOIN roles ro ON ro.id = mr.role_id
		 WHERE m.tenant_id = $1::uuid
		 GROUP BY u.id ORDER BY u.name`, chi.URLParam(r, "id"))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "members query failed")
		return
	}
	defer rows.Close()
	type row struct {
		ID            string   `json:"id"`
		Name          string   `json:"name"`
		Email         string   `json:"email"`
		PlatformAdmin bool     `json:"platform_admin"`
		Roles         []string `json:"roles"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Name, &x.Email, &x.PlatformAdmin, &x.Roles); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "members query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"members": out})
}

// POST /api/admin/impersonate {tenant_id, user_id} — нэг удаагийн handover
// URL буцаана; шалгалтууд (гишүүнчлэл, бай platform_admin биш) DB талд.
// Audit: тухайн tenant-ийн гинжид "platform.impersonate" (админ = actor).
func (h *Admin) Impersonate(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	var in struct {
		TenantID string `json:"tenant_id"`
		UserID   string `json:"user_id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if in.TenantID == "" || in.UserID == "" {
		httpx.Error(w, http.StatusBadRequest, "tenant_id, user_id шаардлагатай")
		return
	}
	token, err := h.Svc.CreateHandover(r.Context(), p.UserID, in.UserID, in.TenantID)
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "impersonation боломжгүй (гишүүн биш эсвэл платформ админ)")
		return
	}
	h.Rec.RecordAs(r.Context(), in.TenantID, p.UserID, "platform.impersonate", in.UserID,
		map[string]any{"admin_email": p.Email})
	httpx.JSON(w, http.StatusOK, map[string]string{
		"url": strings.TrimRight(h.PortalURL, "/") + "/api/auth/handover?token=" + url.QueryEscape(token),
	})
}

// PUT /api/admin/tenants/{id}/state {suspended, reason, read_only} —
// түдгэлзүүлэх / зөвхөн-унших. Admin pool (column grant app role-д байхгүй).
func (h *Admin) SetTenantState(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	id := chi.URLParam(r, "id")
	var in struct {
		Suspended bool   `json:"suspended"`
		Reason    string `json:"reason"`
		ReadOnly  bool   `json:"read_only"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Reason = strings.TrimSpace(in.Reason)
	if len(in.Reason) > 300 {
		httpx.Error(w, http.StatusBadRequest, "шалтгаан 300 тэмдэгтээс хэтрэхгүй")
		return
	}
	if !in.Suspended {
		in.Reason = ""
	}
	tag, err := h.Pool.Exec(r.Context(), `
		UPDATE tenants
		   SET suspended_at = CASE WHEN $2::boolean THEN coalesce(suspended_at, now()) ELSE NULL END,
		       suspension_reason = $3::varchar(300), read_only = $4::boolean
		 WHERE id = $1::uuid`, id, in.Suspended, in.Reason, in.ReadOnly)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "state update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "байгууллага олдсонгүй")
		return
	}
	if h.State != nil {
		h.State.Invalidate(id)
	}
	h.Rec.RecordAs(r.Context(), id, p.UserID, "platform.tenant.state", id,
		map[string]any{"suspended": in.Suspended, "reason": in.Reason, "read_only": in.ReadOnly})
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /api/admin/users
func (h *Admin) Users(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT u.id, u.email, u.name, u.platform_admin, u.created_at,
		       (SELECT count(*) FROM memberships m WHERE m.user_id = u.id)
		  FROM users u ORDER BY u.created_at DESC LIMIT 500`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "users query failed")
		return
	}
	defer rows.Close()
	type row struct {
		ID            string    `json:"id"`
		Email         string    `json:"email"`
		Name          string    `json:"name"`
		PlatformAdmin bool      `json:"platform_admin"`
		CreatedAt     time.Time `json:"created_at"`
		Tenants       int       `json:"tenants"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Email, &x.Name, &x.PlatformAdmin, &x.CreatedAt, &x.Tenants); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "users query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"users": out})
}

// GET /api/admin/apps — каталог, суулгалтын тоотой.
func (h *Admin) Apps(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT a.id, a.short_id, a.name, a.version, a.compiled, a.publisher,
		       (SELECT count(*) FROM app_installations i
		         WHERE i.app_id = a.id AND i.status = 'enabled')
		  FROM apps a ORDER BY a.name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "apps query failed")
		return
	}
	defer rows.Close()
	type row struct {
		ID        string `json:"id"`
		ShortID   string `json:"short_id"`
		Name      string `json:"name"`
		Version   string `json:"version"`
		Compiled  bool   `json:"compiled"`
		Publisher string `json:"publisher"`
		Installs  int    `json:"installs"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.ShortID, &x.Name, &x.Version, &x.Compiled,
			&x.Publisher, &x.Installs); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "apps query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"apps": out})
}

// GET /api/admin/audit — бүх tenant-ийн сүүлийн бичилтүүд.
func (h *Admin) Audit(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT a.id, t.slug, coalesce(u.name, ''), a.action, a.object, a.occurred_at
		  FROM audit_log a
		  JOIN tenants t ON t.id = a.tenant_id
		  LEFT JOIN users u ON u.id = a.user_id
		 ORDER BY a.id DESC LIMIT 100`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "audit query failed")
		return
	}
	defer rows.Close()
	type row struct {
		ID         int64     `json:"id"`
		Tenant     string    `json:"tenant"`
		UserName   string    `json:"user_name"`
		Action     string    `json:"action"`
		Object     string    `json:"object"`
		OccurredAt time.Time `json:"occurred_at"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Tenant, &x.UserName, &x.Action, &x.Object, &x.OccurredAt); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "audit query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"entries": out})
}
