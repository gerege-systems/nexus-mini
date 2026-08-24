package oidc

// OIDC provider-ийн БҮТЭН урсгал бодит DB дээр: authorize → consent → code →
// token (PKCE) → userinfo/introspect → refresh rotation → replay → revoke →
// end_session. make check-db-д ажиллана.

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	p                        *Provider
	svc                      *auth.Service
	authPool, owner          *pgxpool.Pool
	userID, tenantID, cookie string
	clientID, secret         string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	authURL, ownerURL := os.Getenv("NEXUS_TEST_DATABASE_URL_AUTH"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if authURL == "" || ownerURL == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL_AUTH / _OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	ctx := context.Background()
	open := func(u string) *pgxpool.Pool {
		p, err := pgxpool.New(ctx, u)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	f := &fixture{authPool: open(authURL), owner: open(ownerURL)}
	clean := func() {
		_, _ = f.owner.Exec(ctx, `DELETE FROM oauth_clients WHERE name = 'oidctest'`)
		_, _ = f.owner.Exec(ctx, `DELETE FROM tenants WHERE slug = 'oidctest'`)
		_, _ = f.owner.Exec(ctx, `DELETE FROM users WHERE email = 'oidctest@x.mn'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		f.authPool.Close()
		f.owner.Close()
	})

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(f.owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('oidctest@x.mn','x','ОИДК Тест') RETURNING id`).Scan(&f.userID))
	must(f.owner.QueryRow(ctx, `INSERT INTO tenants (slug, name) VALUES ('oidctest','ОИДК') RETURNING id`).Scan(&f.tenantID))
	var memberID, roleID string
	must(f.owner.QueryRow(ctx, `INSERT INTO memberships (tenant_id, user_id) VALUES ($1,$2) RETURNING id`, f.tenantID, f.userID).Scan(&memberID))
	must(f.owner.QueryRow(ctx, `INSERT INTO roles (tenant_id, code, name) VALUES ($1,'admin','Админ') RETURNING id`, f.tenantID).Scan(&roleID))
	_, err := f.owner.Exec(ctx, `INSERT INTO membership_roles (membership_id, role_id) VALUES ($1,$2)`, memberID, roleID)
	must(err)

	plain, hash, err := NewClientSecret()
	must(err)
	f.clientID, f.secret = "nx_oidctest_"+strings.ToLower(f.tenantID[:8]), plain
	_, err = f.owner.Exec(ctx, `
		INSERT INTO oauth_clients (tenant_id, client_id, client_secret_hash, name, redirect_uris, scopes)
		VALUES ($1,$2,$3,'oidctest','["https://rp.mn/cb"]'::jsonb,'openid profile email tenant roles offline_access')`,
		f.tenantID, f.clientID, hash)
	must(err)

	f.svc = auth.NewService(f.authPool, false)
	f.p = New("https://portal.mn/api/oauth2", "https://portal.mn", f.authPool, f.svc, nil)

	// Хэрэглэгчийн session (cookie) — authorize/consent-д хэрэгтэй.
	w := httptest.NewRecorder()
	sid, err := f.svc.StartSession(ctx, w, f.userID)
	must(err)
	_, err = f.svc.SetTenant(ctx, sid, f.tenantID)
	must(err)
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			f.cookie = c.Value
		}
	}
	if f.cookie == "" {
		t.Fatal("session cookie үүсээгүй")
	}
	return f
}

func (f *fixture) get(t *testing.T, h http.HandlerFunc, target string, withCookie bool) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodGet, target, nil)
	if withCookie {
		r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: f.cookie})
	}
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func (f *fixture) postForm(t *testing.T, h http.HandlerFunc, target string, form url.Values, basic bool) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, target, strings.NewReader(form.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if basic {
		r.SetBasicAuth(f.clientID, f.secret)
	}
	w := httptest.NewRecorder()
	h(w, r)
	return w
}

