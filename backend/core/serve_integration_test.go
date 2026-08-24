package core

// Сервер бүхэлдээ: cmdServe амьд асаж, route-ууд угтаж, SIGTERM дээр
// graceful унтарна. Мөн CLI-ийн коммандууд (manifest, withEnv, migrate +
// анхны админ). Бодит DB шаардана (make check-db).

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5/pgxpool"
)

func dbEnv(t *testing.T) (app, admin, auth, owner string) {
	t.Helper()
	app, admin = os.Getenv("NEXUS_TEST_DATABASE_URL"), os.Getenv("NEXUS_TEST_DATABASE_URL_ADMIN")
	auth, owner = os.Getenv("NEXUS_TEST_DATABASE_URL_AUTH"), os.Getenv("NEXUS_TEST_DATABASE_URL_OWNER")
	if app == "" || admin == "" || auth == "" || owner == "" {
		if os.Getenv("NEXUS_TEST_REQUIRE_DB") != "" {
			t.Fatal("NEXUS_TEST_DATABASE_URL / _ADMIN / _AUTH / _OWNER шаардлагатай")
		}
		t.Skip("DB тохируулаагүй — алгасав")
	}
	return
}

func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return fmt.Sprint(l.Addr().(*net.TCPAddr).Port)
}

// TestServeLifecycle — сервер асаж, HTTP хариулж, SIGTERM дээр буцна.
func TestServeLifecycle(t *testing.T) {
	appURL, adminURL, authURL, ownerURL := dbEnv(t)
	port := freePort(t)
	for k, v := range map[string]string{
		"DATABASE_URL": appURL, "DATABASE_URL_ADMIN": adminURL, "DATABASE_URL_AUTH": authURL,
		"DATABASE_URL_OWNER": ownerURL, "PORT": port, "LISTEN_ADDR": "127.0.0.1",
		"ENVIRONMENT": "development", "REGISTRY_URL": "off", "CATALOG_PATH": filepath.Join(t.TempDir(), "жок.json"),
		"REGISTRY_CACHE_DIR": t.TempDir(), "PORTAL_URL": "http://127.0.0.1:" + port,
		"ADMIN_EMAIL": "", "ADMIN_PASSWORD": "",
	} {
		t.Setenv(k, v)
	}
	// Тест процессыг SIGTERM-ээс хамгаална (cmdServe өөрөө барьж авна).
	guard := make(chan os.Signal, 1)
	signal.Notify(guard, syscall.SIGTERM)
	defer signal.Stop(guard)

	done := make(chan error, 1)
	go func() { done <- cmdServe(nil) }()

	base := "http://127.0.0.1:" + port
	client := &http.Client{Timeout: 2 * time.Second}
	var up bool
	for i := 0; i < 60; i++ {
		if resp, err := client.Get(base + "/health"); err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			if resp.StatusCode == 200 && strings.TrimSpace(string(body)) == "ok" {
				up = true
				break
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	if !up {
		t.Fatal("сервер асаагүй")
	}
	// Маршрутууд: нээлттэй, хамгаалалттай, OIDC, аюулгүй байдлын толгойнууд.
	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/api/catalog", 200},
		{http.MethodGet, "/api/me", 401},
		{http.MethodGet, "/api/menu", 401},
		{http.MethodGet, "/api/admin/tenants", 401},
		{http.MethodGet, "/api/oauth2/.well-known/openid-configuration", 200},
		{http.MethodGet, "/api/oauth2/jwks", 200},
		{http.MethodGet, "/api/auth/sso/providers", 200},
		{http.MethodGet, "/api/auth/handover?token=x", 405}, // зөвхөн POST
		{http.MethodGet, "/байхгүй", 404},
	}
	for _, c := range cases {
		req, _ := http.NewRequest(c.method, base+c.path, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("%s %s: %v", c.method, c.path, err)
		}
		resp.Body.Close()
		if resp.StatusCode != c.want {
			t.Errorf("%s %s = %d, хүлээсэн %d", c.method, c.path, resp.StatusCode, c.want)
		}
		if resp.Header.Get("X-Frame-Options") != "DENY" {
			t.Errorf("%s: аюулгүй байдлын толгой алга", c.path)
		}
	}
	// CSRF: өөр Origin-той бичих хүсэлт 403.
	req, _ := http.NewRequest(http.MethodPost, base+"/api/login", strings.NewReader(`{}`))
	req.Header.Set("Origin", "https://evil.mn")
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("өөр origin = %d (403 хүлээсэн)", resp.StatusCode)
	}
	// Rate limit: login-д 10/мин.
	var limited bool
	for i := 0; i < 14; i++ {
		req, _ := http.NewRequest(http.MethodPost, base+"/api/login", strings.NewReader(`{"email":"a@b.mn","password":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests {
			limited = true
			break
		}
	}
	if !limited {
		t.Fatal("login rate limit ажиллаагүй")
	}

	// Graceful shutdown.
	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("cmdServe буцахдаа алдаа: %v", err)
		}
	case <-time.After(15 * time.Second):
		t.Fatal("SIGTERM дээр унтарсангүй")
	}
	// Унтарсны дараа порт хаагдсан.
	if _, err := client.Get(base + "/health"); err == nil {
		t.Fatal("унтарсны дараа ч хариулж байна")
	}
}

func TestServeFailsOnBadConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_URL_ADMIN", "")
	t.Setenv("DATABASE_URL_AUTH", "")
	if err := cmdServe(nil); err == nil {
		t.Fatal("тохиргоогүй үед сервер асав")
	}
}

// TestMigrateAndAdminBootstrap — миграц + env-ээс анхны платформ админ.
func TestMigrateAndAdminBootstrap(t *testing.T) {
	_, _, _, ownerURL := dbEnv(t)
	ctx := context.Background()
	owner, err := pgxpool.New(ctx, ownerURL)
	if err != nil {
		t.Fatal(err)
	}
	clean := func() {
		_, _ = owner.Exec(ctx, `DELETE FROM users WHERE email LIKE 'boot-%'`)
		_, _ = owner.Exec(ctx, `DELETE FROM users WHERE email = 'a@b.mn'`)
	}
	clean()
	t.Cleanup(func() {
		clean()
		owner.Close()
	})
	t.Setenv("DATABASE_URL_OWNER", ownerURL)

	// Аль хэдийн платформ админ байвал ШИНЭЭР үүсгэхгүй.
	var already bool
	if err := owner.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE platform_admin)`).Scan(&already); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ADMIN_EMAIL", "boot-new@x.mn")
	t.Setenv("ADMIN_NAME", "Бүүт")
	t.Setenv("ADMIN_PASSWORD", "password-12")
	if err := cmdMigrate(nil); err != nil {
		t.Fatalf("cmdMigrate: %v", err)
	}
	var exists bool
	if err := owner.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = 'boot-new@x.mn')`).Scan(&exists); err != nil {
		t.Fatal(err)
	}
	if already && exists {
		t.Fatal("админ байхад шинэ админ үүсгэв")
	}

	// Байгаа хэрэглэгчийг өргөмжлөх — нууц үгэнд хүрэхгүй.
	hash, err := password.Hash("password-12")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := owner.Exec(ctx, `INSERT INTO users (email, password_hash, name) VALUES ('boot-promote@x.mn',$1,'Б')`, hash); err != nil {
		t.Fatal(err)
	}
	if err := upsertAdmin(ownerURL, "boot-promote@x.mn", "Б", "Other-Pass1!"); err != nil {
		t.Fatal(err)
	}
	var isAdmin bool
	var storedHash string
	if err := owner.QueryRow(ctx, `SELECT platform_admin, password_hash FROM users WHERE email = 'boot-promote@x.mn'`).Scan(&isAdmin, &storedHash); err != nil {
		t.Fatal(err)
	}
	if !isAdmin || storedHash != hash {
		t.Fatalf("өргөмжлөлт = %v, нууц үг өөрчлөгдсөн: %v", isAdmin, storedHash != hash)
	}
	// Шинэ админ үүсгэх.
	if err := upsertAdmin(ownerURL, "boot-fresh@x.mn", "Шинэ", "password-12"); err != nil {
		t.Fatal(err)
	}
	if err := owner.QueryRow(ctx, `SELECT platform_admin FROM users WHERE email = 'boot-fresh@x.mn'`).Scan(&isAdmin); err != nil {
		t.Fatal(err)
	}
	if !isAdmin {
		t.Fatal("шинэ админ үүсээгүй")
	}
	// Буруу оролт.
	for _, c := range [][3]string{{"тийм-биш", "Н", "password-12"}, {"a@b.mn", "", "password-12"}, {"a@b.mn", "Н", "Ab1!"}, {"a@b.mn", "Н", "Нууцүг123!"}, {"a@b.mn", "Н", "passwordonly"}} {
		if err := upsertAdmin(ownerURL, c[0], c[1], c[2]); err == nil {
			t.Errorf("буруу админ оролт хүлээн авагдав: %v", c)
		}
	}
	// ADMIN_* байхгүй бол чимээгүй алгасна.
	t.Setenv("ADMIN_EMAIL", "")
	t.Setenv("ADMIN_PASSWORD", "")
	if err := ensureAdminFromEnv(ownerURL); err != nil {
		t.Fatalf("ADMIN_* байхгүй: %v", err)
	}
}

func TestWithEnvAndManifest(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "nexus-mini.env")
	if err := os.WriteFile(envPath, []byte("ТЕСТ_ХУВЬСАГЧ=утга\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ТЕСТ_ХУВЬСАГЧ", "")
	var gotArgs []string
	if err := withEnv([]string{"--env", envPath, "нэмэлт"}, func(a []string) error {
		gotArgs = a
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("ТЕСТ_ХУВЬСАГЧ") != "утга" {
		t.Fatal("env файл ачаалагдсангүй")
	}
	if len(gotArgs) != 1 || gotArgs[0] != "нэмэлт" {
		t.Fatalf("--env хассаны дараах аргумент = %v", gotArgs)
	}
	// Байхгүй env файл — алдаагүй (default зам).
	if err := withEnv(nil, func([]string) error { return nil }); err != nil {
		t.Fatalf("env файлгүй: %v", err)
	}

	// manifest — бүртгэгдсэн модульгүй бол алдаа.
	if err := cmdManifest([]string{"байхгүй"}); err == nil {
		t.Fatal("байхгүй модульд манифест гарав")
	}
	if len(nexus.Registered()) > 0 {
		if err := cmdManifest(nil); err != nil {
			t.Fatalf("cmdManifest: %v", err)
		}
	}
}

// TestMainHelp — CLI-ийн оролт (usage) ба үл мэдэх комманд.
func TestMainHelpPath(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	os.Args = []string{"nexus-mini", "help"}
	Main() // модульгүй ч ажиллана
	_ = w.Close()
	os.Stdout = old
	out, _ := io.ReadAll(r)
	if !strings.Contains(string(out), "make migrate") || !strings.Contains(string(out), "make serve") {
		t.Fatalf("usage:\n%s", out)
	}
}
