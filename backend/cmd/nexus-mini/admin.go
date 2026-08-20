package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/platform/migrate"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/password"
	"github.com/jackc/pgx/v5"
)

// cmdMigrate — nexus-mini migrate
func cmdMigrate(_ []string) error {
	ownerURL := os.Getenv("DATABASE_URL_OWNER")
	if ownerURL == "" {
		return fmt.Errorf("DATABASE_URL_OWNER алга — эхлээд `nexus-mini setup` ажиллуул")
	}
	return migrate.RunAll(ownerURL, func(f string, a ...any) { fmt.Printf(f+"\n", a...) })
}

// cmdAdmin — nexus-mini admin [--email --name --password | --from-env]
// Имэйл бүртгэлтэй бол платформын админ болгож өргөмжилнө (нууц үг хэвээр);
// байхгүй бол шинээр үүсгэнэ.
func cmdAdmin(args []string) error {
	fs := flag.NewFlagSet("admin", flag.ContinueOnError)
	email := fs.String("email", "", "имэйл")
	name := fs.String("name", "", "нэр")
	pass := fs.String("password", "", "нууц үг (8+)")
	fromEnv := fs.Bool("from-env", false, "ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD хувьсагчаас (byхгүй бол чимээгүй алгасна)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ownerURL := os.Getenv("DATABASE_URL_OWNER")
	if ownerURL == "" {
		return fmt.Errorf("DATABASE_URL_OWNER алга — эхлээд `nexus-mini setup` ажиллуул")
	}

	if *fromEnv {
		// Docker compose гэх мэт интерактив бус орчинд: хувьсагч өгөөгүй
		// эсвэл платформын админ аль хэдийн байгаа бол юу ч хийхгүй.
		e, n, p := os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_NAME"), os.Getenv("ADMIN_PASSWORD")
		if e == "" || p == "" {
			return nil
		}
		if n == "" {
			n = "Admin"
		}
		exists, err := platformAdminExists(ownerURL)
		if err != nil {
			return err
		}
		if exists {
			return nil
		}
		return upsertAdmin(ownerURL, e, n, p)
	}

	in := bufio.NewReader(os.Stdin)
	if *email == "" {
		*email = prompt(in, "Имэйл", "")
	}
	if *name == "" {
		*name = prompt(in, "Нэр", "")
	}
	if *pass == "" {
		p, err := promptPassword(in, "Нууц үг (8+)")
		if err != nil {
			return err
		}
		*pass = p
	}
	return upsertAdmin(ownerURL, *email, *name, *pass)
}

// createAdminInteractive — setup wizard-аас дуудагдана.
func createAdminInteractive(in *bufio.Reader, ownerURL string) error {
	for {
		email := prompt(in, "  Имэйл", "")
		name := prompt(in, "  Нэр", "")
		pass, err := promptPassword(in, "  Нууц үг (8+)")
		if err != nil {
			return err
		}
		if err := upsertAdmin(ownerURL, email, name, pass); err != nil {
			fmt.Printf("  ✗ %v — дахин оролдъё\n", err)
			continue
		}
		return nil
	}
}

func platformAdminExists(ownerURL string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		return false, err
	}
	defer conn.Close(ctx)
	var exists bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE platform_admin)`).Scan(&exists)
	return exists, err
}

func upsertAdmin(ownerURL, email, name, pass string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	name = strings.TrimSpace(name)
	if !strings.Contains(email, "@") || name == "" {
		return fmt.Errorf("зөв имэйл ба нэр шаардлагатай")
	}
	if len(pass) < 8 {
		return fmt.Errorf("нууц үг 8+ тэмдэгт байх ёстой")
	}
	hash, err := password.Hash(pass)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	conn, err := pgx.Connect(ctx, ownerURL)
	if err != nil {
		return fmt.Errorf("owner холболт: %w", err)
	}
	defer conn.Close(ctx)

	var existed bool
	err = conn.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE email = $1)`, email).Scan(&existed)
	if err != nil {
		return err
	}
	if existed {
		// Бүртгэлтэй хэрэглэгчийг өргөмжилнө — нууц үгэнд нь хүрэхгүй.
		if _, err := conn.Exec(ctx,
			`UPDATE users SET platform_admin = true WHERE email = $1`, email); err != nil {
			return err
		}
		fmt.Printf("✓ %s платформын админ болов (нууц үг хэвээр)\n", email)
		return nil
	}
	if _, err := conn.Exec(ctx, `
		INSERT INTO users (email, password_hash, name, platform_admin)
		VALUES ($1::varchar(255), $2::varchar(255), $3::varchar(120), true)`,
		email, hash, name); err != nil {
		return err
	}
	fmt.Printf("✓ платформын админ үүслээ: %s\n", email)
	return nil
}
