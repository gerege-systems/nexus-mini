package oidc

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	codeTTL    = 5 * time.Minute
	accessTTL  = time.Hour
	refreshTTL = 30 * 24 * time.Hour
	idTokenTTL = time.Hour
)

var knownScopes = map[string]bool{"openid": true, "profile": true, "email": true, "tenant": true, "roles": true, "offline_access": true}

// Provider — OIDC provider. pool = nexus_auth (токен/код/түлхүүр); perms —
// roles scope-д хэрэглэгчийн permission; sessions — portal session шийдэх.
type Provider struct {
	Issuer    string // PORTAL_URL + /api/oauth2
	PortalURL string
	pool      *pgxpool.Pool
	sessions  *auth.Service
	perms     nexus.PermissionStore
	keys      keyCache
}

func New(issuer, portalURL string, authPool *pgxpool.Pool, sessions *auth.Service, perms nexus.PermissionStore) *Provider {
	return &Provider{Issuer: issuer, PortalURL: portalURL, pool: authPool, sessions: sessions, perms: perms}
}

func (p *Provider) key(ctx context.Context) (*signingKey, error) {
	p.keys.mu.Lock()
	defer p.keys.mu.Unlock()
	if p.keys.key != nil {
		return p.keys.key, nil
	}
	k, err := loadOrCreateKey(ctx, p.pool)
	if err != nil {
		return nil, err
	}
	p.keys.key = k
	return k, nil
}

// Purge — хугацаа дууссан код/токен (цагийн ticker).
func (p *Provider) Purge(ctx context.Context) (int, error) {
	var n int
	err := p.pool.QueryRow(ctx, `SELECT oauth_purge()`).Scan(&n)
	return n, err
}

// ─── Discovery / JWKS ──────────────────────────────────────────────────

func (p *Provider) Discovery(w http.ResponseWriter, r *http.Request) {
	nexus.JSON(w, http.StatusOK, map[string]any{
		"issuer":                                p.Issuer,
		"authorization_endpoint":                p.Issuer + "/authorize",
		"token_endpoint":                        p.Issuer + "/token",
		"userinfo_endpoint":                     p.Issuer + "/userinfo",
		"jwks_uri":                              p.Issuer + "/jwks",
		"introspection_endpoint":                p.Issuer + "/introspect",
		"revocation_endpoint":                   p.Issuer + "/revoke",
		"end_session_endpoint":                  p.Issuer + "/end_session",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "refresh_token", "client_credentials"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"scopes_supported":                      []string{"openid", "profile", "email", "tenant", "roles", "offline_access"},
		"code_challenge_methods_supported":      []string{"S256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post", "none"},
		"claims_supported":                      []string{"sub", "email", "name", "tenant", "tenant_id", "roles"},
	})
}

func (p *Provider) JWKS(w http.ResponseWriter, r *http.Request) {
	k, err := p.key(r.Context())
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "keys unavailable")
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=3600")
	nexus.JSON(w, http.StatusOK, map[string]any{"keys": []any{k.jwk()}})
}

// ─── Клиент ───────────────────────────────────────────────────────────

type client struct {
	ID, ClientID, Name, Scopes string
	TenantID                   string
	SecretHash                 *string
	RedirectURIs               []string
	PostLogoutURIs             []string
}

func (p *Provider) loadClient(ctx context.Context, clientID string) (*client, error) {
	var c client
	var ru, plu []byte
	err := p.pool.QueryRow(ctx, `
		SELECT id, client_id, tenant_id, client_secret_hash, name, redirect_uris, post_logout_uris, scopes
		  FROM oauth_clients WHERE client_id = $1::varchar(64)`, clientID).
		Scan(&c.ID, &c.ClientID, &c.TenantID, &c.SecretHash, &c.Name, &ru, &plu, &c.Scopes)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(ru, &c.RedirectURIs)
	_ = json.Unmarshal(plu, &c.PostLogoutURIs)
	return &c, nil
}

