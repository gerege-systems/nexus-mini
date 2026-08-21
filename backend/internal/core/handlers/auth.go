// Package handlers — HTTP handler-ууд. Платформын package-уудаас тусдаа
// байдаг нь import cycle-ээс сэргийлдэг (docs/01-lessons.md #7).
package handlers

import (
	"context"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/identity"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/tenantstate"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Auth struct {
	Pool  *pgxpool.Pool // app pool — definer функцууд
	DB    nexus.DB
	Svc   *auth.Service
	Audit *audit.Recorder
	Perms *rbac.Store
	State *tenantstate.Store
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

// POST /api/signup — landing-аас: хэрэглэгч + байгууллага хамт бүртгэнэ.
func (h *Auth) Signup(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string `json:"name"`
		Email      string `json:"email"`
		Password   string `json:"password"`
		TenantName string `json:"tenant_name"`
		TenantSlug string `json:"tenant_slug"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.TenantName = strings.TrimSpace(in.TenantName)
	in.TenantSlug = strings.ToLower(strings.TrimSpace(in.TenantSlug))
	if in.Name == "" || !emailRe.MatchString(in.Email) || len(in.Password) < 8 ||
		in.TenantName == "" || in.TenantSlug == "" {
		httpx.Error(w, http.StatusBadRequest, "бүх талбарыг зөв бөглөнө үү (нууц үг 8+)")
		return
	}
	hash, err := password.Hash(in.Password)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "hash failed")
		return
	}
	// Хэрэглэгч + байгууллага НЭГ гүйлгээнд — slug давхардвал орфан
	// хэрэглэгч үлдэж "имэйл бүртгэлтэй" гацаа үүсгэдэг байсан (аудит).
	var uid, tenantID string
	err = h.DB.Tx(identity.With(r.Context(), "", ""), func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(),
			`SELECT auth_signup($1::varchar(255), $2::varchar(255), $3::varchar(120))`,
			in.Email, hash, in.Name).Scan(&uid); err != nil {
			return err
		}
		var err error
		tenantID, err = createTenantTx(r.Context(), tx, uid, in.TenantName, in.TenantSlug)
		return err
	})
	if err != nil {
		if nexus.IsUniqueViolation(err) {
			httpx.Error(w, http.StatusConflict, "имэйл эсвэл байгууллагын slug бүртгэлтэй байна")
			return
		}
		log.Printf("signup: %v", err)
		httpx.Error(w, http.StatusInternalServerError, "бүртгэл амжилтгүй боллоо")
		return
	}
	sid, err := h.Svc.StartSession(r.Context(), w, uid)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "session failed")
		return
	}
	// Шинэ байгууллагаа шууд идэвхжүүлнэ — хэрэглэгч орж ирээд store-оо харна.
	if _, err := h.Svc.SetTenant(r.Context(), sid, tenantID); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "session failed")
		return
	}
	h.Audit.RecordAs(r.Context(), tenantID, uid, "tenant.create", in.TenantSlug,
		map[string]any{"name": in.TenantName})
	httpx.JSON(w, http.StatusCreated, map[string]string{"user_id": uid, "tenant_id": tenantID})
}

// createTenantTx — tenant + гишүүнчлэл + default role-ууд + admin оноолт,
// өгөгдсөн гүйлгээн дотор. Гүйлгээний RLS context-оо өөрөө SET LOCAL хийдэг
// тул дуудагчийн ctx identity ямар ч байж болно.
func createTenantTx(ctx context.Context, tx pgx.Tx, userID, name, slug string) (string, error) {
	// Хэрэглэгчийн context — tenants INSERT бодлогод app_user_id() хэрэгтэй.
	if _, err := tx.Exec(ctx,
		`SELECT set_config('app.user_id', $1::text, true)`, userID); err != nil {
		return "", err
	}
	// Tenant-ийн id-г урьдчилан гаргаж context-оо ЭХЭЛЖ тохируулна:
	// INSERT ... RETURNING нь SELECT бодлого шаарддаг тул шинэ tenant
	// өөрийн app.tenant_id-тэй таарч байж л буцаж харагдана.
	var tenantID string
	if err := tx.QueryRow(ctx, `SELECT gen_random_uuid()`).Scan(&tenantID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`SELECT set_config('app.tenant_id', $1::text, true)`, tenantID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO tenants (id, slug, name) VALUES ($1::uuid, $2::varchar(64), $3::varchar(160))`,
		tenantID, slug, name); err != nil {
		return "", err
	}
	var memberID string
	if err := tx.QueryRow(ctx,
		`INSERT INTO memberships (tenant_id, user_id) VALUES ($1::uuid, $2::uuid) RETURNING id`,
		tenantID, userID).Scan(&memberID); err != nil {
		return "", err
	}
	adminRoleID, err := rbac.SeedTenantRoles(ctx, tx, tenantID)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO membership_roles (membership_id, role_id) VALUES ($1::uuid, $2::uuid)`,
		memberID, adminRoleID); err != nil {
		return "", err
	}
	// Платформын core permission-уудын default оноолт.
	if err := rbac.GrantOnInstall(ctx, tx, tenantID, rbac.CorePermissions()); err != nil {
		return "", err
	}
	return tenantID, nil
}

// POST /api/tenants — нэвтэрсэн хэрэглэгч шинэ байгууллага үүсгэнэ.
func (h *Auth) CreateTenant(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	var in struct {
		Name string `json:"name"`
		Slug string `json:"slug"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Slug = strings.ToLower(strings.TrimSpace(in.Slug))
	if in.Name == "" || in.Slug == "" {
		httpx.Error(w, http.StatusBadRequest, "нэр ба slug шаардлагатай")
		return
	}
	var tenantID string
	err := h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		var err error
		tenantID, err = createTenantTx(r.Context(), tx, p.UserID, in.Name, in.Slug)
		return err
	})
	if err != nil {
		httpx.DBError(w, err, "slug давхардаж байна")
		return
	}
	h.Audit.RecordAs(r.Context(), tenantID, p.UserID, "tenant.create", in.Slug,
		map[string]any{"name": in.Name})
	httpx.JSON(w, http.StatusCreated, map[string]string{"tenant_id": tenantID})
}

