package rbac

import (
	"context"
	"fmt"
	"strings"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
)

// SeedTenantRoles — шинэ tenant-д default гурван role: admin ⊃ manager ⊃ user.
// Tenant үүсгэсэн гүйлгээн дотор дуудагдана.
func SeedTenantRoles(ctx context.Context, tx pgx.Tx, tenantID string) (adminRoleID string, err error) {
	rows := [][3]string{
		{"user", "Хэрэглэгч", ""},
		{"manager", "Менежер", "user"},
		{"admin", "Админ", "manager"},
	}
	for _, r := range rows {
		var implies any
		if r[2] != "" {
			implies = r[2]
		}
		var id string
		err = tx.QueryRow(ctx,
			`INSERT INTO roles (tenant_id, code, name, implies)
			 VALUES ($1::uuid, $2::varchar(64), $3::varchar(120), $4::varchar(64))
			 RETURNING id`,
			tenantID, r[0], r[1], implies).Scan(&id)
		if err != nil {
			return "", fmt.Errorf("seed role %s: %w", r[0], err)
		}
		if r[0] == "admin" {
			adminRoleID = id
		}
	}
	return adminRoleID, nil
}

// GrantOnInstall — апп суулгах үеийн default оноолт (docs/02-rbac.md #1):
// admin role бүх permission-ийг scope=all авна; permission бүрийн
// DefaultRoles жагсаалтад заасан role-ууд нэмж авна. "user:own" хэлбэр нь
// тухайн role-д own scope-оор олгоно (permission нь OwnScope зарласан
// байх ёстой).
func GrantOnInstall(ctx context.Context, tx pgx.Tx, tenantID string, perms []nexus.PermissionDefinition) error {
	grant := func(roleCode, permCode, scope string) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO role_permissions (role_id, permission_code, scope)
			SELECT r.id, $2::varchar(128), $3::varchar(3)
			  FROM roles r WHERE r.tenant_id = $1::uuid AND r.code = $4::varchar(64)
			ON CONFLICT (role_id, permission_code)
			DO UPDATE SET scope = CASE WHEN role_permissions.scope = 'all' THEN 'all' ELSE excluded.scope END`,
			tenantID, permCode, scope, roleCode)
		return err
	}
	for _, p := range perms {
		if err := grant("admin", p.Code, "all"); err != nil {
			return fmt.Errorf("grant %s to admin: %w", p.Code, err)
		}
		for _, dr := range p.DefaultRoles {
			role, scope := dr, "all"
			if r, s, ok := strings.Cut(dr, ":"); ok {
				role = r
				if s == "own" && p.OwnScope {
					scope = "own"
				}
			}
			if err := grant(role, p.Code, scope); err != nil {
				return fmt.Errorf("grant %s to %s: %w", p.Code, role, err)
			}
		}
	}
	return nil
}

// RevokeOnUninstall — модулийн бүх permission-ий оноолтыг tenant-аас хасна.
func RevokeOnUninstall(ctx context.Context, tx pgx.Tx, tenantID, moduleID string) error {
	_, err := tx.Exec(ctx, `
		DELETE FROM role_permissions rp
		 USING roles r, permissions p
		 WHERE rp.role_id = r.id AND r.tenant_id = $1::uuid
		   AND p.code = rp.permission_code AND p.module_id = $2::varchar(128)`,
		tenantID, moduleID)
	return err
}