func (c *client) allowsRedirect(u string) bool {
	for _, r := range c.RedirectURIs {
		if r == u {
			return true
		}
	}
	return false
}

func (c *client) allowsScope(scope string) bool {
	allowed := map[string]bool{}
	for _, s := range strings.Fields(c.Scopes) {
		allowed[s] = true
	}
	for _, s := range strings.Fields(scope) {
		if !allowed[s] && s != "offline_access" {
			return false
		}
	}
	return true
}

// authenticateClient — Basic эсвэл body; public клиент (secret байхгүй) бол
// зөвхөн client_id + PKCE.
func (p *Provider) authenticateClient(r *http.Request) (*client, bool) {
	id, secret, hasBasic := r.BasicAuth()
	if !hasBasic {
		id, secret = r.PostFormValue("client_id"), r.PostFormValue("client_secret")
	}
	if id == "" {
		return nil, false
	}
	c, err := p.loadClient(r.Context(), id)
	if err != nil {
		return nil, false
	}
	if c.SecretHash == nil {
		return c, secret == "" // public клиент secret илгээх ёсгүй
	}
	if secret == "" || !password.Verify(secret, *c.SecretHash) {
		return nil, false
	}
	return c, true
}

// ─── Authorize ────────────────────────────────────────────────────────

var pkceRe = regexp.MustCompile(`^[A-Za-z0-9._~-]{43,128}$`)

type authzReq struct {
	ClientID, RedirectURI, Scope, State, Nonce, Challenge string
}

func parseAuthz(q url.Values) (authzReq, string) {
	a := authzReq{
		ClientID: q.Get("client_id"), RedirectURI: q.Get("redirect_uri"), Scope: strings.TrimSpace(q.Get("scope")),
		State: q.Get("state"), Nonce: q.Get("nonce"), Challenge: q.Get("code_challenge"),
	}
	switch {
	case q.Get("response_type") != "code":
		return a, "unsupported_response_type"
	case a.ClientID == "" || a.RedirectURI == "":
		return a, "invalid_request"
	case q.Get("code_challenge_method") != "S256" || !pkceRe.MatchString(a.Challenge):
		return a, "invalid_request: PKCE S256 шаардлагатай"
	case len(a.State) > 512 || len(a.Nonce) > 255 || len(a.Scope) > 255:
		return a, "invalid_request"
	}
	if a.Scope == "" {
		a.Scope = "openid"
	}
	for _, s := range strings.Fields(a.Scope) {
		if !knownScopes[s] {
			return a, "invalid_scope"
		}
	}
	return a, ""
}

