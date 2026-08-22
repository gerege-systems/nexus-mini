// Package config — орчны хувьсагчаас уншдаг тохиргоо.
package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/pkg/registry"
)

type Config struct {
	// DatabaseURL — nexus_app role (RLS үйлчилнэ). Апп-ын бүх tenant урсгал.
	DatabaseURL string
	// DatabaseURLAdmin — nexus_admin role (nexus_platform гишүүн). Boot sync
	// ба платформын админ API.
	DatabaseURLAdmin string
	// DatabaseURLAuth — nexus_auth role: зөвхөн pre-auth definer функцууд
	// (session, signup, handover). Апп/модулийн SQL энэ pool-д хүрдэггүй.
	DatabaseURLAuth string
	Port            string
	Env             string // development | production
	// CatalogPath — локал index файл (registry хүрэхгүй/хоосон үеийн fallback).
	CatalogPath string
	// RegistryURL — app store registry-ийн index.json URL. Default: gerege-systems/nexus-registry (GitHub raw)
	// (gerege-systems/nexus-registry). "off" бол зөвхөн локал файл.
	RegistryURL string
	// RegistryKeys — index-ийн гарын үсгийн нийтийн түлхүүрүүд (base64, таслал).
	// Default URL-д default түлхүүр; өөрийн registry бол заавал өгнө.
	RegistryKeys string
	// RegistryCacheDir — татсан index-ийн кэш (offline fallback, ETag).
	RegistryCacheDir string
	// CookieSecure — production дээр true.
	CookieSecure bool
	// PortalURL — portal-ийн гадаад URL (админ панелээс impersonation
	// handover холбоос үүсгэхэд). Админ панель тусдаа домэйн дээр байдаг.
	PortalURL string
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DatabaseURLAdmin: os.Getenv("DATABASE_URL_ADMIN"),
		DatabaseURLAuth:  os.Getenv("DATABASE_URL_AUTH"),
		Port:             getenv("PORT", "8084"),
		Env:              getenv("ENVIRONMENT", "development"),
		CatalogPath:      getenv("CATALOG_PATH", "../catalog/index.json"), // cwd=backend (make serve); systemd/Docker env-ээр абсолют зам
		RegistryURL:      getenv("REGISTRY_URL", registry.DefaultURL),
		RegistryKeys:     os.Getenv("REGISTRY_KEYS"),
		RegistryCacheDir: getenv("REGISTRY_CACHE_DIR", ".registry-cache"),
		PortalURL:        getenv("PORTAL_URL", "http://localhost:3020"),
	}
	c.CookieSecure = c.Env == "production"
	if c.RegistryKeys == "" && c.RegistryURL == registry.DefaultURL {
		c.RegistryKeys = strings.Join(registry.DefaultKeys, ",")
	}
	if c.RegistryURL != "off" && c.RegistryKeys == "" {
		return c, fmt.Errorf("REGISTRY_URL өгсөн бол REGISTRY_KEYS (нийтийн түлхүүр) заавал")
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL тохируулаагүй байна")
	}
	if c.DatabaseURLAdmin == "" {
		return c, fmt.Errorf("DATABASE_URL_ADMIN тохируулаагүй байна")
	}
	if c.Env == "production" {
		// Production хамгаалалт: cookie Secure, portal URL https — үгүй бол
		// асахаас татгалзана (OGN-ийн config/production.go загвар).
		if !strings.HasPrefix(c.PortalURL, "https://") {
			return c, fmt.Errorf("ENVIRONMENT=production үед PORTAL_URL https:// байх ёстой (одоо %q)", c.PortalURL)
		}
		if os.Getenv("ADMIN_PASSWORD") != "" {
			log.Println("АНХААР: production env-д ADMIN_PASSWORD байна — анхны админ үүссэний дараа env-ээс устга")
		}
	}
	if c.DatabaseURLAuth == "" {
		return c, fmt.Errorf("DATABASE_URL_AUTH тохируулаагүй байна (nexus_auth role — deploy/01-roles.sql, .env.example)")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