func pkce() (verifier, challenge string) {
	verifier = strings.Repeat("a", 43)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func (f *fixture) authzQuery(challenge, scope string) string {
	return url.Values{
		"response_type": {"code"}, "client_id": {f.clientID}, "redirect_uri": {"https://rp.mn/cb"},
		"scope": {scope}, "state": {"st"}, "nonce": {"no"},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}.Encode()
}

func TestOIDCFullFlow(t *testing.T) {
	f := newFixture(t)
	verifier, challenge := pkce()
	q := f.authzQuery(challenge, "openid profile email tenant roles offline_access")

	// 1. Нэвтрээгүй → portal login руу (next-тэй).
	w := f.get(t, f.p.Authorize, "/api/oauth2/authorize?"+q, false)
	if w.Code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "/login?next=") {
		t.Fatalf("нэвтрээгүй: %d %s", w.Code, w.Header().Get("Location"))
	}
	// 2. Нэвтэрсэн, зөвшөөрөлгүй → consent хуудас руу.
	w = f.get(t, f.p.Authorize, "/api/oauth2/authorize?"+q, true)
	if w.Code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "/oauth/consent?") {
		t.Fatalf("consent руу шилжсэнгүй: %d %s", w.Code, w.Header().Get("Location"))
	}
	// 3. Consent-ийн мэдээлэл.
	w = f.get(t, f.p.ConsentInfo, "/api/oauth2/consent?"+q, true)
	var info map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &info)
	if w.Code != 200 || info["client_name"] != "oidctest" || info["tenant_name"] != "ОИДК" {
		t.Fatalf("consent info: %d %v", w.Code, info)
	}
	// 4. Зөвшөөрөх → code.
	code := f.approve(t, q)
	// 5. Токен солилцоо (PKCE).
	tok := f.token(t, url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {"https://rp.mn/cb"}, "code_verifier": {verifier}}, 200)
	for _, k := range []string{"access_token", "refresh_token", "id_token", "expires_in", "scope"} {
		if tok[k] == nil {
			t.Fatalf("токенд %s алга: %v", k, tok)
		}
	}
	// id_token-ийн claims.
	claims := decodeJWT(t, tok["id_token"].(string))
	if claims["iss"] != "https://portal.mn/api/oauth2" || claims["aud"] != f.clientID ||
		claims["nonce"] != "no" || claims["email"] != "oidctest@x.mn" || claims["tenant"] != "oidctest" {
		t.Fatalf("claims = %v", claims)
	}
	if roles, _ := claims["roles"].([]any); len(roles) != 1 || roles[0] != "admin" {
		t.Fatalf("roles claim = %v", claims["roles"])
	}
	access, refresh := tok["access_token"].(string), tok["refresh_token"].(string)

	// 6. Код дахин ашиглах — татгалзана.
	f.token(t, url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {"https://rp.mn/cb"}, "code_verifier": {verifier}}, 400)

	// 7. userinfo + introspect.
	r := httptest.NewRequest(http.MethodGet, "/api/oauth2/userinfo", nil)
	r.Header.Set("Authorization", "Bearer "+access)
	w = httptest.NewRecorder()
	f.p.Userinfo(w, r)
	var ui map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &ui)
	if w.Code != 200 || ui["sub"] != f.userID {
		t.Fatalf("userinfo: %d %v", w.Code, ui)
	}
	w = f.postForm(t, f.p.Introspect, "/api/oauth2/introspect", url.Values{"token": {access}}, true)
	var in map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &in)
	if in["active"] != true || in["client_id"] != f.clientID {
		t.Fatalf("introspect = %v", in)
	}

	// 8. Refresh rotation.
	tok2 := f.token(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}}, 200)
	newRefresh := tok2["refresh_token"].(string)
	if newRefresh == refresh {
		t.Fatal("refresh эргэлдээгүй")
	}
	// 9. Хуучин refresh дахин → replay → гэр бүл бүхэлдээ хүчингүй.
	f.token(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refresh}}, 400)
	f.token(t, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {newRefresh}}, 400)
	w = f.postForm(t, f.p.Introspect, "/api/oauth2/introspect", url.Values{"token": {access}}, true)
	_ = json.Unmarshal(w.Body.Bytes(), &in)
	if in["active"] != false {
		t.Fatalf("replay-ийн дараа access идэвхтэй хэвээр: %v", in)
	}

	// 10. Зөвшөөрөл санагдсан тул authorize шууд код өгнө.
	w = f.get(t, f.p.Authorize, "/api/oauth2/authorize?"+q, true)
	if w.Code != http.StatusFound || !strings.Contains(w.Header().Get("Location"), "https://rp.mn/cb?code=") {
		t.Fatalf("санасан зөвшөөрөл: %d %s", w.Code, w.Header().Get("Location"))
	}
	// prompt=consent бол дахин асууна.
	w = f.get(t, f.p.Authorize, "/api/oauth2/authorize?"+q+"&prompt=consent", true)
	if !strings.Contains(w.Header().Get("Location"), "/oauth/consent?") {
		t.Fatalf("prompt=consent: %s", w.Header().Get("Location"))
	}
}

