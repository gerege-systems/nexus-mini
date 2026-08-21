package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	errLastAdmin   = errors.New("last admin")
	errUnknownRole = errors.New("unknown role")
)

// assignRoles — гишүүнчлэлд role-уудыг оноож, үл мэдэх код байвал алдаа
// (өмнө нь чимээгүй алгасаад 201 буцдаг байсан).
// mine — оноож буй хүний grants: role (implies гинжтэйгээ) олгодог бүх
// permission түүнд багтаагүй бол татгалзана (members.manage-тэй хүн өөрийгөө
// admin болгох зэрэг дээшлүүлэлтээс хамгаална).
func assignRoles(r *http.Request, tx pgx.Tx, membershipID, tenantID string, codes []string, mine map[string]nexus.Grant) error {
	if _, err := tx.Exec(r.Context(),
		`DELETE FROM membership_roles WHERE membership_id = $1::uuid`, membershipID); err != nil {
		return err
	}
	for _, code := range codes {
		if err := roleWithinGrants(r.Context(), tx, tenantID, code, mine); err != nil {
			return err
		}
		tag, err := tx.Exec(r.Context(), `
			INSERT INTO membership_roles (membership_id, role_id)
			SELECT $1::uuid, r.id FROM roles r
			 WHERE r.tenant_id = $2::uuid AND r.code = $3::varchar(64)`,
			membershipID, tenantID, code)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: %s", errUnknownRole, code)
		}
	}
	return nil
}

// roleWithinGrants — role (implies гинжээр өвлөгдсөнийг оруулаад) олгодог
// permission бүр `mine`-д (хүссэн scope-оор) байгаа эсэх.
func roleWithinGrants(ctx context.Context, tx pgx.Tx, tenantID, roleCode string, mine map[string]nexus.Grant) error {
	rows, err := tx.Query(ctx, `
		WITH RECURSIVE chain AS (
		  SELECT id, implies, 0 AS depth FROM roles
		   WHERE tenant_id = $1::uuid AND code = $2::varchar(64)
		  UNION ALL
		  SELECT r.id, r.implies, c.depth + 1 FROM chain c
		    JOIN roles r ON r.tenant_id = $1::uuid AND r.code = c.implies
		   WHERE c.depth < 5)
		SELECT rp.permission_code, rp.scope
		  FROM (SELECT DISTINCT id FROM chain) c
		  JOIN role_permissions rp ON rp.role_id = c.id`, tenantID, roleCode)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var code, scope string
		if err := rows.Scan(&code, &scope); err != nil {
			return err
		}
		if !grantCovers(mine, code, scope) {
			return grantError{"өөрт байхгүй эрх олгодог role оноох боломжгүй: " + roleCode + " (" + code + ")"}
		}
	}
	return rows.Err()
}

