package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/internal/platform/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/password"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var errLastAdmin = errors.New("last admin")

// RBACH — role, гишүүдийн удирдлага. core.* permission-ууд платформынх
// (модулийн prefix дүрэм модулиудад л үйлчилнэ).
type RBACH struct {
	DB    nexus.DB
	Pool  *pgxpool.Pool
	Perms *rbac.Store
	Audit *audit.Recorder
}

// GET /api/roles — role-ууд + permission оноолтууд.
func (h *RBACH) Roles(w http.ResponseWriter, r *http.Request) {
	tenantID := nexus.TenantID(r.Context())
	rows, err := h.DB.Query(r.Context(), `
		SELECT r.id, r.code, r.name, coalesce(r.implies, ''), r.active,
		       coalesce(jsonb_object_agg(rp.permission_code, rp.scope)
		                FILTER (WHERE rp.permission_code IS NOT NULL), '{}'::jsonb)
		  FROM roles r
		  LEFT JOIN role_permissions rp ON rp.role_id = r.id
		 WHERE r.tenant_id = $1::uuid
		 GROUP BY r.id ORDER BY r.code`, tenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "roles query failed")
		return
	}
	defer rows.Close()
	type roleRow struct {
		ID      string            `json:"id"`
		Code    string            `json:"code"`
		Name    string            `json:"name"`
		Implies string            `json:"implies"`
		Active  bool              `json:"active"`
		Grants  map[string]string `json:"grants"`
	}
	out := []roleRow{}
	for rows.Next() {
		var x roleRow
		if err := rows.Scan(&x.ID, &x.Code, &x.Name, &x.Implies, &x.Active, &x.Grants); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "roles query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"roles": out})
}

// GET /api/permissions — глобал каталог (UI-д оноолтын хүснэгт зурахад).
func (h *RBACH) Permissions(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT code, module_id, name, description, own_scope FROM permissions ORDER BY code`)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "permissions query failed")
		return
	}
	defer rows.Close()
	type permRow struct {
		Code        string `json:"code"`
		ModuleID    string `json:"module_id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		OwnScope    bool   `json:"own_scope"`
	}
	out := []permRow{}
	for rows.Next() {
		var x permRow
		if err := rows.Scan(&x.Code, &x.ModuleID, &x.Name, &x.Description, &x.OwnScope); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "permissions query failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"permissions": out})
}

