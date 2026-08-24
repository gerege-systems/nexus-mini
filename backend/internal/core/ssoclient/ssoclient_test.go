package ssoclient

// SSO client (relying party) — гадны id_token-ийг ХЭЗЭЭ ч сохроор итгэхгүй.
// Тест бүр жинхэнэ HTTP provider (httptest) босгож discovery + JWKS өгнө.

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
	"strings"
	"testing"
	"time"
)

type fakeIDP struct {
	srv    *httptest.Server
	priv   *rsa.PrivateKey
	kid    string
	issuer string
	tokens map[string]string // code → id_token
}

func newIDP(t *testing.T) *fakeIDP {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	f := &fakeIDP{priv: priv, kid: "test-kid", tokens: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]string{
			"issuer": f.issuer, "authorization_endpoint": f.issuer + "/auth",
			"token_endpoint": f.issuer + "/token", "jwks_uri": f.issuer + "/jwks",
			"userinfo_endpoint": f.issuer + "/userinfo",
		})
	})
	mux.HandleFunc("/jwks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{
			"kty": "RSA", "use": "sig", "alg": "RS256", "kid": f.kid,
			"n": b64(priv.PublicKey.N.Bytes()), "e": b64(big.NewInt(int64(priv.PublicKey.E)).Bytes()),
		}}})
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		idt, ok := f.tokens[r.PostFormValue("code")]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_grant"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"id_token": idt, "access_token": "x"})
	})
	f.srv = httptest.NewServer(mux)
	f.issuer = f.srv.URL
	t.Cleanup(f.srv.Close)
	return f
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

// sign — id_token үүсгэнэ; alg/kid/claims-ийг тестээс дарж болно.
func (f *fakeIDP) sign(t *testing.T, claims map[string]any, alg, kid string) string {
	t.Helper()
	hdr, _ := json.Marshal(map[string]string{"alg": alg, "typ": "JWT", "kid": kid})
	body, _ := json.Marshal(claims)
	signing := b64(hdr) + "." + b64(body)
	if alg == "none" {
		return signing + "."
	}
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, f.priv, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + b64(sig)
}

func (f *fakeIDP) claims(aud, nonce string) map[string]any {
	return map[string]any{"iss": f.issuer, "aud": aud, "sub": "u-1", "email": "A@Example.MN",
		"email_verified": true, "name": "Тест", "nonce": nonce, "exp": time.Now().Add(time.Hour).Unix()}
}

func provider(f *fakeIDP) Provider {
	return Provider{Key: "sso", Name: "Test", Issuer: f.issuer, ClientID: "cid", ClientSecret: "sec"}
}

func TestExchangeHappyPath(t *testing.T) {
	f := newIDP(t)
	c := New([]Provider{provider(f)})
	f.tokens["good"] = f.sign(t, f.claims("cid", "n1"), "RS256", f.kid)
	id, err := c.Exchange(context.Background(), provider(f), "https://rp/cb", "good", "verifier", "n1")
	if err != nil {
		t.Fatal(err)
	}
	if id.Subject != "u-1" || id.Email != "a@example.mn" || !id.EmailVerified || id.Name != "Тест" {
		t.Fatalf("identity = %+v (имэйл жижиг үсгээр байх ёстой)", id)
	}
}