// grantCovers — mine нь code-ийг хүссэн scope-оор эзэмшиж байна уу.
func grantCovers(mine map[string]nexus.Grant, code, scope string) bool {
	g, ok := mine[code]
	return ok && g.Allowed && (scope != "all" || g.Scope == nexus.ScopeAll)
}

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
	err := h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		if in.Implies != "" {
			// implies:"admin" гэх мэтээр өвлөлтөөр эрх дээшлүүлэхээс хамгаална.
			mine, err := h.Perms.UserGrants(r.Context(), tenantID, nexus.UserID(r.Context()))
			if err != nil {
				return err
			}
			if err := roleWithinGrants(r.Context(), tx, tenantID, in.Implies, mine); err != nil {
				return err
			}
		}
		return tx.QueryRow(r.Context(),
			`INSERT INTO roles (tenant_id, code, name, implies)
			 VALUES ($1::uuid, $2::varchar(64), $3::varchar(120), $4::varchar(64)) RETURNING id`,
			tenantID, in.Code, in.Name, implies).Scan(&id)
	})
	var ge grantError
	if errors.As(err, &ge) {
		httpx.Error(w, http.StatusBadRequest, ge.msg)
		return
	}
	if nexus.IsFKViolation(err) {
		httpx.Error(w, http.StatusBadRequest, "өвлөх role олдсонгүй")
		return
	}
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
	// Эрх дээшлүүлэхээс хамгаална: оноож буй хүн тухайн permission-ийг
	// өөрөө (хүссэн scope-оороо) эзэмшсэн байх ёстой.
	mine, err := h.Perms.UserGrants(r.Context(), tenantID, nexus.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "grants lookup failed")
		return
	}
	err = h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		// Role энэ tenant-ийнх мөн үү — RLS хамгаална, гэхдээ ил шалгавал
		// 404 ба 500-г ялгаж чадна.
		var roleCode string
		if err := tx.QueryRow(r.Context(),
			`SELECT code FROM roles WHERE id = $1::uuid AND tenant_id = $2::uuid`,
			roleID, tenantID).Scan(&roleCode); err != nil {
			return err
		}
		if roleCode == "admin" {
			return grantError{"admin role-ийн оноолт автомат — гараар засахгүй"}
		}
		// Байсан оноолтыг (тэр хэвээр нь эсвэл нарийсгаж) үлдээхэд өөрийн
		// эрх шаардахгүй — зөвхөн ШИНЭ/өргөссөн оноолтод л шаардана.
		existing := map[string]string{}
		erows, err := tx.Query(r.Context(),
			`SELECT permission_code, scope FROM role_permissions WHERE role_id = $1::uuid`, roleID)
		if err != nil {
			return err
		}
		for erows.Next() {
			var c, sc string
			if err := erows.Scan(&c, &sc); err != nil {
				erows.Close()
				return err
			}
			existing[c] = sc
		}
		erows.Close()
		if err := erows.Err(); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(),
			`DELETE FROM role_permissions WHERE role_id = $1::uuid`, roleID); err != nil {
			return err
		}
		for code, scope := range in.Grants {
			if scope != "own" {
				scope = "all"
			}
			kept := existing[code] == "all" || existing[code] == scope
			if !kept && !grantCovers(mine, code, scope) {
				return grantError{"өөрт байхгүй эрх оноох боломжгүй: " + code}
			}
			if scope == "own" {
				// Каталог own_scope зарлаагүй permission-д "own" өгвөл модуль
				// шүүдэггүй тул чимээгүй бүрэн эрх болно — хориглоно.
				var ownScope bool
				if err := tx.QueryRow(r.Context(),
					`SELECT own_scope FROM permissions WHERE code = $1::varchar(128)`,
					code).Scan(&ownScope); err != nil {
					return err
				}
				if !ownScope {
					return grantError{"энэ permission own scope дэмждэггүй: " + code}
				}
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
	var ge grantError
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.Error(w, http.StatusNotFound, "role олдсонгүй")
		return
	}
	if errors.As(err, &ge) {
		httpx.Error(w, http.StatusBadRequest, ge.msg)
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

// grantError — оноолтын дүрмийн зөрчил (клиентэд харуулж болох мессеж).
type grantError struct{ msg string }

func (e grantError) Error() string { return e.msg }

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

// GET /api/members/lookup?email= — имэйл бүртгэлтэй эсэх, нэр, энэ
// tenant-ийн гишүүн эсэх. Зөвхөн core.members.manage эрхтэйд нээлттэй тул
// имэйл тоолох (enumeration) эрсдэл tenant админаар хязгаарлагдана.
func (h *RBACH) LookupMember(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("email")))
	if !emailRe.MatchString(email) {
		httpx.Error(w, http.StatusBadRequest, "зөв имэйл шаардлагатай")
		return
	}
	var userID, hash, name string
	var isAdmin bool
	err := h.Pool.QueryRow(r.Context(),
		`SELECT id, password_hash, name, platform_admin FROM auth_user_by_email($1::varchar(255))`,
		email).Scan(&userID, &hash, &name, &isAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		httpx.JSON(w, http.StatusOK, map[string]any{"exists": false})
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	// memberships RLS-тэй тул tenant GUC тавьдаг h.DB-ээр (h.Pool-оор бол
	// мөр харагдахгүй, үргэлж false гарна).
	var member bool
	if err := h.DB.QueryRow(r.Context(),
		`SELECT EXISTS (SELECT 1 FROM memberships WHERE user_id = $1::uuid)`,
		userID).Scan(&member); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "lookup failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"exists": true, "name": name, "member": member})
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
	mine, err := h.Perms.UserGrants(r.Context(), tenantID, nexus.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "grants lookup failed")
		return
	}
	err = h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		var memberID string
		if err := tx.QueryRow(r.Context(),
			`INSERT INTO memberships (tenant_id, user_id) VALUES ($1::uuid, $2::uuid)
			 ON CONFLICT (tenant_id, user_id) DO UPDATE SET user_id = excluded.user_id
			 RETURNING id`, tenantID, userID).Scan(&memberID); err != nil {
			return err
		}
		return assignRoles(r, tx, memberID, tenantID, in.Roles, mine)
	})
	var ge grantError
	if errors.As(err, &ge) {
		httpx.Error(w, http.StatusBadRequest, ge.msg)
		return
	}
	if errors.Is(err, errUnknownRole) {
		httpx.Error(w, http.StatusBadRequest, "үл мэдэх role код байна")
		return
	}
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
	mine, err := h.Perms.UserGrants(r.Context(), tenantID, nexus.UserID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "grants lookup failed")
		return
	}
	err = h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		var ok bool
		if err := tx.QueryRow(r.Context(),
			`SELECT EXISTS (SELECT 1 FROM memberships WHERE id = $1::uuid AND tenant_id = $2::uuid)`,
			membershipID, tenantID).Scan(&ok); err != nil {
			return err
		}
		if !ok {
			return pgx.ErrNoRows
		}
		if err := assignRoles(r, tx, membershipID, tenantID, in.Roles, mine); err != nil {
			return err
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
	var ge grantError
	if errors.As(err, &ge) {
		httpx.Error(w, http.StatusBadRequest, ge.msg)
		return
	}
	if errors.Is(err, errUnknownRole) {
		httpx.Error(w, http.StatusBadRequest, "үл мэдэх role код байна")
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
