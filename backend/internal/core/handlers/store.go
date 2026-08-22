package handlers

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/appstore"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type Store struct {
	DB        nexus.DB
	Installer *appstore.Installer
	Gate      *appstore.Gate
	Audit     *audit.Recorder
}

// GET /api/store/apps — каталог + энэ tenant-ийн суулгалтын төлөв.
func (h *Store) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT a.id, a.short_id, a.name, a.version, a.description, a.publisher,
		       a.go_module, a.compiled,
		       coalesce(i.status, '') AS status, coalesce(i.version, '') AS installed_version
		  FROM apps a
		  LEFT JOIN app_installations i
		    ON i.app_id = a.id AND i.tenant_id = $1::uuid
		 ORDER BY a.name`, nexus.TenantID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "store query failed")
		return
	}
	defer rows.Close()
	type appRow struct {
		ID               string `json:"id"`
		ShortID          string `json:"short_id"`
		Name             string `json:"name"`
		Version          string `json:"version"`
		Description      string `json:"description"`
		Publisher        string `json:"publisher"`
		GoModule         string `json:"go_module"`
		Compiled         bool   `json:"compiled"`
		Status           string `json:"status"`
		InstalledVersion string `json:"installed_version"`
	}
	out := []appRow{}
	for rows.Next() {
		var a appRow
		if err := rows.Scan(&a.ID, &a.ShortID, &a.Name, &a.Version, &a.Description,
			&a.Publisher, &a.GoModule, &a.Compiled, &a.Status, &a.InstalledVersion); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "store query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"apps": out})
}

// GET /api/catalog — НИЙТИЙН каталог (landing-ийн апп дэлгүүр хуудас):
// auth шаардахгүй, суулгалтын мэдээлэл агуулахгүй.
func (h *Store) Catalog(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, short_id, name, version, description, publisher, compiled
		  FROM apps ORDER BY compiled DESC, name`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "catalog query failed")
		return
	}
	defer rows.Close()
	type appRow struct {
		ID          string `json:"id"`
		ShortID     string `json:"short_id"`
		Name        string `json:"name"`
		Version     string `json:"version"`
		Description string `json:"description"`
		Publisher   string `json:"publisher"`
		Compiled    bool   `json:"compiled"`
	}
	out := []appRow{}
	for rows.Next() {
		var a appRow
		if err := rows.Scan(&a.ID, &a.ShortID, &a.Name, &a.Version, &a.Description,
			&a.Publisher, &a.Compiled); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "catalog query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"apps": out})
}

// GET /api/store/apps/{id}/history — нийтлэгчийн хувилбарууд + энэ tenant-ийн
// үйл явдлууд (суулгах/асаах/унтраах/шинэчлэх).
func (h *Store) History(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	type rel struct {
		Version string    `json:"version"`
		SeenAt  time.Time `json:"seen_at"`
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT version, seen_at FROM app_releases WHERE app_id = $1::varchar(128) ORDER BY seen_at DESC`, appID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "history query failed")
		return
	}
	releases, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (rel, error) {
		var x rel
		err := row.Scan(&x.Version, &x.SeenAt)
		return x, err
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "history query failed")
		return
	}
	type ev struct {
		Action      string    `json:"action"`
		FromVersion string    `json:"from_version"`
		ToVersion   string    `json:"to_version"`
		UserName    string    `json:"user_name"`
		At          time.Time `json:"at"`
	}
	erows, err := h.DB.Query(r.Context(), `
		SELECT e.action, e.from_version, e.to_version, coalesce(u.name, ''), e.at
		  FROM installation_events e LEFT JOIN users u ON u.id = e.user_id
		 WHERE e.tenant_id = $1::uuid AND e.app_id = $2::varchar(128)
		 ORDER BY e.id DESC LIMIT 100`, nexus.TenantID(r.Context()), appID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "history query failed")
		return
	}
	events, err := pgx.CollectRows(erows, func(row pgx.CollectableRow) (ev, error) {
		var x ev
		err := row.Scan(&x.Action, &x.FromVersion, &x.ToVersion, &x.UserName, &x.At)
		return x, err
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "history query failed")
		return
	}
	if releases == nil {
		releases = []rel{}
	}
	if events == nil {
		events = []ev{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"releases": releases, "events": events})
}

// POST /api/store/apps/{id}/install
func (h *Store) Install(w http.ResponseWriter, r *http.Request) {
	appID := chi.URLParam(r, "id")
	err := h.Installer.Install(r.Context(), appID)
	switch {
	case errors.Is(err, appstore.ErrNotFound):
		httpx.Error(w, http.StatusNotFound, "апп олдсонгүй")
		return
	case errors.Is(err, appstore.ErrNotCompiled):
		httpx.Error(w, http.StatusConflict,
			"энэ апп бинарид ороогүй байна — `nexus add <short_id>` коммандаар нэмээд дахин build хийнэ")
		return
	case err != nil:
		log.Printf("app install %s: %v", appID, err)
		httpx.Error(w, http.StatusInternalServerError, "суулгалт амжилтгүй боллоо")
		return
	}
	h.Gate.Invalidate(nexus.TenantID(r.Context()))
	h.Audit.Record(r.Context(), "app.install", appID, nil)
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// POST /api/store/apps/{id}/disable ба /enable
func (h *Store) SetStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		appID := chi.URLParam(r, "id")
		if err := h.Installer.SetStatus(r.Context(), appID, status); err != nil {
			if errors.Is(err, appstore.ErrNotFound) {
				httpx.Error(w, http.StatusNotFound, "суулгалт олдсонгүй")
				return
			}
			httpx.Error(w, http.StatusInternalServerError, "status change failed")
			return
		}
		h.Gate.Invalidate(nexus.TenantID(r.Context()))
		h.Audit.Record(r.Context(), "app."+status, appID, nil)
		httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
}