func redirectErr(w http.ResponseWriter, r *http.Request, redirectURI, state, code string) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, code)
		return
	}
	q := u.Query()
	q.Set("error", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

// Authorize — GET /api/oauth2/authorize. Нэвтрээгүй бол portal login руу
// (буцах зам), зөвшөөрөл байхгүй бол consent хуудас руу, байвал код өгнө.
func (p *Provider) Authorize(w http.ResponseWriter, r *http.Request) {
	a, perr := parseAuthz(r.URL.Query())
	c, err := p.loadClient(r.Context(), a.ClientID)
	if err != nil || !c.allowsRedirect(a.RedirectURI) {
		// redirect_uri батлагдаагүй бол ТЭНД буцахгүй (open redirect).
		nexus.Error(w, http.StatusBadRequest, "unknown client or redirect_uri")
		return
	}
	if perr != "" {
		redirectErr(w, r, a.RedirectURI, a.State, strings.SplitN(perr, ":", 2)[0])
		return
	}
	if !c.allowsScope(a.Scope) {
		redirectErr(w, r, a.RedirectURI, a.State, "invalid_scope")
		return
	}
	pr, ok := p.sessions.Resolve(r.Context(), r)
	if !ok {
		next := "/api/oauth2/authorize?" + r.URL.RawQuery
		http.Redirect(w, r, p.PortalURL+"/login?next="+url.QueryEscape(next), http.StatusFound)
		return
	}
	// Хэрэглэгч клиентийн байгууллагын гишүүн байх ёстой.
	if !p.isMember(r.Context(), c.TenantID, pr.UserID) {
		redirectErr(w, r, a.RedirectURI, a.State, "access_denied")
		return
	}
	var granted string
	err = p.pool.QueryRow(r.Context(),
		`SELECT scope FROM oauth_consents WHERE user_id = $1::uuid AND client_id = $2::varchar(64)`,
		pr.UserID, c.ClientID).Scan(&granted)
	if err == nil && scopeSubset(a.Scope, granted) && r.URL.Query().Get("prompt") != "consent" {
		p.issueCodeRedirect(w, r, c, pr.UserID, a)
		return
	}
	http.Redirect(w, r, p.PortalURL+"/oauth/consent?"+r.URL.RawQuery, http.StatusFound)
}

func scopeSubset(want, have string) bool {
	h := map[string]bool{}
	for _, s := range strings.Fields(have) {
		h[s] = true
	}
	for _, s := range strings.Fields(want) {
		if !h[s] {
			return false
		}
	}
	return true
}

func (p *Provider) isMember(ctx context.Context, tenantID, userID string) bool {
	// nexus_auth-д memberships-д шууд эрх байхгүй тул sessions-ийн definer
	// замаар биш, perms store-оор (RLS-тэй app pool) шалгана: гишүүн биш бол
	// grants хоосон байна; гэхдээ эрхгүй гишүүн ч хоосон — тиймээс tenant
	// сонголтын definer функцээр батална.
	ok, err := p.sessions.IsMember(ctx, tenantID, userID)
	return err == nil && ok
}

// ConsentInfo — GET /api/oauth2/consent?... (portal consent хуудас уншина).
func (p *Provider) ConsentInfo(w http.ResponseWriter, r *http.Request) {
	a, perr := parseAuthz(r.URL.Query())
	c, err := p.loadClient(r.Context(), a.ClientID)
	if err != nil || !c.allowsRedirect(a.RedirectURI) || perr != "" || !c.allowsScope(a.Scope) {
		nexus.Error(w, http.StatusBadRequest, "invalid authorization request")
		return
	}
	var tenantName string
	_ = p.pool.QueryRow(r.Context(), `SELECT name FROM tenant_public_name($1::uuid)`, c.TenantID).Scan(&tenantName)
	nexus.JSON(w, http.StatusOK, map[string]any{
		"client_name": c.Name, "tenant_name": tenantName, "scopes": strings.Fields(a.Scope),
		"redirect_host": hostOf(a.RedirectURI),
	})
}

func hostOf(u string) string {
	if p, err := url.Parse(u); err == nil {
		return p.Host
	}
	return ""
}

// Consent — POST /api/oauth2/consent {approve: bool, query: "..."} → {redirect}.
func (p *Provider) Consent(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Approve bool   `json:"approve"`
		Query   string `json:"query"`
	}
	if !nexus.Decode(w, r, &in) {
		return
	}
	q, err := url.ParseQuery(in.Query)
	if err != nil {
		nexus.Error(w, http.StatusBadRequest, "bad query")
		return
	}
	a, perr := parseAuthz(q)
	c, err := p.loadClient(r.Context(), a.ClientID)
	if err != nil || !c.allowsRedirect(a.RedirectURI) || perr != "" || !c.allowsScope(a.Scope) {
		nexus.Error(w, http.StatusBadRequest, "invalid authorization request")
		return
	}
	pr, ok := p.sessions.Resolve(r.Context(), r)
	if !ok {
		nexus.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	if !in.Approve {
		nexus.JSON(w, http.StatusOK, map[string]string{"redirect": withErr(a.RedirectURI, a.State, "access_denied")})
		return
	}
	if !p.isMember(r.Context(), c.TenantID, pr.UserID) {
		nexus.JSON(w, http.StatusOK, map[string]string{"redirect": withErr(a.RedirectURI, a.State, "access_denied")})
		return
	}
	if _, err := p.pool.Exec(r.Context(), `
		INSERT INTO oauth_consents (user_id, client_id, scope) VALUES ($1::uuid, $2::varchar(64), $3::varchar(255))
		ON CONFLICT (user_id, client_id) DO UPDATE SET scope = excluded.scope, granted_at = now()`,
		pr.UserID, c.ClientID, a.Scope); err != nil {
		nexus.Error(w, http.StatusInternalServerError, "consent failed")
		return
	}
	redirect, err := p.issueCode(r.Context(), c, pr.UserID, a)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "code failed")
		return
	}
	nexus.JSON(w, http.StatusOK, map[string]string{"redirect": redirect})
}

