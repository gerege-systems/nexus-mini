// Package appstore — каталог sync, суулгалт, "апп суусан эсэх" gate.
package appstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CatalogEntry — каталог файлын (эсвэл registry-ийн, үе 2) нэг апп.
type CatalogEntry struct {
	ID          string `json:"id"`
	ShortID     string `json:"short_id"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Publisher   string `json:"publisher"`
	GoModule    string `json:"go_module"`
}

type catalogFile struct {
	Apps []CatalogEntry `json:"apps"`
}

// backfillGrants — шинээр гарч ирсэн permission-уудын default оноолтыг
// тухайн модулийг суулгасан (core бол бүх) tenant-д хийнэ. Admin pool
// (nexus_platform) RLS-ийг давдаг тул tenant ctx шаардлагагүй.
func backfillGrants(ctx context.Context, admin *pgxpool.Pool, newPerms map[string][]nexus.PermissionDefinition) error {
	for moduleID, perms := range newPerms {
		q := `SELECT tenant_id FROM app_installations WHERE app_id = $1::varchar(128) AND status = 'enabled'`
		if moduleID == "core" {
			q = `SELECT id FROM tenants WHERE $1::varchar(128) = 'core'`
		}
		rows, err := admin.Query(ctx, q, moduleID)
		if err != nil {
			return fmt.Errorf("backfill tenants %s: %w", moduleID, err)
		}
		tenants, err := pgx.CollectRows(rows, pgx.RowTo[string])
		if err != nil {
			return fmt.Errorf("backfill tenants %s: %w", moduleID, err)
		}
		for _, tid := range tenants {
			tx, err := admin.Begin(ctx)
			if err != nil {
				return err
			}
			if err := rbac.GrantOnInstall(ctx, tx, tid, perms); err != nil {
				_ = tx.Rollback(ctx)
				return fmt.Errorf("backfill %s → %s: %w", moduleID, tid, err)
			}
			if err := tx.Commit(ctx); err != nil {
				return err
			}
		}
		if len(tenants) > 0 {
			log.Printf("permission backfill: %s — %d шинэ код, %d tenant", moduleID, len(perms), len(tenants))
		}
	}
	return nil
}

// Sync — бинари асахад нэг удаа: компиллогдсон модулиудын permission ба
// апп мөрүүдийг admin pool-оор (nexus_platform эрхтэй) каталогт тулгана.
func Sync(ctx context.Context, admin *pgxpool.Pool, catalogPath string) error {
	// Модуль шинэчлэгдэж ШИНЭ permission нэмэгдвэл GrantOnInstall дахин
	// ажилладаггүй тул энд: анх удаа орж буй кодыг (xmax = 0) тэмдэглээд,
	// тухайн модулийг суулгасан tenant бүрт зөвхөн тэр кодуудын default
	// оноолтыг хийнэ. Байгаа кодод хүрэхгүй — tenant-ийн гараар хассан
	// оноолт сэргэхгүй.
	newPerms := map[string][]nexus.PermissionDefinition{} // moduleID → шинэ

	// Платформын core permission-ууд.
	for _, p := range rbac.CorePermissions() {
		dr, _ := json.Marshal(p.DefaultRoles)
		var inserted bool
		if err := admin.QueryRow(ctx, `
			INSERT INTO permissions (code, module_id, name, description, own_scope, default_roles)
			VALUES ($1::varchar(128), 'core', $2::varchar(160), $3::varchar(500), $4::boolean, $5::jsonb)
			ON CONFLICT (code) DO UPDATE SET
			  name = excluded.name, description = excluded.description,
			  own_scope = excluded.own_scope, default_roles = excluded.default_roles
			RETURNING (xmax = 0)`,
			p.Code, p.Name, p.Description, p.OwnScope, dr).Scan(&inserted); err != nil {
			return fmt.Errorf("core permission sync %s: %w", p.Code, err)
		}
		if inserted {
			newPerms["core"] = append(newPerms["core"], p)
		}
	}

	compiled := map[string]bool{}
	for _, m := range nexus.Registered() {
		compiled[m.ID()] = true
		_, err := admin.Exec(ctx, `
			INSERT INTO apps (id, short_id, name, version, compiled, updated_at)
			VALUES ($1::varchar(128), $2::varchar(32), $3::varchar(160), $4::varchar(32), true, now())
			ON CONFLICT (id) DO UPDATE SET
			  short_id = excluded.short_id, name = excluded.name,
			  version = excluded.version, compiled = true, updated_at = now()`,
			m.ID(), m.ShortID(), m.Name(), m.Version())
		if err != nil {
			return fmt.Errorf("app sync %s: %w", m.ID(), err)
		}
		for _, p := range m.Permissions() {
			dr, _ := json.Marshal(p.DefaultRoles)
			var inserted bool
			err := admin.QueryRow(ctx, `
				INSERT INTO permissions (code, module_id, name, description, own_scope, default_roles)
				VALUES ($1::varchar(128), $2::varchar(128), $3::varchar(160), $4::varchar(500), $5::boolean, $6::jsonb)
				ON CONFLICT (code) DO UPDATE SET
				  module_id = excluded.module_id, name = excluded.name,
				  description = excluded.description, own_scope = excluded.own_scope,
				  default_roles = excluded.default_roles
				RETURNING (xmax = 0)`,
				p.Code, m.ID(), p.Name, p.Description, p.OwnScope, dr).Scan(&inserted)
			if err != nil {
				return fmt.Errorf("permission sync %s: %w", p.Code, err)
			}
			if inserted {
				newPerms[m.ID()] = append(newPerms[m.ID()], p)
			}
		}
	}
	if err := backfillGrants(ctx, admin, newPerms); err != nil {
		return err
	}

	// Каталог файл: компиллогдоогүй аппууд store-д "татаж авах боломжтой"
	// гэж харагдана. Файл байхгүй бол алгасна.
	raw, err := os.ReadFile(catalogPath)
	if err != nil {
		if os.IsNotExist(err) {
			return markUncompiled(ctx, admin, compiled)
		}
		return err
	}
	var cat catalogFile
	if err := json.Unmarshal(raw, &cat); err != nil {
		return fmt.Errorf("каталог %s гэмтэлтэй: %w", catalogPath, err)
	}
	for _, e := range cat.Apps {
		if compiled[e.ID] {
			// Компиллогдсон хувилбар нь үнэн эх — каталог зөвхөн
			// тайлбар/publisher-ийг нөхнө.
			_, err = admin.Exec(ctx, `
				UPDATE apps SET description = $2::varchar(1000), publisher = $3::varchar(120),
				                go_module = $4::varchar(255)
				 WHERE id = $1::varchar(128)`,
				e.ID, e.Description, e.Publisher, e.GoModule)
		} else {
			_, err = admin.Exec(ctx, `
				INSERT INTO apps (id, short_id, name, version, description, publisher, go_module, compiled, updated_at)
				VALUES ($1::varchar(128), $2::varchar(32), $3::varchar(160), $4::varchar(32),
				        $5::varchar(1000), $6::varchar(120), $7::varchar(255), false, now())
				ON CONFLICT (id) DO UPDATE SET
				  name = excluded.name, version = excluded.version,
				  description = excluded.description, publisher = excluded.publisher,
				  go_module = excluded.go_module, compiled = false, updated_at = now()`,
				e.ID, e.ShortID, e.Name, e.Version, e.Description, e.Publisher, e.GoModule)
		}
		if err != nil {
			return fmt.Errorf("каталог sync %s: %w", e.ID, err)
		}
	}
	return markUncompiled(ctx, admin, compiled)
}

// markUncompiled — өмнө нь компиллогдсон гэж тэмдэглэгдсэн ч энэ бинарид
// байхгүй болсон аппуудын тугийг унтраана.
func markUncompiled(ctx context.Context, admin *pgxpool.Pool, compiled map[string]bool) error {
	rows, err := admin.Query(ctx, `SELECT id FROM apps WHERE compiled`)
	if err != nil {
		return err
	}
	defer rows.Close()
	var stale []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		if !compiled[id] {
			stale = append(stale, id)
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range stale {
		if _, err := admin.Exec(ctx,
			`UPDATE apps SET compiled = false, updated_at = now() WHERE id = $1::varchar(128)`, id); err != nil {
			return err
		}
	}
	return nil
}

// ─── Суулгалт ───────────────────────────────────────────────────────────

var (
	ErrNotCompiled = errors.New("app not compiled into this binary")
	ErrNotFound    = errors.New("app not found")
)

type Installer struct {
	db    nexus.DB
	perms *rbac.Store
}

func NewInstaller(db nexus.DB, perms *rbac.Store) *Installer {
	return &Installer{db: db, perms: perms}
}

// Install — аппыг tenant-д суулгана: хамаарлуудыг нь рекурсивээр эхэлж
// суулгаад, RBAC default оноолтыг хийнэ. Бүгд нэг гүйлгээнд.
func (i *Installer) Install(ctx context.Context, appID string) error {
	mods := map[string]nexus.Module{}
	for _, m := range nexus.Registered() {
		mods[m.ID()] = m
	}
	target, ok := mods[appID]
	if !ok {
		// Каталогт байгаа ч компиллогдоогүй байж болно — ялгаж мэдээлнэ.
		var exists bool
		if err := i.db.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM apps WHERE id = $1::varchar(128))`, appID).Scan(&exists); err != nil {
			return err
		}
		if exists {
			return ErrNotCompiled
		}
		return ErrNotFound
	}

	order, err := ResolveOrder(mods, target)
	if err != nil {
		return err
	}

	tenantID := nexus.TenantID(ctx)
	err = i.db.Tx(ctx, func(tx pgx.Tx) error {
		for _, m := range order {
			_, err := tx.Exec(ctx, `
				INSERT INTO app_installations (tenant_id, app_id, version, status)
				VALUES ($1::uuid, $2::varchar(128), $3::varchar(32), 'enabled')
				ON CONFLICT (tenant_id, app_id)
				DO UPDATE SET status = 'enabled', version = excluded.version`,
				tenantID, m.ID(), m.Version())
			if err != nil {
				return fmt.Errorf("install %s: %w", m.ID(), err)
			}
			if err := rbac.GrantOnInstall(ctx, tx, tenantID, m.Permissions()); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	i.perms.Invalidate(tenantID)
	return nil
}

// ResolveOrder — суулгах дарааллыг DFS-ээр гаргана: хамаарлууд эхэлж,
// мөчлөг/дутуу хамаарал илэрвэл алдаа. Цэвэр функц — unit тесттэй.
func ResolveOrder(mods map[string]nexus.Module, target nexus.Module) ([]nexus.Module, error) {
	var order []nexus.Module
	state := map[string]int{} // 0 үзээгүй, 1 явж байна, 2 дууссан
	var visit func(m nexus.Module) error
	visit = func(m nexus.Module) error {
		switch state[m.ID()] {
		case 1:
			return fmt.Errorf("хамаарлын мөчлөг: %s", m.ID())
		case 2:
			return nil
		}
		state[m.ID()] = 1
		for _, d := range m.Dependencies() {
			dm, ok := mods[d.ID]
			if !ok {
				return fmt.Errorf("%s-ийн хамаарал %s энэ бинарид алга", m.ID(), d.ID)
			}
			if err := visit(dm); err != nil {
				return err
			}
		}
		state[m.ID()] = 2
		order = append(order, m)
		return nil
	}
	if err := visit(target); err != nil {
		return nil, err
	}
	return order, nil
}

// SetStatus — enable/disable.
func (i *Installer) SetStatus(ctx context.Context, appID, status string) error {
	tag, err := i.db.Exec(ctx, `
		UPDATE app_installations SET status = $3::varchar(16)
		 WHERE tenant_id = $1::uuid AND app_id = $2::varchar(128)`,
		nexus.TenantID(ctx), appID, status)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Gate: "энэ tenant-д апп идэвхтэй юу" ───────────────────────────────

type Gate struct {
	db    nexus.DB
	mu    sync.Mutex
	cache map[string]gateEntry
}

type gateEntry struct {
	enabled bool
	at      time.Time
}

const gateTTL = 15 * time.Second

func NewGate(db nexus.DB) *Gate { return &Gate{db: db, cache: map[string]gateEntry{}} }

func (g *Gate) enabled(ctx context.Context, tenantID, appID string) (bool, error) {
	key := tenantID + ":" + appID
	g.mu.Lock()
	if e, ok := g.cache[key]; ok && time.Since(e.at) < gateTTL {
		g.mu.Unlock()
		return e.enabled, nil
	}
	g.mu.Unlock()
	var ok bool
	err := g.db.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM app_installations
		 WHERE tenant_id = $1::uuid AND app_id = $2::varchar(128) AND status = 'enabled')`,
		tenantID, appID).Scan(&ok)
	if err != nil {
		return false, err
	}
	g.mu.Lock()
	if len(g.cache) >= 10000 {
		g.cache = make(map[string]gateEntry)
	}
	g.cache[key] = gateEntry{enabled: ok, at: time.Now()}
	g.mu.Unlock()
	return ok, nil
}

// Invalidate — суулгалтын төлөв өөрчлөгдөхөд tenant-ийн gate кэшийг унагаана.
func (g *Gate) Invalidate(tenantID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for k := range g.cache {
		if strings.HasPrefix(k, tenantID+":") {
			delete(g.cache, k)
		}
	}
}

// Middleware — суулгаагүй/унтраасан аппын route-уудад 403.
func (g *Gate) Middleware(appID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ok, err := g.enabled(r.Context(), nexus.TenantID(r.Context()), appID)
			if err != nil {
				http.Error(w, `{"error":"installation check failed"}`, http.StatusInternalServerError)
				return
			}
			if !ok {
				http.Error(w, `{"error":"app not installed"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
