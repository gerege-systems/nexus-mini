package organisation

import (
	"net/http"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
)

// GET /people — байгууллагын бүх гишүүн, байршилтайгаа (байхгүй бол хоосон).
func (h *handler) listPeople(w http.ResponseWriter, r *http.Request) {
	rows, err := h.deps.DB.Query(r.Context(), `
		SELECT m.id, u.id, u.name, p.department_id::text, coalesce(d.name, ''), coalesce(p.job_title, '')
		  FROM memberships m
		  JOIN users u ON u.id = m.user_id
		  LEFT JOIN org_positions p ON p.membership_id = m.id
		  LEFT JOIN org_departments d ON d.id = p.department_id
		 WHERE m.tenant_id = $1::uuid
		 ORDER BY u.name`, nexus.TenantID(r.Context()))
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "people query failed")
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (personRow, error) {
		var p personRow
		err := row.Scan(&p.MembershipID, &p.UserID, &p.Name, &p.DepartmentID, &p.DepartmentName, &p.JobTitle)
		return p, err
	})
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "people query failed")
		return
	}
	if out == nil {
		out = []personRow{}
	}
	nexus.JSON(w, http.StatusOK, map[string]any{"people": out})
}

// PUT /people/{membership_id} — хэлтэс + албан тушаал (upsert). Гишүүнчлэл
// энэ tenant-ийнх мөн эсэхийг INSERT…SELECT-ээр шалгана (RLS давхар).
func (h *handler) updatePosition(w http.ResponseWriter, r *http.Request) {
	mid, ok := nexus.UUIDParam(w, r, "membership_id")
	if !ok {
		return
	}
	var in positionInput
	if !nexus.Decode(w, r, &in) {
		return
	}
	if !in.valid() {
		nexus.Error(w, http.StatusBadRequest, "албан тушаал ≤120 тэмдэгт")
		return
	}
	tag, err := h.deps.DB.Exec(r.Context(), `
		INSERT INTO org_positions (membership_id, tenant_id, department_id, job_title)
		SELECT m.id, m.tenant_id, $3::uuid, $4::varchar(120)
		  FROM memberships m WHERE m.id = $1::uuid AND m.tenant_id = $2::uuid
		ON CONFLICT (membership_id) DO UPDATE SET
		  department_id = excluded.department_id, job_title = excluded.job_title, updated_at = now()`,
		mid, nexus.TenantID(r.Context()), in.DepartmentID, in.JobTitle)
	if err != nil {
		// FK эсвэл same-tenant триггер (23514) — хоёулаа хэрэглэгчийн алдаа.
		if nexus.IsFKViolation(err) || isCheckViolation(err) {
			nexus.Error(w, http.StatusBadRequest, "хэлтэс олдсонгүй эсвэл энэ байгууллагынх биш")
			return
		}
		nexus.Error(w, http.StatusInternalServerError, "position save failed")
		return
	}
	if tag.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "гишүүн олдсонгүй")
		return
	}
	h.deps.Audit.Record(r.Context(), "organisation.position.update", mid,
		map[string]any{"department_id": in.DepartmentID, "job_title": in.JobTitle})
	nexus.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
