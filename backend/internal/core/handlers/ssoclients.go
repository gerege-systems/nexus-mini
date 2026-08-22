package handlers

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/oidc"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
)

// OAuth2/OIDC клиентийн бүртгэл — tenant-ийн (core.sso.manage). Secret нэг л
// удаа харуулна (hash хадгална); public клиент (SPA/mobile) secret-гүй, PKCE.

type ssoClientRow struct {
	ID             string    `json:"id"`
	ClientID       string    `json:"client_id"`
	Name           string    `json:"name"`
	Public         bool      `json:"public"`
	RedirectURIs   []string  `json:"redirect_uris"`
	PostLogoutURIs []string  `json:"post_logout_uris"`
	Scopes         string    `json:"scopes"`
	CreatedAt      time.Time `json:"created_at"`
}

type ssoClientInput struct {
	Name           string   `json:"name"`
	Public         bool     `json:"public"`
	RedirectURIs   []string `json:"redirect_uris"`
	PostLogoutURIs []string `json:"post_logout_uris"`
	Scopes         string   `json:"scopes"`
}

func (in *ssoClientInput) valid() string {
	in.Name = strings.TrimSpace(in.Name)
	in.Scopes = strings.TrimSpace(in.Scopes)
	if in.Scopes == "" {
		in.Scopes = "openid profile email"
	}
	if in.Name == "" || len(in.Name) > 120 {
		return "нэр (≤120) шаардлагатай"
	}
	if len(in.RedirectURIs) == 0 || len(in.RedirectURIs) > 10 || len(in.PostLogoutURIs) > 10 {
		return "1..10 redirect URI"
	}
	for _, list := range [][]string{in.RedirectURIs, in.PostLogoutURIs} {
		for i, u := range list {
			list[i] = strings.TrimSpace(u)
			p, err := url.Parse(list[i])
			if err != nil || p.Fragment != "" || len(list[i]) > 500 ||
				!(p.Scheme == "https" || (p.Scheme == "http" && (p.Hostname() == "localhost" || p.Hostname() == "127.0.0.1"))) {
				return "redirect URI нь https (эсвэл http://localhost) байх ёстой: " + list[i]
			}
		}
	}
	for _, s := range strings.Fields(in.Scopes) {
		switch s {
		case "openid", "profile", "email", "tenant", "roles", "offline_access":
		default:
			return "үл мэдэх scope: " + s
		}
	}
	return ""
}

// GET /api/sso-clients
func (h *Auth) SSOClients(w http.ResponseWriter, r *http.Request) {
	rows, err := h.DB.Query(r.Context(), `
		SELECT id, client_id, name, client_secret_hash IS NULL, redirect_uris, post_logout_uris, scopes, created_at
		  FROM oauth_clients WHERE tenant_id = $1::uuid ORDER BY created_at DESC`, nexus.TenantID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "clients query failed")
		return
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (ssoClientRow, error) {
		var c ssoClientRow
		var ru, plu []byte
		err := row.Scan(&c.ID, &c.ClientID, &c.Name, &c.Public, &ru, &plu, &c.Scopes, &c.CreatedAt)
		_ = json.Unmarshal(ru, &c.RedirectURIs)
		_ = json.Unmarshal(plu, &c.PostLogoutURIs)
		if c.PostLogoutURIs == nil {
			c.PostLogoutURIs = []string{}
		}
		return c, err
	})
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "clients query failed")
		return
	}
	if out == nil {
		out = []ssoClientRow{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"clients": out, "issuer": h.Issuer})
}

// POST /api/sso-clients → {client_id, client_secret?} (secret нэг л удаа).
func (h *Auth) CreateSSOClient(w http.ResponseWriter, r *http.Request) {
	var in ssoClientInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if msg := in.valid(); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	clientID := "nx_" + strings.ToLower(strings.ReplaceAll(nexus.TenantID(r.Context())[:8], "-", "")) + "_" + randomID(12)
	var secretPlain string
	var secretHash *string
	if !in.Public {
		plain, hash, err := oidc.NewClientSecret()
		if err != nil {
			httpx.Error(w, http.StatusInternalServerError, "secret failed")
			return
		}
		secretPlain, secretHash = plain, &hash
	}
	ru, _ := json.Marshal(in.RedirectURIs)
	plu, _ := json.Marshal(in.PostLogoutURIs)
	if plu == nil || string(plu) == "null" {
		plu = []byte("[]")
	}
	var id string
	if err := h.DB.QueryRow(r.Context(), `
		INSERT INTO oauth_clients (tenant_id, client_id, client_secret_hash, name, redirect_uris, post_logout_uris, scopes, created_by)
		VALUES ($1::uuid, $2::varchar(64), $3::varchar(255), $4::varchar(120), $5::jsonb, $6::jsonb, $7::varchar(255), $8::uuid)
		RETURNING id`,
		nexus.TenantID(r.Context()), clientID, secretHash, in.Name, ru, plu, in.Scopes, nexus.UserID(r.Context())).Scan(&id); err != nil {
		httpx.DBError(w, err, "client_id давхардлаа")
		return
	}
	h.Audit.Record(r.Context(), "sso.client.create", clientID, map[string]any{"name": in.Name, "public": in.Public})
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id, "client_id": clientID, "client_secret": secretPlain})
}

// PUT /api/sso-clients/{id} — нэр, URI, scope (secret/public төрөл өөрчлөхгүй).
func (h *Auth) UpdateSSOClient(w http.ResponseWriter, r *http.Request) {
	id, ok := nexus.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	var in ssoClientInput
	if !httpx.Decode(w, r, &in) {
		return
	}
	if msg := in.valid(); msg != "" {
		httpx.Error(w, http.StatusBadRequest, msg)
		return
	}
	ru, _ := json.Marshal(in.RedirectURIs)
	plu, _ := json.Marshal(in.PostLogoutURIs)
	if string(plu) == "null" {
		plu = []byte("[]")
	}
	tag, err := h.DB.Exec(r.Context(), `
		UPDATE oauth_clients SET name = $3::varchar(120), redirect_uris = $4::jsonb, post_logout_uris = $5::jsonb, scopes = $6::varchar(255)
		 WHERE id = $1::uuid AND tenant_id = $2::uuid`, id, nexus.TenantID(r.Context()), in.Name, ru, plu, in.Scopes)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "update failed")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "клиент олдсонгүй")
		return
	}
	h.Audit.Record(r.Context(), "sso.client.update", id, nil)
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

// DELETE /api/sso-clients/{id} — токен/зөвшөөрөл trigger-ээр хамт устна.
func (h *Auth) DeleteSSOClient(w http.ResponseWriter, r *http.Request) {
	id, ok := nexus.UUIDParam(w, r, "id")
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(), `DELETE FROM oauth_clients WHERE id = $1::uuid AND tenant_id = $2::uuid`, id, nexus.TenantID(r.Context()))
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "delete failed")
		return
	}
	if tag.RowsAffected() == 0 {
		httpx.Error(w, http.StatusNotFound, "клиент олдсонгүй")
		return
	}
	h.Audit.Record(r.Context(), "sso.client.delete", id, nil)
	httpx.JSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func randomID(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[randByte()%byte(len(alphabet))]
	}
	return string(b)
}
