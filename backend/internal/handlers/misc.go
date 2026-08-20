package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/platform/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

// Misc — цэс, audit.
type Misc struct {
	DB    nexus.DB
	Perms *rbac.Store
}

// GET /api/menu — идэвхтэй tenant-д суусан аппуудын цэс. Хэрэглэгчид
// тухайн модулийн ядаж нэг permission байвал л модулийн цэс харагдана.
func (h *Misc) Menu(w http.ResponseWriter, r *http.Request) {
	locale := r.Header.Get("X-Lang")
	if locale == "" {
		locale = "mn"
	}
	tenantID := nexus.TenantID(r.Context())
	userID := nexus.UserID(r.Context())

	installed := map[string]bool{}
	rows, err := h.DB.Query(r.Context(),
		`SELECT app_id FROM app_installations WHERE tenant_id = $1::uuid AND status = 'enabled'`,
		tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "menu query failed")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		installed[id] = true
	}

	grants, err := h.Perms.UserGrants(r.Context(), tenantID, userID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "grants failed")
		return
	}

	type menuItem struct {
		ID    string `json:"id"`
		Label string `json:"label"`
		Path  string `json:"path"`
		Icon  string `json:"icon"`
		Order int    `json:"order"`
	}
	type appMenu struct {
		AppID   string     `json:"app_id"`
		ShortID string     `json:"short_id"`
		Name    string     `json:"name"`
		Items   []menuItem `json:"items"`
	}
	out := []appMenu{}
	for _, m := range nexus.Registered() {
		if !installed[m.ID()] {
			continue
		}
		prefix := m.ShortID() + "."
		hasAny := false
		for code := range grants {
			if strings.HasPrefix(code, prefix) {
				hasAny = true
				break
			}
		}
		if !hasAny {
			continue
		}
		am := appMenu{AppID: m.ID(), ShortID: m.ShortID(), Name: m.Name()}
		for _, item := range m.Menus() {
			am.Items = append(am.Items, menuItem{
				ID:    item.ID,
				Label: item.LocalizedLabel(locale),
				Path:  item.Path,
				Icon:  item.Icon,
				Order: item.Order,
			})
		}
		out = append(out, am)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"apps": out})
}

// GET /api/audit?limit=50&before=<id> — tenant-ийн audit лог.
func (h *Misc) Audit(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil && v > 0 && v <= 200 {
		limit = v
	}
	before := int64(1<<62 - 1)
	if v, err := strconv.ParseInt(r.URL.Query().Get("before"), 10, 64); err == nil && v > 0 {
		before = v
	}
	rows, err := h.DB.Query(r.Context(), `
		SELECT a.id, coalesce(a.user_id::text, ''), coalesce(u.name, ''), a.action, a.object,
		       a.details, a.hash, a.occurred_at
		  FROM audit_log a LEFT JOIN users u ON u.id = a.user_id
		 WHERE a.tenant_id = $1::uuid AND a.id < $2::bigint
		 ORDER BY a.id DESC LIMIT $3::int`,
		nexus.TenantID(r.Context()), before, limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "audit query failed")
		return
	}
	defer rows.Close()
	type auditRow struct {
		ID         int64          `json:"id"`
		UserID     string         `json:"user_id"`
		UserName   string         `json:"user_name"`
		Action     string         `json:"action"`
		Object     string         `json:"object"`
		Details    map[string]any `json:"details"`
		Hash       string         `json:"hash"`
		OccurredAt time.Time      `json:"occurred_at"`
	}
	out := []auditRow{}
	for rows.Next() {
		var x auditRow
		if err := rows.Scan(&x.ID, &x.UserID, &x.UserName, &x.Action, &x.Object,
			&x.Details, &x.Hash, &x.OccurredAt); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"entries": out})
}

// GET /api/audit/verify — гинжийг шалгана.
func (h *Misc) AuditVerify(w http.ResponseWriter, r *http.Request) {
	var brokenAt *int64
	err := h.DB.QueryRow(r.Context(), `SELECT audit_verify($1::uuid)`,
		nexus.TenantID(r.Context())).Scan(&brokenAt)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "verify failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"intact": brokenAt == nil, "broken_at": brokenAt})
}
