// Package devices — Төхөөрөмжийн бүртгэл: SDK-г баталдаг жишээ модуль.
//
// Модуль хэрхэн бичихийн загвар болдог тул docs/03-module-guide.md-тэй
// хамт уншина. Гол цэгүүд: permission-ууд ShortID prefix-тэй, default
// оноолт нь ТУНХАГЛАЛ (DefaultRoles), "user:own" нь own scope-оор,
// route-ууд аль хэдийн хамгаалагдсан router дээр суудаг.
package devices

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) ID() string      { return "mn.gerege.nexus_mini.devices" }
func (m *Module) ShortID() string { return "devices" }
func (m *Module) Name() string    { return "Төхөөрөмжийн бүртгэл" }
func (m *Module) Version() string { return "1.0.0" }

func (m *Module) Dependencies() []nexus.Dependency { return nil }

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{
			Code:         "devices.read",
			Name:         "Төхөөрөмж харах",
			Description:  "Байгууллагын төхөөрөмжийн жагсаалтыг харах",
			DefaultRoles: []string{"manager", "user"},
		},
		{
			Code:         "devices.manage",
			Name:         "Төхөөрөмж бүртгэх, засах",
			Description:  "Төхөөрөмж нэмэх, засах, устгах",
			OwnScope:     true,
			DefaultRoles: []string{"manager", "user:own"},
		},
	}
}

func (m *Module) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{
			ID:     "devices.list",
			Label:  "Төхөөрөмжүүд",
			Labels: map[string]string{"en": "Devices"},
			Path:   "/devices",
			Icon:   "device",
			Order:  10,
		},
	}
}

func (m *Module) Migrations() fs.FS { return migrations }

func (m *Module) RegisterRoutes(r chi.Router, deps nexus.Deps) {
	h := &handler{deps: deps}
	r.With(nexus.RequirePermission(deps.Perms, "devices.read")).Get("/", h.list)
	r.With(nexus.RequirePermission(deps.Perms, "devices.manage")).Post("/", h.create)
	r.With(nexus.RequirePermission(deps.Perms, "devices.manage")).Put("/{id}", h.update)
	r.With(nexus.RequirePermission(deps.Perms, "devices.manage")).Delete("/{id}", h.remove)
}

type handler struct{ deps nexus.Deps }

type deviceRow struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Serial    string `json:"serial"`
	Status    string `json:"status"`
	Note      string `json:"note"`
	CreatedBy string `json:"created_by"`
	OwnerName string `json:"owner_name"`
	CreatedAt string `json:"created_at"`
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	q := "%" + strings.TrimSpace(r.URL.Query().Get("q")) + "%"
	rows, err := h.deps.DB.Query(r.Context(), `
		SELECT d.id, d.name, d.kind, d.serial, d.status, d.note,
		       d.created_by::text, coalesce(u.name, ''), d.created_at::text
		  FROM devices d LEFT JOIN users u ON u.id = d.created_by
		 WHERE d.tenant_id = $1::uuid
		   AND (d.name ILIKE $2::text OR d.serial ILIKE $2::text OR d.kind ILIKE $2::text)
		 ORDER BY d.created_at DESC LIMIT 200`,
		nexus.TenantID(r.Context()), q)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "devices query failed")
		return
	}
	defer rows.Close()
	out := []deviceRow{}
	for rows.Next() {
		var d deviceRow
		if err := rows.Scan(&d.ID, &d.Name, &d.Kind, &d.Serial, &d.Status, &d.Note,
			&d.CreatedBy, &d.OwnerName, &d.CreatedAt); err != nil {
			nexus.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, d)
	}
	if err := rows.Err(); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "devices query failed")
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]any{"devices": out, "scope": nexus.Scope(r.Context())})
}

type deviceInput struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Serial string `json:"serial"`
	Status string `json:"status"`
	Note   string `json:"note"`
}

func (in *deviceInput) valid() bool {
	in.Name = strings.TrimSpace(in.Name)
	in.Serial = strings.TrimSpace(in.Serial)
	if in.Status == "" {
		in.Status = "active"
	}
	switch in.Status {
	case "active", "repair", "lost", "retired":
	default:
		return false
	}
	return in.Name != "" && in.Serial != ""
}

func (h *handler) create(w http.ResponseWriter, r *http.Request) {
	var in deviceInput
	if !nexus.Decode(w, r, &in) {
		return
	}
	if !in.valid() {
		nexus.Error(w, http.StatusBadRequest, "нэр, сериал шаардлагатай; статус буруу")
		return
	}
	var id string
	err := h.deps.DB.QueryRow(r.Context(), `
		INSERT INTO devices (tenant_id, name, kind, serial, status, note, created_by)
		VALUES ($1::uuid, $2::varchar(120), $3::varchar(64), $4::varchar(120),
		        $5::varchar(16), $6::varchar(500), $7::uuid)
		RETURNING id`,
		nexus.TenantID(r.Context()), in.Name, in.Kind, in.Serial, in.Status, in.Note,
		nexus.UserID(r.Context())).Scan(&id)
	if err != nil {
		nexus.DBError(w, err, "сериал давхардаж байна")
		return
	}
	h.deps.Audit.Record(r.Context(), "devices.create", in.Serial, map[string]any{"name": in.Name})
	nexus.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

// ownFilter — ScopeOwn үед created_by шүүлт (docs/02-rbac.md #3). $n нь
// нэмэлт параметрийн байрлал.
func ownFilter(r *http.Request, n int) (string, []any) {
	if nexus.Scope(r.Context()) == nexus.ScopeOwn {
		return fmt.Sprintf(" AND created_by = $%d::uuid", n), []any{nexus.UserID(r.Context())}
	}
	return "", nil
}

func (h *handler) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in deviceInput
	if !nexus.Decode(w, r, &in) {
		return
	}
	if !in.valid() {
		nexus.Error(w, http.StatusBadRequest, "нэр, сериал шаардлагатай; статус буруу")
		return
	}
	extra, extraArgs := ownFilter(r, 8)
	args := []any{id, nexus.TenantID(r.Context()), in.Name, in.Kind, in.Serial, in.Status, in.Note}
	tag, err := h.deps.DB.Exec(r.Context(), `
		UPDATE devices SET name = $3::varchar(120), kind = $4::varchar(64),
		       serial = $5::varchar(120), status = $6::varchar(16), note = $7::varchar(500),
		       updated_at = now()
		 WHERE id = $1::uuid AND tenant_id = $2::uuid`+extra,
		append(args, extraArgs...)...)
	if err != nil {
		nexus.DBError(w, err, "сериал давхардаж байна")
		return
	}
	if tag.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "олдсонгүй эсвэл таны бүртгэл биш")
		return
	}
	h.deps.Audit.Record(r.Context(), "devices.update", id, nil)
	nexus.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (h *handler) remove(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	extra, extraArgs := ownFilter(r, 3)
	tag, err := h.deps.DB.Exec(r.Context(),
		`DELETE FROM devices WHERE id = $1::uuid AND tenant_id = $2::uuid`+extra,
		append([]any{id, nexus.TenantID(r.Context())}, extraArgs...)...)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "олдсонгүй эсвэл таны бүртгэл биш")
		return
	}
	h.deps.Audit.Record(r.Context(), "devices.delete", id, nil)
	nexus.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
