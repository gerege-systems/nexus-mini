// Package config — орчны хувьсагчаас уншдаг тохиргоо.
package config

import (
	"fmt"
	"os"
)

type Config struct {
	// DatabaseURL — nexus_app role (RLS үйлчилнэ). Апп-ын бүх tenant урсгал.
	DatabaseURL string
	// DatabaseURLAdmin — nexus_admin role (nexus_platform гишүүн). Boot sync
	// ба платформын админ API.
	DatabaseURLAdmin string
	Port             string
	Env              string // development | production
	// CatalogPath — локал каталог файл (registry fallback).
	CatalogPath string
	// RegistryURL — төв app store registry (үе 2; хоосон бол локал каталог).
	RegistryURL string
	// CookieSecure — production дээр true.
	CookieSecure bool
}

func Load() (Config, error) {
	c := Config{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		DatabaseURLAdmin: os.Getenv("DATABASE_URL_ADMIN"),
		Port:             getenv("PORT", "8084"),
		Env:              getenv("ENVIRONMENT", "development"),
		CatalogPath:      getenv("CATALOG_PATH", "catalog/apps.json"),
		RegistryURL:      os.Getenv("REGISTRY_URL"),
	}
	c.CookieSecure = c.Env == "production"
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL тохируулаагүй байна")
	}
	if c.DatabaseURLAdmin == "" {
		return c, fmt.Errorf("DATABASE_URL_ADMIN тохируулаагүй байна")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
