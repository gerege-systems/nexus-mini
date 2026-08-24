package config

// Тохиргооны шалгуурууд: заавал талбарууд, production-ы хамгаалалт,
// default утгууд.

import (
	"strings"
	"testing"
)

func base(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"DATABASE_URL": "postgres://app@localhost/db", "DATABASE_URL_ADMIN": "postgres://admin@localhost/db",
		"DATABASE_URL_AUTH": "postgres://auth@localhost/db", "ENVIRONMENT": "development",
		"PORT": "", "PORTAL_URL": "", "REGISTRY_URL": "", "REGISTRY_KEYS": "", "CATALOG_PATH": "",
		"REGISTRY_CACHE_DIR": "", "SSO_ISSUER": "", "GOOGLE_CLIENT_ID": "", "SSO_AUTO_SIGNUP": "",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	base(t)
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Port != "8084" || c.Env != "development" || c.CookieSecure {
		t.Fatalf("default = %+v", c)
	}
	if c.CatalogPath != "../catalog/index.json" || c.RegistryCacheDir != ".registry-cache" {
		t.Fatalf("зам default = %q %q", c.CatalogPath, c.RegistryCacheDir)
	}
	if c.PortalURL != "http://localhost:3020" {
		t.Fatalf("PortalURL = %q", c.PortalURL)
	}
	// Default registry-д default түлхүүр автоматаар тавигдана.
	if c.RegistryKeys == "" || !strings.Contains(c.RegistryURL, "nexus-registry") {
		t.Fatalf("registry = %q / %q", c.RegistryURL, c.RegistryKeys)
	}
}

func TestLoadRequiresDatabaseURLs(t *testing.T) {
	for _, missing := range []string{"DATABASE_URL", "DATABASE_URL_ADMIN", "DATABASE_URL_AUTH"} {
		t.Run(missing, func(t *testing.T) {
			base(t)
			t.Setenv(missing, "")
			if _, err := Load(); err == nil || !strings.Contains(err.Error(), missing) {
				t.Fatalf("%s дутуу үед алдаа = %v", missing, err)
			}
		})
	}
}

func TestProductionGuard(t *testing.T) {
	base(t)
	t.Setenv("ENVIRONMENT", "production")
	t.Setenv("PORTAL_URL", "http://insecure.mn")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("production дээр http PORTAL_URL = %v", err)
	}
	t.Setenv("PORTAL_URL", "https://nexus.mn")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.CookieSecure {
		t.Fatal("production дээр CookieSecure=false")
	}
}

func TestCustomRegistryRequiresKeys(t *testing.T) {
	base(t)
	t.Setenv("REGISTRY_URL", "https://bold.mn/index.json")
	if _, err := Load(); err == nil || !strings.Contains(err.Error(), "REGISTRY_KEYS") {
		t.Fatalf("түлхүүргүй registry = %v", err)
	}
	t.Setenv("REGISTRY_KEYS", "AAAA")
	if _, err := Load(); err != nil {
		t.Fatalf("түлхүүртэй registry = %v", err)
	}
	// "off" — түлхүүр шаардахгүй.
	base(t)
	t.Setenv("REGISTRY_URL", "off")
	if _, err := Load(); err != nil {
		t.Fatalf("registry off = %v", err)
	}
}

func TestSSOFlags(t *testing.T) {
	base(t)
	t.Setenv("SSO_AUTO_SIGNUP", "true")
	t.Setenv("GOOGLE_CLIENT_ID", "gid")
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !c.SSOAutoSignup || c.GoogleClientID != "gid" || c.SSOName != "SSO" {
		t.Fatalf("SSO = %+v", c)
	}
	base(t)
	t.Setenv("SSO_AUTO_SIGNUP", "TRUE") // зөвхөн яг "true"
	c, _ = Load()
	if c.SSOAutoSignup {
		t.Fatal("SSO_AUTO_SIGNUP=TRUE идэвхжив (зөвхөн 'true' байх ёстой)")
	}
}