func TestOIDCTokenRejections(t *testing.T) {
	f := newFixture(t)
	verifier, challenge := pkce()
	q := f.authzQuery(challenge, "openid offline_access")
	code := f.approve(t, q)

	// Буруу PKCE verifier.
	f.token(t, url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {"https://rp.mn/cb"}, "code_verifier": {strings.Repeat("b", 43)}}, 400)
	// Код нэг удаа — дээрх оролдлого хэрэглэсэн тул зөв verifier ч ажиллахгүй.
	f.token(t, url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {"https://rp.mn/cb"}, "code_verifier": {verifier}}, 400)

	// Буруу redirect_uri.
	code2 := f.approve(t, q)
	f.token(t, url.Values{"grant_type": {"authorization_code"}, "code": {code2},
		"redirect_uri": {"https://evil.mn/cb"}, "code_verifier": {verifier}}, 400)

	// Буруу secret → 401.
	r := httptest.NewRequest(http.MethodPost, "/api/oauth2/token",
		strings.NewReader(url.Values{"grant_type": {"client_credentials"}}.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.SetBasicAuth(f.clientID, "буруу")
	w := httptest.NewRecorder()
	f.p.Token(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("буруу secret = %d", w.Code)
	}
	// Дэмжигдээгүй grant.
	f.token(t, url.Values{"grant_type": {"password"}, "username": {"a"}}, 400)
	// client_credentials — sub-гүй, tenant-тай.
	tok := f.token(t, url.Values{"grant_type": {"client_credentials"}, "scope": {"tenant"}}, 200)
	if tok["refresh_token"] != nil || tok["id_token"] != nil {
		t.Fatalf("client_credentials-д refresh/id_token гарав: %v", tok)
	}
	w = f.postForm(t, f.p.Introspect, "/api/oauth2/introspect", url.Values{"token": {tok["access_token"].(string)}}, true)
	var in map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &in)
	if in["active"] != true || in["sub"] != nil || in["tenant_id"] != f.tenantID {
		t.Fatalf("cc introspect = %v", in)
	}
	// Revoke → идэвхгүй.
	w = f.postForm(t, f.p.Revoke, "/api/oauth2/revoke", url.Values{"token": {tok["access_token"].(string)}}, true)
	if w.Code != 200 {
		t.Fatalf("revoke = %d", w.Code)
	}
	w = f.postForm(t, f.p.Introspect, "/api/oauth2/introspect", url.Values{"token": {tok["access_token"].(string)}}, true)
	_ = json.Unmarshal(w.Body.Bytes(), &in)
	if in["active"] != false {
		t.Fatalf("revoke-ийн дараа идэвхтэй: %v", in)
	}
}

func TestOIDCAuthorizeGuards(t *testing.T) {
	f := newFixture(t)
	_, challenge := pkce()
	// Үл мэдэх клиент / буруу redirect — ТЭР ЗАМ РУУ буцаахгүй (open redirect).
	for _, q := range []string{
		url.Values{"response_type": {"code"}, "client_id": {"үл-мэдэх"}, "redirect_uri": {"https://evil.mn/cb"},
			"code_challenge": {challenge}, "code_challenge_method": {"S256"}}.Encode(),
		url.Values{"response_type": {"code"}, "client_id": {f.clientID}, "redirect_uri": {"https://evil.mn/cb"},
			"code_challenge": {challenge}, "code_challenge_method": {"S256"}}.Encode(),
	} {
		w := f.get(t, f.p.Authorize, "/api/oauth2/authorize?"+q, true)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("open redirect хамгаалалт: %d %s", w.Code, w.Header().Get("Location"))
		}
	}
	// PKCE-гүй → клиентийн redirect руу error=invalid_request.
	q := url.Values{"response_type": {"code"}, "client_id": {f.clientID}, "redirect_uri": {"https://rp.mn/cb"},
		"state": {"st"}}.Encode()
	w := f.get(t, f.p.Authorize, "/api/oauth2/authorize?"+q, true)
	loc := w.Header().Get("Location")
	if w.Code != http.StatusFound || !strings.Contains(loc, "error=invalid_request") || !strings.Contains(loc, "state=st") {
		t.Fatalf("PKCE шаардлага: %d %s", w.Code, loc)
	}
	// Гишүүн биш хэрэглэгч → access_denied.
	ctx := context.Background()
	var otherUser string
	if err := f.owner.QueryRow(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('oidctest-out@x.mn','x','Гадны') RETURNING id`).Scan(&otherUser); err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = f.owner.Exec(ctx, `DELETE FROM users WHERE email = 'oidctest-out@x.mn'`) }()
	w2 := httptest.NewRecorder()
	sid, err := f.svc.StartSession(ctx, w2, otherUser)
	if err != nil {
		t.Fatal(err)
	}
	_ = sid
	var cookie string
	for _, c := range w2.Result().Cookies() {
		if c.Name == auth.CookieName {
			cookie = c.Value
		}
	}
	_, ch := pkce()
	r := httptest.NewRequest(http.MethodGet, "/api/oauth2/authorize?"+f.authzQuery(ch, "openid"), nil)
	r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: cookie})
	w3 := httptest.NewRecorder()
	f.p.Authorize(w3, r)
	if !strings.Contains(w3.Header().Get("Location"), "error=access_denied") {
		t.Fatalf("гишүүн бус: %s", w3.Header().Get("Location"))
	}
}

func TestOIDCDiscoveryAndJWKS(t *testing.T) {
	f := newFixture(t)
	w := f.get(t, f.p.Discovery, "/api/oauth2/.well-known/openid-configuration", false)
	var d map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &d)
	if d["issuer"] != "https://portal.mn/api/oauth2" || d["token_endpoint"] != "https://portal.mn/api/oauth2/token" {
		t.Fatalf("discovery = %v", d)
	}
	methods, _ := d["code_challenge_methods_supported"].([]any)
	if len(methods) != 1 || methods[0] != "S256" {
		t.Fatalf("PKCE арга = %v", methods)
	}
	w = f.get(t, f.p.JWKS, "/api/oauth2/jwks", false)
	var jwks struct {
		Keys []map[string]any `json:"keys"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &jwks)
	if len(jwks.Keys) != 1 || jwks.Keys[0]["alg"] != "RS256" || jwks.Keys[0]["d"] != nil {
		t.Fatalf("jwks = %v (нууц түлхүүр алдагдсан бол d талбар гарна)", jwks.Keys)
	}
	// Purge ажиллана.
	if _, err := f.p.Purge(context.Background()); err != nil {
		t.Fatalf("Purge: %v", err)
	}
}

