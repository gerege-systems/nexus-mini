package oidc

import (
	"net/url"
	"strings"
	"testing"
)

// Эдгээр нь DB шаарддаггүй цэвэр логик — OIDC-ийн эмзэг инвариантууд.

func TestAllowsRedirectIsExactMatch(t *testing.T) {
	c := &client{RedirectURIs: []string{"https://app.example.com/cb"}}
	ok := []string{"https://app.example.com/cb"}
	bad := []string{
		"https://app.example.com/cb/",         // төгсгөлийн зураас
		"https://app.example.com/cb?x=1",      // нэмэлт query
		"https://app.example.com/cb/../evil",  // зам
		"https://app.example.com.evil.com/cb", // суффикс домэйн
		"https://evil.com/cb",                 // өөр хост
		"http://app.example.com/cb",           // схем
		"HTTPS://app.example.com/cb",          // регистр
		"",
	}
	for _, u := range ok {
		if !c.allowsRedirect(u) {
			t.Errorf("зөвшөөрөгдсөн redirect татгалзлаа: %q", u)
		}
	}
	for _, u := range bad {
		if c.allowsRedirect(u) {
			t.Errorf("бүртгэлгүй redirect зөвшөөрөгдлөө: %q", u)
		}
	}
}

func TestAllowsScope(t *testing.T) {
	c := &client{Scopes: "openid profile"}
	if !c.allowsScope("openid") || !c.allowsScope("openid profile") {
		t.Error("зөвшөөрөгдсөн scope татгалзлаа")
	}
	// offline_access нь бүртгэлгүй ч тусгайлан зөвшөөрөгддөг.
	if !c.allowsScope("openid offline_access") {
		t.Error("offline_access татгалзлаа")
	}
	if c.allowsScope("openid email") || c.allowsScope("roles") {
		t.Error("бүртгэлгүй scope зөвшөөрөгдлөө")
	}
}

func baseAuthz() url.Values {
	v := url.Values{}
	v.Set("response_type", "code")
	v.Set("client_id", "app")
	v.Set("redirect_uri", "https://app.example.com/cb")
	v.Set("code_challenge_method", "S256")
	v.Set("code_challenge", strings.Repeat("a", 43))
	v.Set("scope", "openid")
	return v
}

func TestParseAuthzRequiresPKCE_S256(t *testing.T) {
	if _, err := parseAuthz(baseAuthz()); err != "" {
		t.Fatalf("хүчинтэй хүсэлт татгалзлаа: %s", err)
	}
	cases := map[string]func(url.Values){
		"PKCE огт байхгүй":       func(v url.Values) { v.Del("code_challenge"); v.Del("code_challenge_method") },
		"challenge дутуу":        func(v url.Values) { v.Del("code_challenge") },
		"plain арга":             func(v url.Values) { v.Set("code_challenge_method", "plain") },
		"challenge хэт богино":   func(v url.Values) { v.Set("code_challenge", strings.Repeat("a", 42)) },
		"challenge хэт урт":      func(v url.Values) { v.Set("code_challenge", strings.Repeat("a", 129)) },
		"challenge буруу тэмдэг": func(v url.Values) { v.Set("code_challenge", strings.Repeat("a", 42)+"$") },
	}
	for name, mut := range cases {
		v := baseAuthz()
		mut(v)
		if _, err := parseAuthz(v); err == "" {
			t.Errorf("%s — PKCE-гүй хүсэлт өнгөрлөө", name)
		}
	}
}

func TestParseAuthzRejectsUnknownScopeAndImplicit(t *testing.T) {
	v := baseAuthz()
	v.Set("scope", "openid admin")
	if _, err := parseAuthz(v); err != "invalid_scope" {
		t.Errorf("танигдахгүй scope өнгөрлөө: %q", err)
	}
	for _, rt := range []string{"token", "id_token", "code token", ""} {
		v := baseAuthz()
		v.Set("response_type", rt)
		if _, err := parseAuthz(v); err != "unsupported_response_type" {
			t.Errorf("response_type=%q өнгөрлөө: %q", rt, err)
		}
	}
}

func TestScopeSubset(t *testing.T) {
	if !scopeSubset("openid", "openid profile") {
		t.Error("дэд олонлог татгалзлаа")
	}
	if scopeSubset("openid email", "openid profile") {
		t.Error("зөвшөөрөгдөөгүй scope дэд олонлог гэж тооцогдлоо")
	}
}

func TestVerifySecret(t *testing.T) {
	plain, hash, err := NewClientSecret()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("шинэ secret sha256 биш: %q", hash)
	}
	if !verifySecret(plain, hash) {
		t.Error("зөв secret татгалзлаа")
	}
	if verifySecret(plain+"x", hash) || verifySecret("", hash) {
		t.Error("буруу secret өнгөрлөө")
	}
}
