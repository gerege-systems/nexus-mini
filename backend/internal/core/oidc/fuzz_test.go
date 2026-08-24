package oidc

// Fuzz: authorize-ийн параметр задлалт болон JWT шалгалт ямар ч оролтод
// panic хийхгүй; батлагдсан бол PKCE/scope-ийн дүрэм биелнэ.

import (
	"crypto/rand"
	"crypto/rsa"
	"net/url"
	"strings"
	"testing"
)

// testKeyFuzz — fuzz-ийн бүх давталтад нэг түлхүүр (RSA үүсгэлт удаан).
var testKeyFuzz = func() *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	return k
}()

func FuzzParseAuthz(f *testing.F) {
	f.Add("response_type=code&client_id=a&redirect_uri=https%3A%2F%2Fa.mn%2Fcb&code_challenge=" + strings.Repeat("a", 43) + "&code_challenge_method=S256")
	f.Add("response_type=token&client_id=a")
	f.Add("")
	f.Fuzz(func(t *testing.T, raw string) {
		q, err := url.ParseQuery(raw)
		if err != nil {
			return
		}
		a, errMsg := parseAuthz(q)
		if errMsg != "" {
			return
		}
		// Батлагдсан хүсэлт: PKCE S256 заавал, scope allowlist дотор.
		if !pkceRe.MatchString(a.Challenge) || q.Get("code_challenge_method") != "S256" {
			t.Fatalf("PKCE-гүй хүсэлт нэвтэрлээ: %q", raw)
		}
		if q.Get("response_type") != "code" {
			t.Fatalf("code биш response_type нэвтэрлээ: %q", q.Get("response_type"))
		}
		for _, s := range strings.Fields(a.Scope) {
			if !knownScopes[s] {
				t.Fatalf("үл мэдэх scope нэвтэрлээ: %q", s)
			}
		}
	})
}

func FuzzVerifyJWT(f *testing.F) {
	f.Add("a.b.c")
	f.Add("")
	f.Add(strings.Repeat("x", 100))
	f.Fuzz(func(t *testing.T, token string) {
		k := &signingKey{kid: "k", priv: testKeyFuzz}
		if claims, ok := verifyJWT(k, token); ok && claims == nil {
			t.Fatal("ok=true атал claims nil")
		}
	})
}
