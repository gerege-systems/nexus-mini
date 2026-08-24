package handlers_test

// SSO-оор нэвтрэх (relying party) HTTP урсгал: state cookie (HMAC), провайдер
// жагсаалт, callback-ийн бүх татгалзал, JIT бүртгэл (хаалттай/нээлттэй).

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/handlers"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/ssoclient"
)

// fakeIDP — тест доторх OIDC provider (discovery + JWKS + token).
type fakeIDP struct {
	srv    *httptest.Server
	priv   *rsa.PrivateKey
	issuer string
	email  string
	nonce  string
}

func startIDP(t *testing.T) *fakeIDP {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIDP{priv: priv, email: "SSO-Хэрэглэгч@x.mn"}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": f.issuer, "authorization_endpoint": f.issuer + "/auth",
			"token_endpoint": f.issuer + "/token", "jwks_uri": f.issuer + "/jwks",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": "k1",
			"n": b64(priv.PublicKey.N.Bytes()), "e": b64(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.PostFormValue("code") != "good" {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		claims := map[string]any{"iss": f.issuer, "aud": "cid", "sub": "u-1", "email": f.email,
			"email_verified": true, "name": "SSO Хэрэглэгч", "nonce": f.nonce,
			"exp": time.Now().Add(time.Hour).Unix()}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": f.sign(t, claims)})
	})
	f.srv = httptest.NewServer(mux)
	f.issuer = f.srv.URL
	t.Cleanup(f.srv.Close)
	return f
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func (f *fakeIDP) sign(t *testing.T, claims map[string]any) string {
	t.Helper()
	hdr, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": "k1"})
	body, _ := json.Marshal(claims)
	signing := b64(hdr) + "." + b64(body)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.priv, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + b64(sig)
}

// useIDP — харнесс дэх SSO клиентийг энэ provider руу заана.
func (h *harness) useIDP(t *testing.T, idp *fakeIDP, autoSignup bool) {
	t.Helper()
	c := ssoclient.New([]ssoclient.Provider{{Key: "sso", Name: "Тест SSO", Issuer: idp.issuer, ClientID: "cid", ClientSecret: "sec"}})
	authH := h.authHandler
	*h.sso = *handlers.NewSSO(c, authH, "https://portal.mn", autoSignup, false, "тест-нууц")
}

func TestSSOProvidersList(t *testing.T) {
	h := newHarness(t)
	// Тохируулаагүй үед хоосон.
	w := h.do(t, nil, http.MethodGet, "/api/auth/sso/providers", nil)
	var out map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if len(out["providers"].([]any)) != 0 {
		t.Fatalf("тохируулаагүй provider = %v", out)
	}
	idp := startIDP(t)
	h.useIDP(t, idp, false)
	w = h.do(t, nil, http.MethodGet, "/api/auth/sso/providers", nil)
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	list := out["providers"].([]any)
	if len(list) != 1 || list[0].(map[string]any)["key"] != "sso" {
		t.Fatalf("providers = %v", out)
	}
}

func TestSSOLoginFlow(t *testing.T) {
	h := newHarness(t)
	idp := startIDP(t)
	h.useIDP(t, idp, false)
	// Бүртгэлтэй хэрэглэгч (имэйл нь IdP-ийнхтэй тааруулна, том/жижиг үсэг ялгаагүй).
	existing := h.signup(t, "ssouser")
	ctx := context.Background()
	if _, err := h.owner.Exec(ctx, `UPDATE users SET email = 'sso-хэрэглэгч@x.mn' WHERE id = $1::uuid`, existing.userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = h.owner.Exec(ctx, `DELETE FROM users WHERE email = 'sso-хэрэглэгч@x.mn'`) })

	// 1. Start — provider руу шилжинэ, state cookie тавигдана.
	w := h.do(t, nil, http.MethodGet, "/api/auth/sso/sso/start?next=/dashboard", nil)
	if w.Code != http.StatusFound {
		t.Fatalf("start = %d: %s", w.Code, w.Body.String())
	}
	loc, err := url.Parse(w.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	q := loc.Query()
	if q.Get("code_challenge_method") != "S256" || q.Get("state") == "" || q.Get("nonce") == "" || q.Get("client_id") != "cid" {
		t.Fatalf("authorize URL = %s", loc)
	}
	idp.nonce = q.Get("nonce")
	var stateCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "nexus_sso" {
			stateCookie = c
		}
	}
	if stateCookie == nil || !stateCookie.HttpOnly {
		t.Fatalf("state cookie = %v", stateCookie)
	}

	// 2. Callback — session үүснэ.
	cb := func(cookie *http.Cookie, query string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(http.MethodGet, "/api/auth/sso/sso/callback?"+query, nil)
		if cookie != nil {
			r.AddCookie(cookie)
		}
		return recordRequest(h, r)
	}
	w = cb(stateCookie, "code=good&state="+url.QueryEscape(q.Get("state")))
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "https://portal.mn/dashboard" {
		t.Fatalf("callback = %d %s: %s", w.Code, w.Header().Get("Location"), w.Body.String())
	}
	var sessCookie string
	for _, c := range w.Result().Cookies() {
		if c.Name == auth.CookieName {
			sessCookie = c.Value
		}
	}
	if sessCookie == "" {
		t.Fatal("SSO нэвтрэлтийн дараа session алга")
	}
	s := &session{h: h, cookie: sessCookie}
	me := s.json(t, http.MethodGet, "/api/me", nil)
	if me["user"].(map[string]any)["email"] != "sso-хэрэглэгч@x.mn" {
		t.Fatalf("нэвтэрсэн хэрэглэгч = %v", me["user"])
	}

	// 3. Татгалзлууд.
	cases := []struct {
		name, query string
		cookie      *http.Cookie
	}{
		{"state cookie-гүй", "code=good&state=" + q.Get("state"), nil},
		{"state зөрүү", "code=good&state=өөр", stateCookie},
		{"provider алдаа буцаав", "error=access_denied&state=" + q.Get("state"), stateCookie},
		{"код буруу", "code=буруу&state=" + q.Get("state"), stateCookie},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := cb(c.cookie, c.query)
			if w.Code != http.StatusSeeOther || !strings.Contains(w.Header().Get("Location"), "/login?error=") {
				t.Fatalf("= %d %s", w.Code, w.Header().Get("Location"))
			}
		})
	}
	// Хуурамч (өөрчилсөн) state cookie — HMAC барина.
	bad := *stateCookie
	bad.Value = stateCookie.Value[:len(stateCookie.Value)-2] + "xx"
	if w := cb(&bad, "code=good&state="+q.Get("state")); !strings.Contains(w.Header().Get("Location"), "/login?error=") {
		t.Fatalf("хуурамч cookie = %s", w.Header().Get("Location"))
	}
	// Үл мэдэх provider.
	if w := h.do(t, nil, http.MethodGet, "/api/auth/sso/байхгүй/start", nil); w.Code != http.StatusNotFound {
		t.Fatalf("үл мэдэх provider = %d", w.Code)
	}
}

