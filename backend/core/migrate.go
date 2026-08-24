package core

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/migrate"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/password"
	"github.com/jackc/pgx/v5"
)

// cmdMigrate — make migrate: миграц + env-ээс анхны админ.
func cmdMigrate(_ []string) error {
	ownerURL := os.Getenv("DATABASE_URL_OWNER")
	if ownerURL == "" {
		return fmt.Errorf("DATABASE_URL_OWNER алга — nexus-mini.env-ээ бөглөсөн үү? (.env.example-г хар)")
	}
	if err := migrate.RunAll(ownerURL, func(f string, a ...any) { fmt.Printf(f+"\n", a...) }); err != nil {
		return err
	}
	return ensureAdminFromEnv(ownerURL)
}

// ensureAdminFromEnv — ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD өгөгдсөн бөгөөд
// платформын админ огт байхгүй үед л үүсгэнэ. Байгаа бол юунд ч хүрэхгүй —
// env доторх нууц үг хожим өөрчлөгдсөн ч дахин бичихгүй.
func ensureAdminFromEnv(ownerURL string) error {
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
	if err := password.Validate(pass); err != nil {
		return fmt.Errorf("ADMIN_PASSWORD: %w", err)
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