func withErr(redirectURI, state, code string) string {
	u, _ := url.Parse(redirectURI)
	q := u.Query()
	q.Set("error", code)
	if state != "" {
		q.Set("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (p *Provider) issueCode(ctx context.Context, c *client, userID string, a authzReq) (string, error) {
	code := randToken()
	if _, err := p.pool.Exec(ctx, `
		INSERT INTO oauth_codes (code_hash, client_id, tenant_id, user_id, redirect_uri, scope, nonce, code_challenge, expires_at)
		VALUES ($1::char(64), $2::varchar(64), $3::uuid, $4::uuid, $5::varchar(1000), $6::varchar(255), $7::varchar(255), $8::varchar(128), $9::timestamptz)`,
		sha256hex(code), c.ClientID, c.TenantID, userID, a.RedirectURI, a.Scope, a.Nonce, a.Challenge, time.Now().Add(codeTTL)); err != nil {
		return "", err
	}
	u, _ := url.Parse(a.RedirectURI)
	q := u.Query()
	q.Set("code", code)
	if a.State != "" {
		q.Set("state", a.State)
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (p *Provider) issueCodeRedirect(w http.ResponseWriter, r *http.Request, c *client, userID string, a authzReq) {
	u, err := p.issueCode(r.Context(), c, userID, a)
	if err != nil {
		redirectErr(w, r, a.RedirectURI, a.State, "server_error")
		return
	}
	http.Redirect(w, r, u, http.StatusFound)
}

// ─── Token ────────────────────────────────────────────────────────────

func tokenErr(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Cache-Control", "no-store")
	nexus.JSON(w, status, map[string]string{"error": code, "error_description": desc})
}

// Token — POST /api/oauth2/token (form).
func (p *Provider) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		tokenErr(w, 400, "invalid_request", "form")
		return
	}
	c, ok := p.authenticateClient(r)
	if !ok {
		w.Header().Set("WWW-Authenticate", `Basic realm="nexus-mini"`)
		tokenErr(w, 401, "invalid_client", "client authentication failed")
		return
	}
	switch r.PostFormValue("grant_type") {
	case "authorization_code":
		p.grantCode(w, r, c)
	case "refresh_token":
		p.grantRefresh(w, r, c)
	case "client_credentials":
		if c.SecretHash == nil {
			tokenErr(w, 400, "unauthorized_client", "public client")
			return
		}
		scope := strings.TrimSpace(r.PostFormValue("scope"))
		if scope == "" {
			scope = "tenant"
		}
		if !c.allowsScope(scope) {
			tokenErr(w, 400, "invalid_scope", "")
			return
		}
		p.respondTokens(w, r, c, nil, scope, "", false)
	default:
		tokenErr(w, 400, "unsupported_grant_type", "")
	}
}

func (p *Provider) grantCode(w http.ResponseWriter, r *http.Request, c *client) {
	code, verifier, redirect := r.PostFormValue("code"), r.PostFormValue("code_verifier"), r.PostFormValue("redirect_uri")
	if code == "" || !pkceRe.MatchString(verifier) {
		tokenErr(w, 400, "invalid_request", "code / code_verifier")
		return
	}
	var userID, scope, nonce, challenge, storedRedirect, clientID string
	err := p.pool.QueryRow(r.Context(), `
		DELETE FROM oauth_codes WHERE code_hash = $1::char(64) AND expires_at > clock_timestamp()
		RETURNING user_id, scope, nonce, code_challenge, redirect_uri, client_id`, sha256hex(code)).
		Scan(&userID, &scope, &nonce, &challenge, &storedRedirect, &clientID)
	if errors.Is(err, pgx.ErrNoRows) {
		tokenErr(w, 400, "invalid_grant", "code unknown, used or expired")
		return
	}
	if err != nil {
		tokenErr(w, 500, "server_error", "")
		return
	}
	sum := sha256.Sum256([]byte(verifier))
	if clientID != c.ClientID || storedRedirect != redirect ||
		subtle.ConstantTimeCompare([]byte(base64.RawURLEncoding.EncodeToString(sum[:])), []byte(challenge)) != 1 {
		tokenErr(w, 400, "invalid_grant", "client / redirect_uri / PKCE mismatch")
		return
	}
	p.respondTokens(w, r, c, &userID, scope, nonce, true)
}

func (p *Provider) grantRefresh(w http.ResponseWriter, r *http.Request, c *client) {
	rt := r.PostFormValue("refresh_token")
	if rt == "" {
		tokenErr(w, 400, "invalid_request", "refresh_token")
		return
	}
	var family, scope, clientID string
	var userID *string
	var revoked *time.Time
	err := p.pool.QueryRow(r.Context(), `
		SELECT family, scope, client_id, user_id, revoked_at FROM oauth_tokens
		 WHERE token_hash = $1::char(64) AND kind = 'refresh' AND expires_at > clock_timestamp()`, sha256hex(rt)).
		Scan(&family, &scope, &clientID, &userID, &revoked)
	if errors.Is(err, pgx.ErrNoRows) || clientID != c.ClientID {
		tokenErr(w, 400, "invalid_grant", "refresh token unknown")
		return
	}
	if err != nil {
		tokenErr(w, 500, "server_error", "")
		return
	}
	if revoked != nil {
		// Replay: хүчингүй болсон refresh дахин ирлээ — гэр бүлийг бүхэлд нь хаана.
		_, _ = p.pool.Exec(r.Context(), `UPDATE oauth_tokens SET revoked_at = now() WHERE family = $1::uuid AND revoked_at IS NULL`, family)
		log.Printf("oidc: refresh replay, family %s revoked (client %s)", family, clientID)
		tokenErr(w, 400, "invalid_grant", "refresh token reused — family revoked")
		return
	}
	if _, err := p.pool.Exec(r.Context(), `UPDATE oauth_tokens SET revoked_at = now() WHERE family = $1::uuid AND revoked_at IS NULL`, family); err != nil {
		tokenErr(w, 500, "server_error", "")
		return
	}
	p.respondTokensFamily(w, r, c, userID, scope, "", true, family)
}

func (p *Provider) respondTokens(w http.ResponseWriter, r *http.Request, c *client, userID *string, scope, nonce string, withRefresh bool) {
	p.respondTokensFamily(w, r, c, userID, scope, nonce, withRefresh, "")
}

func (p *Provider) respondTokensFamily(w http.ResponseWriter, r *http.Request, c *client, userID *string, scope, nonce string, withRefresh bool, family string) {
	ctx := r.Context()
	if family == "" {
		var f string
		if err := p.pool.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&f); err != nil {
			tokenErr(w, 500, "server_error", "")
			return
		}
		family = f
	}
	access := randToken()
	if _, err := p.pool.Exec(ctx, `
		INSERT INTO oauth_tokens (token_hash, kind, family, client_id, tenant_id, user_id, scope, expires_at)
		VALUES ($1::char(64), 'access', $2::uuid, $3::varchar(64), $4::uuid, $5::uuid, $6::varchar(255), $7::timestamptz)`,
		sha256hex(access), family, c.ClientID, c.TenantID, userID, scope, time.Now().Add(accessTTL)); err != nil {
		tokenErr(w, 500, "server_error", "")
		return
	}
	out := map[string]any{"access_token": access, "token_type": "Bearer", "expires_in": int(accessTTL.Seconds()), "scope": scope}
	if withRefresh && userID != nil && strings.Contains(" "+scope+" ", " offline_access ") {
		refresh := randToken()
		if _, err := p.pool.Exec(ctx, `
			INSERT INTO oauth_tokens (token_hash, kind, family, client_id, tenant_id, user_id, scope, expires_at)
			VALUES ($1::char(64), 'refresh', $2::uuid, $3::varchar(64), $4::uuid, $5::uuid, $6::varchar(255), $7::timestamptz)`,
			sha256hex(refresh), family, c.ClientID, c.TenantID, userID, scope, time.Now().Add(refreshTTL)); err != nil {
			tokenErr(w, 500, "server_error", "")
			return
		}
		out["refresh_token"] = refresh
	}
	if userID != nil && strings.Contains(" "+scope+" ", " openid ") {
		claims, err := p.claims(ctx, c, *userID, scope)
		if err != nil {
			tokenErr(w, 500, "server_error", "claims")
			return
		}
		now := time.Now()
		claims["iss"] = p.Issuer
		claims["aud"] = c.ClientID
		claims["iat"] = now.Unix()
		claims["exp"] = now.Add(idTokenTTL).Unix()
		if nonce != "" {
			claims["nonce"] = nonce
		}
		k, err := p.key(ctx)
		if err != nil {
			tokenErr(w, 500, "server_error", "keys")
			return
		}
		idt, err := signJWT(k, claims)
		if err != nil {
			tokenErr(w, 500, "server_error", "sign")
			return
		}
		out["id_token"] = idt
	}
	w.Header().Set("Cache-Control", "no-store")
	nexus.JSON(w, http.StatusOK, out)
}

// claims — sub + scope-ийн дагуу (profile → name, email → email, tenant →
// tenant/tenant_id, roles → role кодууд).
func (p *Provider) claims(ctx context.Context, c *client, userID, scope string) (map[string]any, error) {
	var name, email, slug string
	var roles []string
	if err := p.pool.QueryRow(ctx,
		`SELECT name, email, tenant_slug, roles FROM oidc_user_claims($1::uuid, $2::uuid)`, userID, c.TenantID).
		Scan(&name, &email, &slug, &roles); err != nil {
		return nil, err
	}
	out := map[string]any{"sub": userID}
	has := func(s string) bool { return strings.Contains(" "+scope+" ", " "+s+" ") }
	if has("profile") {
		out["name"] = name
	}
	if has("email") {
		out["email"] = email
		out["email_verified"] = false
	}
	if has("tenant") {
		out["tenant"] = slug
		out["tenant_id"] = c.TenantID
	}
	if has("roles") {
		out["roles"] = roles
	}
	return out, nil
}

// ─── Userinfo / Introspect / Revoke / End session ────────────────────

type tokenInfo struct {
	ClientID, TenantID, Scope string
	UserID                    *string
	Exp                       time.Time
}

func (p *Provider) lookupAccess(ctx context.Context, token string) (*tokenInfo, bool) {
	var ti tokenInfo
	err := p.pool.QueryRow(ctx, `
		SELECT client_id, tenant_id, user_id, scope, expires_at FROM oauth_tokens
		 WHERE token_hash = $1::char(64) AND kind = 'access' AND revoked_at IS NULL AND expires_at > clock_timestamp()`,
		sha256hex(token)).Scan(&ti.ClientID, &ti.TenantID, &ti.UserID, &ti.Scope, &ti.Exp)
	if err != nil {
		return nil, false
	}
	return &ti, true
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

func (p *Provider) Userinfo(w http.ResponseWriter, r *http.Request) {
	ti, ok := p.lookupAccess(r.Context(), bearer(r))
	if !ok || ti.UserID == nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		nexus.Error(w, http.StatusUnauthorized, "invalid_token")
		return
	}
	c, err := p.loadClient(r.Context(), ti.ClientID)
	if err != nil {
		nexus.Error(w, http.StatusUnauthorized, "invalid_token")
		return
	}
	claims, err := p.claims(r.Context(), c, *ti.UserID, ti.Scope)
	if err != nil {
		nexus.Error(w, http.StatusInternalServerError, "claims")
		return
	}
	nexus.JSON(w, http.StatusOK, claims)
}

func (p *Provider) Introspect(w http.ResponseWriter, r *http.Request) {
	if _, ok := p.authenticateClient(r); !ok {
		tokenErr(w, 401, "invalid_client", "")
		return
	}
	ti, ok := p.lookupAccess(r.Context(), r.PostFormValue("token"))
	if !ok {
		nexus.JSON(w, http.StatusOK, map[string]any{"active": false})
		return
	}
	out := map[string]any{"active": true, "client_id": ti.ClientID, "scope": ti.Scope, "exp": ti.Exp.Unix(),
		"tenant_id": ti.TenantID, "token_type": "Bearer"}
	if ti.UserID != nil {
		out["sub"] = *ti.UserID
	}
	nexus.JSON(w, http.StatusOK, out)
}

func (p *Provider) Revoke(w http.ResponseWriter, r *http.Request) {
	c, ok := p.authenticateClient(r)
	if !ok {
		tokenErr(w, 401, "invalid_client", "")
		return
	}
	// RFC 7009: үл мэдэх токенд ч 200. Refresh бол гэр бүлээрээ.
	_, _ = p.pool.Exec(r.Context(), `
		UPDATE oauth_tokens SET revoked_at = now()
		 WHERE revoked_at IS NULL AND client_id = $2::varchar(64)
		   AND family = (SELECT family FROM oauth_tokens WHERE token_hash = $1::char(64))`,
		sha256hex(r.PostFormValue("token")), c.ClientID)
	w.WriteHeader(http.StatusOK)
}

// EndSession — RP-initiated logout: id_token_hint батлагдвал бүртгэлтэй
// post_logout_redirect_uri руу; portal session-ийг ч дуусгана.
func (p *Provider) EndSession(w http.ResponseWriter, r *http.Request) {
	hint := r.URL.Query().Get("id_token_hint")
	target := p.PortalURL + "/"
	if hint != "" {
		if k, err := p.key(r.Context()); err == nil {
			if claims, ok := verifyJWT(k, hint); ok {
				if aud, _ := claims["aud"].(string); aud != "" {
					if c, err := p.loadClient(r.Context(), aud); err == nil {
						if want := r.URL.Query().Get("post_logout_redirect_uri"); want != "" {
							for _, u := range c.PostLogoutURIs {
								if u == want {
									target = want
								}
							}
						}
					}
				}
			}
		}
	}
	p.sessions.EndSession(r.Context(), w, r)
	if st := r.URL.Query().Get("state"); st != "" {
		u, _ := url.Parse(target)
		q := u.Query()
		q.Set("state", st)
		u.RawQuery = q.Encode()
		target = u.String()
	}
	http.Redirect(w, r, target, http.StatusFound)
}

// ─── Тусламж: клиент secret ───────────────────────────────────────────

// NewClientSecret — 32 байт санамсаргүй secret + argon2 hash (хадгалахад).
func NewClientSecret() (plain, hash string, err error) {
	plain = randToken()
	hash, err = password.Hash(plain)
	return
}

func (p *Provider) String() string { return fmt.Sprintf("oidc(%s)", p.Issuer) }