// GET /api/auth/handover?token= — админ панелийн impersonation token-ийг
// portal домэйн дээр session болгоно (нэг удаа, 60с). Дараа нь dashboard.
func (h *Auth) Handover(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		httpx.Error(w, http.StatusBadRequest, "token шаардлагатай")
		return
	}
	ok, err := h.Svc.ConsumeHandover(r.Context(), w, token)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "handover failed")
		return
	}
	if !ok {
		httpx.Error(w, http.StatusUnauthorized, "handover token хүчингүй эсвэл хугацаа дууссан")
		return
	}
	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// POST /api/login
func (h *Auth) Login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	var uid, hash, name string
	var isAdmin bool
	err := h.Pool.QueryRow(r.Context(),
		`SELECT id, password_hash, name, platform_admin FROM auth_user_by_email($1::varchar(255))`,
		strings.ToLower(strings.TrimSpace(in.Email))).Scan(&uid, &hash, &name, &isAdmin)
	if err == pgx.ErrNoRows || (err == nil && !password.Verify(in.Password, hash)) {
		httpx.Error(w, http.StatusUnauthorized, "имэйл эсвэл нууц үг буруу")
		return
	}
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "login failed")
		return
	}
	if _, err := h.Svc.StartSession(r.Context(), w, uid); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "session failed")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"user_id": uid, "name": name})
}

// POST /api/logout
func (h *Auth) Logout(w http.ResponseWriter, r *http.Request) {
	if p, ok := h.Svc.Resolve(r.Context(), r); ok && p.TenantID != "" {
		h.Audit.RecordAs(r.Context(), p.TenantID, p.UserID, "auth.logout", p.Email, nil)
	}
	h.Svc.EndSession(r.Context(), w, r)
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// GET /api/me — хэрэглэгч + байгууллагууд + идэвхтэй tenant + эрхүүд.
func (h *Auth) Me(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())

	type tenantRow struct {
		ID   string `json:"id"`
		Slug string `json:"slug"`
		Name string `json:"name"`
	}
	var tenants []tenantRow
	rows, err := h.DB.Query(r.Context(), `
		SELECT t.id, t.slug, t.name FROM tenants t
		 JOIN memberships m ON m.tenant_id = t.id
		WHERE m.user_id = $1::uuid ORDER BY t.name`, p.UserID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenants query failed")
		return
	}
	defer rows.Close()
	for rows.Next() {
		var t tenantRow
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name); err != nil {
			httpx.Error(w, http.StatusInternalServerError, "scan failed")
			return
		}
		tenants = append(tenants, t)
	}
	if err := rows.Err(); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "tenants query failed")
		return
	}
	if tenants == nil {
		tenants = []tenantRow{}
	}

	perms := map[string]string{}
	if p.TenantID != "" {
		grants, err := h.Perms.UserGrants(r.Context(), p.TenantID, p.UserID)
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "grants failed")
			return
		}
		for code, g := range grants {
			perms[code] = string(g.Scope)
		}
	}

	var state any
	if p.TenantID != "" && h.State != nil {
		if st, err := h.State.Get(r.Context(), p.TenantID); err == nil {
			state = st
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"tenant_state":    state,
		"impersonated_by": p.ImpersonatedBy,
		"user": map[string]any{
			"id": p.UserID, "name": p.Name, "email": p.Email,
			"platform_admin": p.PlatformAdmin,
		},
		"tenant_id":   p.TenantID,
		"tenants":     tenants,
		"permissions": perms,
	})
}

// POST /api/session/tenant — идэвхтэй байгууллагаа сонгоно.
func (h *Auth) SelectTenant(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	var in struct {
		TenantID string `json:"tenant_id"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	ok, err := h.Svc.SetTenant(r.Context(), p.SessionID, in.TenantID)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "switch failed")
		return
	}
	if !ok {
		httpx.Error(w, http.StatusForbidden, "энэ байгууллагын гишүүн биш")
		return
	}
	if in.TenantID != "" && in.TenantID != p.TenantID {
		// Байгууллагад нэвтэрсэн үйлдлийг тухайн байгууллагын гинжид бичнэ.
		h.Audit.RecordAs(r.Context(), in.TenantID, p.UserID, "auth.login", p.Email, nil)
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