func TestExchangeRejectsBadTokens(t *testing.T) {
	f := newIDP(t)
	p := provider(f)
	c := New([]Provider{p})
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	cases := []struct{ name, token, nonce string }{
		{"alg=none", f.sign(t, f.claims("cid", "n1"), "none", f.kid), "n1"},
		{"өөр aud", f.sign(t, f.claims("өөр-клиент", "n1"), "RS256", f.kid), "n1"},
		{"nonce зөрүү", f.sign(t, f.claims("cid", "буруу"), "RS256", f.kid), "n1"},
		{"хугацаа дууссан", f.sign(t, func() map[string]any {
			cl := f.claims("cid", "n1")
			cl["exp"] = time.Now().Add(-2 * time.Hour).Unix()
			return cl
		}(), "RS256", f.kid), "n1"},
		{"өөр issuer", f.sign(t, func() map[string]any {
			cl := f.claims("cid", "n1")
			cl["iss"] = "https://evil.mn"
			return cl
		}(), "RS256", f.kid), "n1"},
		{"үл мэдэх kid", f.sign(t, f.claims("cid", "n1"), "RS256", "хуурмаг-kid"), "n1"},
		{"sub/email алга", f.sign(t, map[string]any{"iss": f.issuer, "aud": "cid", "nonce": "n1",
			"exp": time.Now().Add(time.Hour).Unix()}, "RS256", f.kid), "n1"},
	}
	for i, cs := range cases {
		t.Run(cs.name, func(t *testing.T) {
			code := "c" + string(rune('a'+i))
			f.tokens[code] = cs.token
			if _, err := c.Exchange(context.Background(), p, "https://rp/cb", code, "v", cs.nonce); err == nil {
				t.Fatal("хүлээн зөвшөөрөгдөх ёсгүй байсан")
			}
		})
	}
	// Өөр түлхүүрээр гарын үсэглэсэн (kid таарсан ч) — гарын үсэг унана.
	f2 := &fakeIDP{priv: other, kid: f.kid, issuer: f.issuer, tokens: map[string]string{}}
	f.tokens["forged"] = f2.sign(t, f.claims("cid", "n1"), "RS256", f.kid)
	if _, err := c.Exchange(context.Background(), p, "https://rp/cb", "forged", "v", "n1"); err == nil {
		t.Fatal("хуурамч гарын үсэг батлагдав")
	}
}

func TestDiscoveryIssuerMismatch(t *testing.T) {
	f := newIDP(t)
	p := provider(f)
	p.Issuer = f.issuer + "/өөр" // discovery-ийн issuer таарахгүй
	c := New([]Provider{p})
	if _, err := c.AuthURL(context.Background(), p, "https://rp/cb", "s", "n", "ch"); err == nil {
		t.Fatal("issuer зөрүүтэй discovery хүлээн авагдав")
	}
}

func TestAuthURLCarriesPKCEAndState(t *testing.T) {
	f := newIDP(t)
	p := provider(f)
	c := New([]Provider{p})
	u, err := c.AuthURL(context.Background(), p, "https://rp/cb", "st", "no", "chal")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"response_type=code", "client_id=cid", "state=st", "nonce=no",
		"code_challenge=chal", "code_challenge_method=S256", "scope=openid"} {
		if !strings.Contains(u, want) {
			t.Errorf("AuthURL-д %q алга: %s", want, u)
		}
	}
}

func TestPKCEAndPublicList(t *testing.T) {
	v, ch := NewPKCE()
	sum := sha256.Sum256([]byte(v))
	if ch != b64(sum[:]) || len(v) < 43 {
		t.Fatal("PKCE challenge S256 биш")
	}
	if a, b := RandString(), RandString(); a == b {
		t.Fatal("RandString давтагдав")
	}
	c := New([]Provider{{Key: "google", Name: "Google", Issuer: "https://x", ClientID: "c"},
		{Key: "хоосон", Name: "Хоосон"}}) // issuer/client_id-гүй нь орохгүй
	if got := c.Public(); len(got) != 1 || got[0]["key"] != "google" {
		t.Fatalf("Public = %v", got)
	}
	if _, ok := c.Get("хоосон"); ok {
		t.Fatal("тохируулаагүй provider бүртгэгдэв")
	}
}

func TestAudContains(t *testing.T) {
	cases := []struct {
		aud  any
		want bool
	}{{"cid", true}, {"өөр", false}, {[]any{"a", "cid"}, true}, {[]any{"a"}, false}, {nil, false}, {42, false}}
	for _, c := range cases {
		if got := audContains(c.aud, "cid"); got != c.want {
			t.Errorf("audContains(%v) = %v", c.aud, got)
		}
	}
}
