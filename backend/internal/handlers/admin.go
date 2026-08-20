package handlers

import (
	"net/http"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/platform/httpx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Admin — платформын админ панелийн API. Admin pool (nexus_admin,
// nexus_platform гишүүн)-оор ажилладаг тул RLS бодлогууд платформ гэж
// таньж бүх tenant-ийг харуулна. Route-ууд RequirePlatformAdmin-ий цаана.
type Admin struct {
	Pool *pgxpool.Pool // admin pool
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
		Members   int       `json:"members"`
		Apps      int       `json:"apps"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Slug, &x.Name, &x.CreatedAt, &x.Members, &x.Apps); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tenants": out})
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
	httpx.JSON(w, http.StatusOK, map[string]any{"entries": out})
}
