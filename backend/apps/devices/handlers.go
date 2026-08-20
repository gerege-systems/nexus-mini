package devices

// Devices resource-ийн HTTP handler-ууд. Route↔permission холболт
// module.go-д; энд зөвхөн бизнес логик. Өөр resource нэмэгдвэл (жишээ нь
// тайлан) reports_handlers.go гэж тусдаа файл үүсгэнэ.

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

type handler struct{ deps nexus.Deps }

// GET / — жагсаалт (q хайлттай).
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

// POST / — бүртгэх.
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

// PUT /{id} — засах (own scope-той бол зөвхөн өөрийн мөр).
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

// DELETE /{id} — устгах (own scope-той бол зөвхөн өөрийн мөр).
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
