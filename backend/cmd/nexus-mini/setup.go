package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/platform/migrate"
	"github.com/jackc/pgx/v5"
	"golang.org/x/term"
)

// cmdSetup — анхны тохируулгын интерактив wizard:
//  1. Postgres superuser холболтоор nexus_* role-ууд + DB үүсгэнэ
//  2. nexus-mini.env бичнэ
//  3. Миграц ажиллуулна
//  4. Платформын админаа бүртгэнэ
func cmdSetup(_ []string) error {
	in := bufio.NewReader(os.Stdin)
	fmt.Println("nexus-mini — анхны тохируулга")
	fmt.Println("─────────────────────────────")

	if _, err := os.Stat("nexus-mini.env"); err == nil {
		fmt.Println("nexus-mini.env аль хэдийн байна — setup өмнө нь хийгдсэн бололтой.")
		if !confirm(in, "Дахин тохируулах уу (role-ууд хэвээр, нууц үг шинэчлэгдэнэ)?") {
			return nil
		}
	}

	superURL := prompt(in, "Postgres superuser холболт", "postgres://postgres@127.0.0.1:5432/postgres")
	dbName := prompt(in, "Үүсгэх өгөгдлийн сангийн нэр", "nexus_mini")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, superURL)
	if err != nil {
		return fmt.Errorf("superuser холболт амжилтгүй: %w", err)
	}
	defer conn.Close(ctx)

	// Role-ууд: байвал нууц үгийг нь шинэчилнэ, байхгүй бол үүсгэнэ.
	pw := map[string]string{"nexus_owner": randHex(), "nexus_app": randHex(), "nexus_admin": randHex()}
	if err := ensureRole(ctx, conn, "nexus_platform", ""); err != nil {
		return err
	}
	for role, p := range pw {
		if err := ensureRole(ctx, conn, role, p); err != nil {
			return err
		}
	}
	if _, err := conn.Exec(ctx, `GRANT nexus_platform TO nexus_admin`); err != nil {
		return fmt.Errorf("grant nexus_platform: %w", err)
	}
	var dbExists bool
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&dbExists); err != nil {
		return err
	}
	if !dbExists {
		if _, err := conn.Exec(ctx,
			fmt.Sprintf(`CREATE DATABASE %s OWNER nexus_owner`, pgx.Identifier{dbName}.Sanitize())); err != nil {
			return fmt.Errorf("DB үүсгэх: %w", err)
		}
		fmt.Printf("✓ %s өгөгдлийн сан үүслээ\n", dbName)
	} else {
		fmt.Printf("✓ %s өгөгдлийн сан байна\n", dbName)
	}

	// superURL-ээс host/port-оо өвлөж холболтын URL-ууд угсарна.
	su, err := url.Parse(superURL)
	if err != nil {
		return err
	}
	host := su.Host
	mkURL := func(role string) string {
		return fmt.Sprintf("postgres://%s:%s@%s/%s", role, pw[role], host, dbName)
	}
	envBody := fmt.Sprintf(`# nexus-mini тохиргоо — `+"`nexus-mini setup`"+` үүсгэв
DATABASE_URL=%s
DATABASE_URL_ADMIN=%s
DATABASE_URL_OWNER=%s
PORT=8084
ENVIRONMENT=development
CATALOG_PATH=%s
`, mkURL("nexus_app"), mkURL("nexus_admin"), mkURL("nexus_owner"), findCatalog())
	if err := os.WriteFile("nexus-mini.env", []byte(envBody), 0o600); err != nil {
		return err
	}
	fmt.Println("✓ nexus-mini.env бичигдлээ (600)")

	if err := migrate.RunAll(mkURL("nexus_owner"), func(f string, a ...any) {
		fmt.Printf("✓ "+f+"\n", a...)
	}); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Платформын админаа бүртгэе:")
	if err := createAdminInteractive(in, mkURL("nexus_owner")); err != nil {
		return err
	}

	fmt.Println()
	fmt.Println("Бэлэн боллоо. Дараагийн алхам:")
	fmt.Println("  nexus-mini serve        # API (:8084)")
	fmt.Println("  cd frontend && pnpm dev # вэб (:3020)")
	return nil
}

func ensureRole(ctx context.Context, conn *pgx.Conn, role, password string) error {
	var exists bool
	if err := conn.QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`, role).Scan(&exists); err != nil {
		return err
	}
	ident := pgx.Identifier{role}.Sanitize()
	var sqlStr string
	switch {
	case password == "" && exists:
		return nil
	case password == "":
		sqlStr = fmt.Sprintf(`CREATE ROLE %s NOLOGIN`, ident)
	case exists:
		sqlStr = fmt.Sprintf(`ALTER ROLE %s LOGIN PASSWORD '%s'`, ident, password)
	default:
		sqlStr = fmt.Sprintf(`CREATE ROLE %s LOGIN PASSWORD '%s'`, ident, password)
	}
	if _, err := conn.Exec(ctx, sqlStr); err != nil {
		return fmt.Errorf("role %s: %w", role, err)
	}
	fmt.Printf("✓ role %s\n", role)
	return nil
}

// findCatalog — каталог файлыг cwd орчмоос хайж абсолют замыг буцаана
// (backend/-ээс ч, репогийн язгуураас ч ажиллуулж болдог байхын тулд).
func findCatalog() string {
	for _, p := range []string{"catalog/apps.json", "../catalog/apps.json"} {
		if abs, err := filepath.Abs(p); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return "catalog/apps.json"
}

func randHex() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}

func prompt(in *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := in.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

func confirm(in *bufio.Reader, label string) bool {
	fmt.Printf("%s [y/N]: ", label)
	line, _ := in.ReadString('\n')
	line = strings.ToLower(strings.TrimSpace(line))
	return line == "y" || line == "yes"
}

// promptPassword — терминал бол нууцалж уншина, pipe бол энгийнээр.
func promptPassword(in *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	if term.IsTerminal(int(syscall.Stdin)) {
		b, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		return string(b), err
	}
	line, err := in.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimSpace(line), nil
}