func TestSSOJITSignup(t *testing.T) {
	h := newHarness(t)
	idp := startIDP(t)
	idp.email = "htest-sso-jit@x.mn"

	// AUTO_SIGNUP хаалттай — бүртгэлгүй имэйл татгалзана.
	h.useIDP(t, idp, false)
	state, cookie := h.ssoStart(t, idp)
	w := h.ssoCallback(t, cookie, "code=good&state="+url.QueryEscape(state))
	if !strings.Contains(w.Header().Get("Location"), "/login?error=") {
		t.Fatalf("хаалттай JIT = %s", w.Header().Get("Location"))
	}
	var n int
	if err := h.owner.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE email = 'htest-sso-jit@x.mn'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatal("хаалттай үед данс үүсэв")
	}

	// AUTO_SIGNUP нээлттэй — данс үүснэ, session өгнө.
	h.useIDP(t, idp, true)
	state, cookie = h.ssoStart(t, idp)
	w = h.ssoCallback(t, cookie, "code=good&state="+url.QueryEscape(state))
	if w.Code != http.StatusSeeOther || strings.Contains(w.Header().Get("Location"), "error=") {
		t.Fatalf("JIT = %d %s", w.Code, w.Header().Get("Location"))
	}
	if err := h.owner.QueryRow(context.Background(), `SELECT count(*) FROM users WHERE email = 'htest-sso-jit@x.mn'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("JIT данс = %d", n)
	}
	// Нууц үггүй данс — нууц үгээр нэвтэрч болохгүй.
	if w := h.do(t, nil, http.MethodPost, "/api/login",
		map[string]string{"email": "htest-sso-jit@x.mn", "password": "password-12"}); w.Code == 200 {
		t.Fatal("SSO данс нууц үгээр нэвтэрлээ")
	}
}