// approve — consent зөвшөөрч кодыг гаргаж авна.
func (f *fixture) approve(t *testing.T, q string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"approve": true, "query": q})
	r := httptest.NewRequest(http.MethodPost, "/api/oauth2/consent", strings.NewReader(string(body)))
	r.Header.Set("Content-Type", "application/json")
	r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: f.cookie})
	w := httptest.NewRecorder()
	f.p.Consent(w, r)
	var out map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	u, err := url.Parse(out["redirect"])
	if err != nil || u.Query().Get("code") == "" {
		t.Fatalf("consent → %d %s", w.Code, w.Body.String())
	}
	if u.Query().Get("state") != "st" {
		t.Fatalf("state буцаагүй: %s", out["redirect"])
	}
	return u.Query().Get("code")
}

func (f *fixture) token(t *testing.T, form url.Values, wantCode int) map[string]any {
	t.Helper()
	w := f.postForm(t, f.p.Token, "/api/oauth2/token", form, true)
	if w.Code != wantCode {
		t.Fatalf("token(%s) = %d, хүлээсэн %d: %s", form.Get("grant_type"), w.Code, wantCode, w.Body.String())
	}
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	return out
}

func decodeJWT(t *testing.T, tok string) map[string]any {
	t.Helper()
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("id_token формат: %s", tok)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	var claims map[string]any
	if err := json.Unmarshal(raw, &claims); err != nil {
		t.Fatal(err)
	}
	return claims
}

func TestOIDCEndSession(t *testing.T) {
	f := newFixture(t)
	verifier, challenge := pkce()
	q := f.authzQuery(challenge, "openid")
	code := f.approve(t, q)
	tok := f.token(t, url.Values{"grant_type": {"authorization_code"}, "code": {code},
		"redirect_uri": {"https://rp.mn/cb"}, "code_verifier": {verifier}}, 200)
	idToken := tok["id_token"].(string)

	// Бүртгэлгүй post_logout_redirect_uri — portal руу (open redirect хаалттай).
	w := f.get(t, f.p.EndSession, "/api/oauth2/end_session?id_token_hint="+idToken+
		"&post_logout_redirect_uri="+url.QueryEscape("https://evil.mn/"), true)
	if w.Code != http.StatusFound || !strings.HasPrefix(w.Header().Get("Location"), "https://portal.mn/") {
		t.Fatalf("бүртгэлгүй URI = %d %s", w.Code, w.Header().Get("Location"))
	}
	// id_token_hint-гүй ч portal руу, session дуусна.
	w = f.get(t, f.p.EndSession, "/api/oauth2/end_session?state=xyz", true)
	loc := w.Header().Get("Location")
	if w.Code != http.StatusFound || !strings.Contains(loc, "state=xyz") {
		t.Fatalf("end_session = %d %s", w.Code, loc)
	}
	// Бүртгэлтэй post_logout URI — тэр рүү буцна.
	ctx := context.Background()
	if _, err := f.owner.Exec(ctx, `UPDATE oauth_clients SET post_logout_uris = '["https://rp.mn/bye"]'::jsonb WHERE client_id = $1`, f.clientID); err != nil {
		t.Fatal(err)
	}
	w = f.get(t, f.p.EndSession, "/api/oauth2/end_session?id_token_hint="+idToken+
		"&post_logout_redirect_uri="+url.QueryEscape("https://rp.mn/bye"), true)
	if w.Header().Get("Location") != "https://rp.mn/bye" {
		t.Fatalf("бүртгэлтэй URI = %s", w.Header().Get("Location"))
	}
	if f.p.String() == "" {
		t.Fatal("String() хоосон")
	}
}
