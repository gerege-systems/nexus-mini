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
	"github.com/gerege-systems/nexus-mini/backend/pkg/registry"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// CatalogEntry — каталог файлын (эсвэл registry-ийн, үе 2) нэг апп.
// Source — каталогийн эх: registry URL (гарын үсэгтэй) + локал файл fallback.
type Source struct {
	URL      string // "off" = зөвхөн файл
	Keys     string // base64, таслал
	CacheDir string
	FilePath string
}

// loadIndex — registry-ээс (амжилтгүй бол кэш), тэр ч байхгүй бол локал файл.
// Boot-ийг хэзээ ч унагаахгүй — алдааг логлоод хоосон index-тэй үргэлжилнэ.
func loadIndex(ctx context.Context, src Source) *registry.Index {
	if src.URL != "" && src.URL != "off" {
		keys, err := registry.ParseKeys(src.Keys)
		if err != nil {
			log.Printf("registry: %v", err)
		} else {
			ix, err := registry.Fetch(ctx, src.URL, keys, src.CacheDir)
			if err != nil {
				log.Printf("registry: %v", err)
			}
			if ix != nil {
				return ix
			}
		}
	}
	if src.FilePath != "" {
		ix, err := registry.LoadFile(src.FilePath)
		if err == nil {
			return ix
		}
		if !os.IsNotExist(err) {
			log.Printf("каталог %s: %v", src.FilePath, err)
		}
	}
	return &registry.Index{}
}

// recordRelease — нийтлэгчийн хувилбарыг түүхэнд (анх харагдсан цаг).
func recordRelease(ctx context.Context, admin *pgxpool.Pool, appID, version string) error {
	if appID == "" {
		return nil
	}
	_, err := admin.Exec(ctx, `
		INSERT INTO app_releases (app_id, version) VALUES ($1::varchar(128), $2::varchar(32))
		ON CONFLICT DO NOTHING`, appID, version)
	return err
}

// syncPerm — permission-ийг каталогт тулгаад, анх удаа орж байвал
// (INSERT зам: xmax = 0) тухайн модулийг суулгасан (core бол бүх) tenant-д
// default оноолтыг хийнэ. Бүгд нэг tx — хагас төлөв үлдэхгүй. Байгаа кодод
// хүрэхгүй тул tenant-ийн гараар хассан оноолт сэргэхгүй. Admin pool
// (nexus_platform) RLS-ийг давдаг.
func syncPerm(ctx context.Context, admin *pgxpool.Pool, moduleID string, p nexus.PermissionDefinition) (err error) {
	tx, err := admin.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()
	dr, _ := json.Marshal(p.DefaultRoles)
	var inserted bool
	if err = tx.QueryRow(ctx, `
		INSERT INTO permissions (code, module_id, name, description, own_scope, default_roles)
		VALUES ($1::varchar(128), $2::varchar(128), $3::varchar(160), $4::varchar(500), $5::boolean, $6::jsonb)
		ON CONFLICT (code) DO UPDATE SET
		  module_id = excluded.module_id, name = excluded.name,
		  description = excluded.description, own_scope = excluded.own_scope,
		  default_roles = excluded.default_roles
		RETURNING (xmax = 0)`,
		p.Code, moduleID, p.Name, p.Description, p.OwnScope, dr).Scan(&inserted); err != nil {
		return fmt.Errorf("permission sync %s: %w", p.Code, err)
	}
	if inserted {
		q := `SELECT tenant_id FROM app_installations WHERE app_id = $1::varchar(128) AND status = 'enabled'`
		if moduleID == "core" {
			q = `SELECT id FROM tenants WHERE $1::varchar(128) = 'core'`
		}
		rows, qerr := tx.Query(ctx, q, moduleID)
		if qerr != nil {
			return fmt.Errorf("backfill tenants %s: %w", p.Code, qerr)
		}
		tenants, cerr := pgx.CollectRows(rows, pgx.RowTo[string])
		if cerr != nil {
			return fmt.Errorf("backfill tenants %s: %w", p.Code, cerr)
		}
		for _, tid := range tenants {
			if err = rbac.GrantOnInstall(ctx, tx, tid, []nexus.PermissionDefinition{p}); err != nil {
				return fmt.Errorf("backfill %s → %s: %w", p.Code, tid, err)
			}
		}
		if len(tenants) > 0 {
			log.Printf("permission backfill: %s → %d tenant", p.Code, len(tenants))
		}
	}
	return tx.Commit(ctx)
}