// POST /api/roles — custom role үүсгэнэ.
func (h *RBACH) CreateRole(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Code    string `json:"code"`
		Name    string `json:"name"`
		Implies string `json:"implies"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Code = strings.ToLower(strings.TrimSpace(in.Code))
	in.Name = strings.TrimSpace(in.Name)
	if in.Code == "" || in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "code ба нэр шаардлагатай")
		return
	}
	var implies any
	if in.Implies != "" {
		implies = in.Implies
	}
	tenantID := nexus.TenantID(r.Context())
	var id string
	err := h.DB.QueryRow(r.Context(),
		`INSERT INTO roles (tenant_id, code, name, implies)
		 VALUES ($1::uuid, $2::varchar(64), $3::varchar(120), $4::varchar(64)) RETURNING id`,
		tenantID, in.Code, in.Name, implies).Scan(&id)
	if err != nil {
		httpx.DBError(w, err, "code давхардаж байна (эсвэл формат буруу)")
		return
	}
	h.Perms.Invalidate(tenantID)
	h.Audit.Record(r.Context(), "role.create", in.Code, map[string]any{"name": in.Name})
	httpx.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

// PUT /api/roles/{id}/grants — role-ийн оноолтыг бүхэлд нь солино.
func (h *RBACH) SetGrants(w http.ResponseWriter, r *http.Request) {
	roleID := chi.URLParam(r, "id")
	var in struct {
		Grants map[string]string `json:"grants"` // code → "all"|"own"
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	tenantID := nexus.TenantID(r.Context())
	err := h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		// Role энэ tenant-ийнх мөн үү — RLS хамгаална, гэхдээ ил шалгавал
		// 404 ба 500-г ялгаж чадна.
		var ok bool
		if err := tx.QueryRow(r.Context(),
			`SELECT EXISTS (SELECT 1 FROM roles WHERE id = $1::uuid AND tenant_id = $2::uuid)`,
			roleID, tenantID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return pgx.ErrNoRows
		}
		if _, err := tx.Exec(r.Context(),
			`DELETE FROM role_permissions WHERE role_id = $1::uuid`, roleID); err != nil {
			return err
		}
		for code, scope := range in.Grants {
			if scope != "own" {
				scope = "all"
			}
			if _, err := tx.Exec(r.Context(),
				`INSERT INTO role_permissions (role_id, permission_code, scope)
				 VALUES ($1::uuid, $2::varchar(128), $3::varchar(3))`,
				roleID, code, scope); err != nil {
				return err
			}
		}
		return nil
	})
	if err == pgx.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "role олдсонгүй")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusBadRequest, "оноолт хадгалагдсангүй (permission код зөв үү?)")
		return
	}
	h.Perms.Invalidate(tenantID)
	h.Audit.Record(r.Context(), "role.grants", roleID, map[string]any{"count": len(in.Grants)})
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// tenantHasAdmin — өөрчлөлтийн ДАРАА tenant-д admin role-тэй гишүүн
// үлдсэн эсэх (tenant lockout-оос сэргийлнэ, аудитын #4).
func tenantHasAdmin(r *http.Request, tx pgx.Tx, tenantID string) (bool, error) {
	var ok bool
	err := tx.QueryRow(r.Context(), `
		SELECT EXISTS (
		  SELECT 1 FROM membership_roles mr
		    JOIN roles ro ON ro.id = mr.role_id AND ro.code = 'admin'
		    JOIN memberships m ON m.id = mr.membership_id
		   WHERE m.tenant_id = $1::uuid)`, tenantID).Scan(&ok)
	return ok, err
}

// GET /api/members — гишүүд role-уудтайгаа.
func (h *RBACH) Members(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT m.id, u.id, u.name, u.email,
		       coalesce(array_agg(ro.code ORDER BY ro.code)
		                FILTER (WHERE ro.code IS NOT NULL), '{}')
		  FROM memberships m
		  JOIN users u ON u.id = m.user_id
		  LEFT JOIN membership_roles mr ON mr.membership_id = m.id
		  LEFT JOIN roles ro ON ro.id = mr.role_id
		 WHERE m.tenant_id = $1::uuid
		 GROUP BY m.id, u.id ORDER BY u.name`, nexus.TenantID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "members query failed")
		return
	}
	defer rows.Close()
	type memberRow struct {
		MembershipID string   `json:"membership_id"`
		UserID       string   `json:"user_id"`
		Name         string   `json:"name"`
		Email        string   `json:"email"`
		Roles        []string `json:"roles"`
	}
	out := []memberRow{}
	for rows.Next() {
		var x memberRow
		if err := rows.Scan(&x.MembershipID, &x.UserID, &x.Name, &x.Email, &x.Roles); err != nil {
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

// POST /api/members — гишүүн нэмнэ. Имэйл нь бүртгэлтэй бол шууд
// гишүүнчлэл үүсгэнэ; үгүй бол түр нууц үгтэй хэрэглэгч үүсгэнэ (үе 1 —
// имэйл илгээлт üе 3-т).
func (h *RBACH) AddMember(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string   `json:"email"`
		Name     string   `json:"name"`
		Password string   `json:"password"`
		Roles    []string `json:"roles"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if !emailRe.MatchString(in.Email) {
		httpx.Error(w, http.StatusBadRequest, "зөв имэйл шаардлагатай")
		return
	}
	if len(in.Roles) == 0 {
		in.Roles = []string{"user"}
	}

	// Бүртгэлтэй эсэхийг pre-auth definer функцээр шалгана (RLS-ийн улмаас
	// өөр tenant-ийн хэрэглэгч бидэнд харагддаггүй — сургамж #1).
	var userID string
	var existingHash, existingName string
	var isAdmin bool
	err := h.Pool.QueryRow(r.Context(),
		`SELECT id, password_hash, name, platform_admin FROM auth_user_by_email($1::varchar(255))`,
		in.Email).Scan(&userID, &existingHash, &existingName, &isAdmin)
	if err == pgx.ErrNoRows {
		in.Name = strings.TrimSpace(in.Name)
		if in.Name == "" || len(in.Password) < 8 {
			httpx.Error(w, http.StatusBadRequest, "шинэ хэрэглэгчид нэр ба 8+ тэмдэгт түр нууц үг өгнө")
			return
		}
		hash, herr := password.Hash(in.Password)
		if herr != nil {
			httpx.Error(w, http.StatusInternalServerError, "hash failed")
			return
		}
		if err := h.Pool.QueryRow(r.Context(),
			`SELECT auth_signup($1::varchar(255), $2::varchar(255), $3::varchar(120))`,
			in.Email, hash, in.Name).Scan(&userID); err != nil {
			httpx.Error(w, http.StatusConflict, "хэрэглэгч үүсгэж чадсангүй")
			return
		}
	} else if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "lookup failed")
		return
	}

	tenantID := nexus.TenantID(r.Context())
	err = h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		var memberID string
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO memberships (tenant_id, user_id) VALUES ($1::uuid, $2::uuid)
			 ON CONFLICT (tenant_id, user_id) DO UPDATE SET user_id = excluded.user_id
			 RETURNING id`, tenantID, userID).Scan(&memberID); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(),
			`DELETE FROM membership_roles WHERE membership_id = $1::uuid`, memberID); err != nil {
			return err
		}
		for _, code := range in.Roles {
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO membership_roles (membership_id, role_id)
				SELECT $1::uuid, r.id FROM roles r
				 WHERE r.tenant_id = $2::uuid AND r.code = $3::varchar(64)`,
				memberID, tenantID, code); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "member add failed")
		return
	}
	h.Perms.Invalidate(tenantID)
	h.Audit.Record(r.Context(), "member.add", in.Email, map[string]any{"roles": in.Roles})
	httpx.JSON(w, http.StatusCreated, map[string]string{"user_id": userID})
}

// PUT /api/members/{id}/roles — гишүүний role-уудыг солино.
func (h *RBACH) SetMemberRoles(w http.ResponseWriter, r *http.Request) {
	membershipID := chi.URLParam(r, "id")
	var in struct {
		Roles []string `json:"roles"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	tenantID := nexus.TenantID(r.Context())
	err := h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		var ok bool
		if err := tx.QueryRow(r.Context(),
			`SELECT EXISTS (SELECT 1 FROM memberships WHERE id = $1::uuid AND tenant_id = $2::uuid)`,
			membershipID, tenantID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return pgx.ErrNoRows
		}
		if _, err := tx.Exec(r.Context(),
			`DELETE FROM membership_roles WHERE membership_id = $1::uuid`, membershipID); err != nil {
			return err
		}
		for _, code := range in.Roles {
			if _, err := tx.Exec(r.Context(), `
				INSERT INTO membership_roles (membership_id, role_id)
				SELECT $1::uuid, r.id FROM roles r
				 WHERE r.tenant_id = $2::uuid AND r.code = $3::varchar(64)`,
				membershipID, tenantID, code); err != nil {
				return err
			}
		}
		ok, err := tenantHasAdmin(r, tx, tenantID)
		if err != nil {
			return err
		}
		if !ok {
			return errLastAdmin
		}
		return nil
	})
	if err == errLastAdmin {
		httpx.Error(w, http.StatusConflict, "байгууллагад дор хаяж нэг админ үлдэх ёстой")
		return
	}
	if err == pgx.ErrNoRows {
		httpx.Error(w, http.StatusNotFound, "гишүүн олдсонгүй")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "roles update failed")
		return
	}
	h.Perms.Invalidate(tenantID)
	h.Audit.Record(r.Context(), "member.roles", membershipID, map[string]any{"roles": in.Roles})
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// DELETE /api/members/{id}
func (h *RBACH) RemoveMember(w http.ResponseWriter, r *http.Request) {
	membershipID := chi.URLParam(r, "id")
	tenantID := nexus.TenantID(r.Context())
	var removed bool
	err := h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		tag, err := tx.Exec(r.Context(),
			`DELETE FROM memberships WHERE id = $1::uuid AND tenant_id = $2::uuid AND user_id <> $3::uuid`,
			membershipID, tenantID, nexus.UserID(r.Context()))
		if err != nil {
			return err
		}
		removed = tag.RowsAffected() > 0
		if !removed {
			return nil
		}
		ok, err := tenantHasAdmin(r, tx, tenantID)
		if err != nil {
			return err
		}
		if !ok {
			return errLastAdmin
		}
		return nil
	})
	if err == errLastAdmin {
		httpx.Error(w, http.StatusConflict, "байгууллагад дор хаяж нэг админ үлдэх ёстой")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "remove failed")
		return
	}
	if !removed {
		httpx.Error(w, http.StatusNotFound, "гишүүн олдсонгүй (өөрийгөө хасаж болохгүй)")
		return
	}
	h.Perms.Invalidate(tenantID)
	h.Audit.Record(r.Context(), "member.remove", membershipID, nil)
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
