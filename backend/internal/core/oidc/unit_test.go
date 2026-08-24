package oidc

// OIDC-ийн DB шаарддаггүй хэсгүүд: JWT гарын үсэг/шалгалт, scope, secret,
// алдааны redirect, PKCE/scope-ийн хилүүд.

import (
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"strings"
	"testing"
)

func testKey(t *testing.T) *signingKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return &signingKey{kid: "kid-1", priv: priv}
}

func TestSignAndVerifyJWT(t *testing.T) {
	k := testKey(t)
	tok, err := signJWT(k, map[string]any{"sub": "u1", "aud": "c1"})
	if err != nil {
		t.Fatal(err)
	}
	claims, ok := verifyJWT(k, tok)
	if !ok || claims["sub"] != "u1" || claims["aud"] != "c1" {
		t.Fatalf("verify = %v %v", claims, ok)
	}
	parts := strings.Split(tok, ".")
	bad := []string{
		"", "a.b", tok + "x",
		parts[0] + "." + parts[1] + ".", // гарын үсэггүй
		parts[0] + ".QUJD." + parts[2],  // өөрчилсөн payload
	}
	for _, b := range bad {
		if _, ok := verifyJWT(k, b); ok {
			t.Errorf("гэмтсэн токен батлагдав: %q", b[:min(12, len(b))])
		}
	}
	// Өөр түлхүүрээр батлагдахгүй.
	if _, ok := verifyJWT(testKey(t), tok); ok {
		t.Error("өөр түлхүүрээр батлагдав")
	}
	// JWK бүтэц.
	j := k.jwk()
	if j["kty"] != "RSA" || j["alg"] != "RS256" || j["kid"] != "kid-1" || j["n"] == "" {
		t.Fatalf("jwk = %v", j)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestClientSecretHashing(t *testing.T) {
	plain, hash, err := NewClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Fatalf("hash = %q (sha256 хүлээсэн)", hash)
	}
	if !verifySecret(plain, hash) {
		t.Fatal("зөв secret татгалзав")
	}
	for _, bad := range []string{"", plain + "x", strings.ToUpper(plain)} {
		if verifySecret(bad, hash) {
			t.Errorf("буруу secret батлагдав: %q", bad)
		}
	}
	// Хуучин argon2 мөр (fallback) — password.Hash-ийн формат.
	if verifySecret("x", "$argon2id$v=19$m=65536,t=1,p=4$YWJj$ZGVm") {
		t.Error("буруу argon2 secret батлагдав")
	}
}

func TestScopeSubsetAndHelpers(t *testing.T) {
	cases := []struct {
		want, have string
		ok         bool
	}{
		{"openid", "openid profile", true},
		{"openid profile", "openid", false},
		{"", "openid", true},
		{"openid email", "email openid", true},
	}
	for _, c := range cases {
		if got := scopeSubset(c.want, c.have); got != c.ok {
			t.Errorf("scopeSubset(%q,%q) = %v", c.want, c.have, got)
		}
	}
	if hostOf("https://erp.bold.mn/cb?x=1") != "erp.bold.mn" || hostOf("::бүтэхгүй") != "" {
		t.Error("hostOf буруу")
	}
	u := withErr("https://erp.bold.mn/cb?a=1", "st", "access_denied")
	p, _ := url.Parse(u)
	if p.Query().Get("error") != "access_denied" || p.Query().Get("state") != "st" || p.Query().Get("a") != "1" {
		t.Fatalf("withErr = %s", u)
	}
}

func TestClientAllowsRedirectAndScope(t *testing.T) {
	c := &client{RedirectURIs: []string{"https://a.mn/cb", "http://localhost:3000/cb"}, Scopes: "openid profile email"}
	for _, good := range c.RedirectURIs {
		if !c.allowsRedirect(good) {
			t.Errorf("зөв redirect татгалзав: %s", good)
		}
	}
	for _, bad := range []string{"https://a.mn/cb2", "https://a.mn", "https://a.mn/cb/", "https://a.mn/cb?x=1",
		"https://evil.mn/cb", "HTTPS://A.MN/cb", "https://a.mn:443/cb"} {
		if c.allowsRedirect(bad) {
			t.Errorf("буруу redirect зөвшөөрөгдөв: %s", bad)
		}
	}
	if !c.allowsScope("openid email") || !c.allowsScope("openid offline_access") {
		t.Error("зөвшөөрөгдсөн scope татгалзав")
	}
	if c.allowsScope("openid roles") || c.allowsScope("tenant") {
		t.Error("зөвшөөрөөгүй scope нэвтэрлээ")
	}
}