// Sync — бинари асахад нэг удаа: компиллогдсон модулиудын permission ба
// апп мөрүүдийг admin pool-оор (nexus_platform эрхтэй) каталогт тулгана.
func Sync(ctx context.Context, admin *pgxpool.Pool, src Source) error {
	// Платформын core permission-ууд. Модуль шинэчлэгдэж ШИНЭ permission
	// нэмэгдвэл GrantOnInstall дахин ажилладаггүй тул syncPerm анх удаа орж
	// буй кодод (xmax = 0) суулгасан tenant бүрт default оноолтыг мөн tx-д
	// хийнэ — алдвал бүхэлдээ буцаж, дараагийн асалтад дахин оролдоно.
	for _, p := range rbac.CorePermissions() {
		if err := syncPerm(ctx, admin, "core", p); err != nil {
			return err
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
			if err := syncPerm(ctx, admin, m.ID(), p); err != nil {
				return err
			}
		}
		if err := recordRelease(ctx, admin, m.ID(), m.Version()); err != nil {
			return err
		}
		// Суулгасан tenant-уудын хувилбарыг компиллогдсон руу нь өргөнө
		// (түүхэнд "upgrade" үйл явдал, actor = систем). Шинэ permission-ийг
		// syncPerm аль хэдийн backfill хийсэн.
		rows, err := admin.Query(ctx, `
			WITH old AS (
			  SELECT id, tenant_id, version FROM app_installations
			   WHERE app_id = $1::varchar(128) AND version <> $2::varchar(32))
			UPDATE app_installations i SET version = $2::varchar(32)
			  FROM old WHERE i.id = old.id
			RETURNING old.tenant_id, old.version`,
			m.ID(), m.Version())
		if err != nil {
			return fmt.Errorf("installation upgrade %s: %w", m.ID(), err)
		}
		type up struct{ tenant, from string }
		ups, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (up, error) {
			var u up
			err := r.Scan(&u.tenant, &u.from)
			return u, err
		})
		if err != nil {
			return fmt.Errorf("installation upgrade %s: %w", m.ID(), err)
		}
		for _, u := range ups {
			if _, err := admin.Exec(ctx, `
				INSERT INTO installation_events (tenant_id, app_id, action, from_version, to_version)
				VALUES ($1::uuid, $2::varchar(128), 'upgrade', $3::varchar(32), $4::varchar(32))`,
				u.tenant, m.ID(), u.from, m.Version()); err != nil {
				return err
			}
		}
		if len(ups) > 0 {
			log.Printf("app upgrade: %s → %s, %d tenant", m.ID(), m.Version(), len(ups))
		}
	}

	// Registry/локал index: компиллогдоогүй аппууд store-д "татаж авах
	// боломжтой" гэж харагдана; компиллогдсоных нь тайлбар/publisher нөхөгдөнө.
	ix := loadIndex(ctx, src)
	for _, e := range ix.Apps {
		var err error
		if err := recordRelease(ctx, admin, e.ID, e.Version); err != nil {
			return err
		}
		if compiled[e.ID] {
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
	// Компиллогдсон модулийн тайлбарыг кодоос (Describer) — registry-гүй ч.
	for _, m := range nexus.Registered() {
		mf := registry.FromModule(m)
		if mf.Description != "" || mf.Publisher != "" {
			if _, err := admin.Exec(ctx, `
				UPDATE apps SET description = $2::varchar(1000), publisher = $3::varchar(120), go_module = $4::varchar(255)
				 WHERE id = $1::varchar(128)`, mf.ID, mf.Description, mf.Publisher, mf.GoModule); err != nil {
				return fmt.Errorf("describer sync %s: %w", mf.ID, err)
			}
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
			if _, err := tx.Exec(ctx, `
				INSERT INTO installation_events (tenant_id, app_id, action, to_version, user_id)
				VALUES ($1::uuid, $2::varchar(128), 'install', $3::varchar(32), $4::uuid)`,
				tenantID, m.ID(), m.Version(), nexus.UserID(ctx)); err != nil {
				return fmt.Errorf("install event %s: %w", m.ID(), err)
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
	action := "enable"
	if status == "disabled" {
		action = "disable"
	}
	_, err = i.db.Exec(ctx, `
		INSERT INTO installation_events (tenant_id, app_id, action, user_id)
		VALUES ($1::uuid, $2::varchar(128), $3::varchar(16), $4::uuid)`,
		nexus.TenantID(ctx), appID, action, nexus.UserID(ctx))
	return err
}

// ─── Gate: "энэ tenant-д апп идэвхтэй юу" ───────────────────────────────

type Gate struct {
	db     nexus.DB
	mu     sync.Mutex
	cache  map[string]gateEntry
	notify func(tenantID string)
}

// SetNotifier — Invalidate бүрд дуудагдах cross-process мэдэгдэгч.
func (g *Gate) SetNotifier(fn func(tenantID string)) { g.notify = fn }

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
	g.InvalidateLocal(tenantID)
	if g.notify != nil {
		g.notify(tenantID)
	}
}

// InvalidateLocal — зөвхөн энэ процессын кэш (bus-аас ирсэн мэдэгдэлд).
func (g *Gate) InvalidateLocal(tenantID string) {
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
