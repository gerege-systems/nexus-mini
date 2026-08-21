package organisation

import (
	"errors"
	"net/http"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

// GET /departments — хавтгай жагсаалт (parent_id-тэй); модыг клиент босгоно.
func (h *handler) listDepartments(w http.ResponseWriter, r *http.Request) {
	rows, err := h.deps.DB.Query(r.Context(), `
		SELECT d.id, d.code, d.name, d.parent_id::text, d.manager_membership_id::text,
		       coalesce(u.name, ''), d.active,
		       (SELECT count(*) FROM org_positions p WHERE p.department_id = d.id)
		  FROM org_departments d
		  LEFT JOIN memberships m ON m.id = d.manager_membership_id
		  LEFT JOIN users u ON u.id = m.user_id
		 WHERE d.tenant_id = $1::uuid
		 ORDER BY d.code`, nexus.TenantID(r.Context()))
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "departments query failed")
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (departmentRow, error) {
		var d departmentRow
		err := row.Scan(&d.ID, &d.Code, &d.Name, &d.ParentID, &d.ManagerID, &d.ManagerName, &d.Active, &d.People)
		return d, err
	})
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "departments query failed")
		return
	}
	if out == nil {
		out = []departmentRow{}
	}
	nexus.JSON(w, http.StatusOK, map[string]any{"departments": out})
}

var errCycle = errors.New("cycle")

// checkParent — parent нь энэ tenant-ийнх, мөн self/үр сад биш (мөчлөггүй).
// Гүн ≤ 32 — санамсаргүй урт гинжээс хамгаална.
func checkParent(r *http.Request, tx pgx.Tx, selfID string, parentID *string) error {
	if parentID == nil {
		return nil
	}
	cur := *parentID
	for i := 0; i < 32 && cur != ""; i++ {
		if cur == selfID {
			return errCycle
		}
		var next *string
		if err := tx.QueryRow(r.Context(),
			`SELECT parent_id::text FROM org_departments WHERE id = $1::uuid AND tenant_id = $2::uuid`,
			cur, nexus.TenantID(r.Context())).Scan(&next); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return pgx.ErrNoRows // parent энэ tenant-д байхгүй
			}
			return err
		}
		if next == nil {
			return nil
		}
		cur = *next
	}
	return errCycle
}

func (h *handler) createDepartment(w http.ResponseWriter, r *http.Request) {
	var in departmentInput
	if !nexus.Decode(w, r, &in) {
		return
	}
	if !in.valid() {
		nexus.Error(w, http.StatusBadRequest, "код (≤32), нэр (≤120) шаардлагатай")
		return
	}
	var id string
	err := h.deps.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		if err := checkParent(r, tx, "", in.ParentID); err != nil {
			return err
		}
		return tx.QueryRow(r.Context(), `
			INSERT INTO org_departments (tenant_id, code, name, parent_id, manager_membership_id)
			VALUES ($1::uuid, $2::varchar(32), $3::varchar(120), $4::uuid, $5::uuid) RETURNING id`,
			nexus.TenantID(r.Context()), in.Code, in.Name, in.ParentID, in.ManagerID).Scan(&id)
	})
	if h.deptErr(w, err) {
		return
	}
	h.deps.Audit.Record(r.Context(), "organisation.department.create", in.Code, map[string]any{"name": in.Name})
	nexus.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

func (h *handler) updateDepartment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in departmentInput
	if !nexus.Decode(w, r, &in) {
		return
	}
	if !in.valid() {
		nexus.Error(w, http.StatusBadRequest, "код (≤32), нэр (≤120) шаардлагатай")
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	err := h.deps.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		if err := checkParent(r, tx, id, in.ParentID); err != nil {
			return err
		}
		tag, err := tx.Exec(r.Context(), `
			UPDATE org_departments SET code = $3::varchar(32), name = $4::varchar(120),
			       parent_id = $5::uuid, manager_membership_id = $6::uuid, active = $7::boolean, updated_at = now()
			 WHERE id = $1::uuid AND tenant_id = $2::uuid`,
			id, nexus.TenantID(r.Context()), in.Code, in.Name, in.ParentID, in.ManagerID, active)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errNotFound
		}
		return nil
	})
	if h.deptErr(w, err) {
		return
	}
	h.deps.Audit.Record(r.Context(), "organisation.department.update", id, nil)
	nexus.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

var errNotFound = errors.New("not found")

// deptErr — нийтлэг алдааны зураглал; true = хариу аль хэдийн бичигдсэн.
func (h *handler) deptErr(w http.ResponseWriter, err error) bool {
	switch {
	case err == nil:
		return false
	case errors.Is(err, errCycle):
		nexus.Error(w, http.StatusBadRequest, "дээд нэгж нь өөрөө эсвэл өөрийн харьяа нэгж байж болохгүй")
	case errors.Is(err, pgx.ErrNoRows):
		nexus.Error(w, http.StatusBadRequest, "дээд нэгж олдсонгүй")
	case errors.Is(err, errNotFound):
		nexus.Error(w, http.StatusNotFound, "хэлтэс олдсонгүй")
	case nexus.IsFKViolation(err):
		nexus.Error(w, http.StatusBadRequest, "менежер энэ байгууллагын гишүүн биш")
	default:
		nexus.DBError(w, err, "ийм кодтой хэлтэс аль хэдийн байна")
	}
	return true
}

// DELETE /departments/{id} — харьяа нэгжүүд дээд түвшингүй болно
// (ON DELETE SET NULL), ажилтнууд хэлтэсгүй болно.
func (h *handler) deleteDepartment(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	tag, err := h.deps.DB.Exec(r.Context(),
		`DELETE FROM org_departments WHERE id = $1::uuid AND tenant_id = $2::uuid`,
		id, nexus.TenantID(r.Context()))
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		nexus.Error(w, http.StatusNotFound, "хэлтэс олдсонгүй")
		return
	}
	h.deps.Audit.Record(r.Context(), "organisation.department.delete", id, nil)
	nexus.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
