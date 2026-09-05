package handlers

import (
	"net/http"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/jackc/pgx/v5"
)

// PUT /api/me — өөрийн нэрээ солино.
func (h *Auth) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	if p.ImpersonatedBy != "" {
		httpx.Error(w, http.StatusForbidden, "нэрийн өмнөөс орсон session-д профайл өөрчлөхгүй")
		return
	}
	var in struct {
		Name string `json:"name"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		httpx.Error(w, http.StatusBadRequest, "нэр хоосон байж болохгүй")
		return
	}
	if _, err := h.DB.Exec(r.Context(),
		`UPDATE users SET name = $2::varchar(120) WHERE id = $1::uuid`,
		p.UserID, in.Name); err != nil {
		httpx.Error(w, http.StatusInternalServerError, "хадгалж чадсангүй")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// POST /api/me/password — одоогийн нууц үгээ баталгаажуулж шинэчилнэ;
// бусад төхөөрөмжийн session-уудыг хүчингүй болгоно.
func (h *Auth) ChangePassword(w http.ResponseWriter, r *http.Request) {
	p, _ := auth.PrincipalFrom(r.Context())
	if p.ImpersonatedBy != "" {
		httpx.Error(w, http.StatusForbidden, "нэрийн өмнөөс орсон session-д профайл өөрчлөхгүй")
		return
	}
	var in struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !httpx.Decode(w, r, &in) {
		return
	}
	if err := password.Validate(in.New); err != nil {
		httpx.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	var uid, hash, name string
	var isAdmin bool
	err := h.Pool.QueryRow(r.Context(),
		`SELECT id, password_hash, name, platform_admin FROM auth_user_by_email($1::varchar(255))`,
		p.Email).Scan(&uid, &hash, &name, &isAdmin)
	if err != nil || !password.Verify(in.Current, hash) {
		httpx.Error(w, http.StatusForbidden, "одоогийн нууц үг буруу")
		return
	}
	newHash, err := password.Hash(in.New)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "hash failed")
		return
	}
	err = h.DB.Tx(r.Context(), func(tx pgx.Tx) error {
		if _, err := tx.Exec(r.Context(),
			`UPDATE users SET password_hash = $2::varchar(255), must_change_password = false WHERE id = $1::uuid`,
			p.UserID, newHash); err != nil {
			return err
		}
		// Бусад төхөөрөмж дээрх session-уудыг унтраана — энэ session үлдэнэ.
		_, err := tx.Exec(r.Context(),
			`DELETE FROM sessions WHERE user_id = $1::uuid AND id <> $2::uuid`,
			p.UserID, p.SessionID)
		return err
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "солиход алдаа гарлаа")
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}
