// cmd/api — nexus-mini HTTP API.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/handlers"
	"github.com/gerege-systems/nexus-mini/backend/internal/modules"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/appstore"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/config"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/platform/rbac"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	modules.RegisterAll()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	pools, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DatabaseURLAdmin)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer pools.Close()

	// Boot sync: permission ба апп каталог.
	syncCtx, syncCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := appstore.Sync(syncCtx, pools.Admin, cfg.CatalogPath); err != nil {
		log.Fatalf("каталог sync: %v", err)
	}
	syncCancel()

	tdb := db.NewTenantDB(pools.App)
	authSvc := auth.NewService(pools.App, cfg.CookieSecure)
	perms := rbac.NewStore(tdb)
	rec := audit.NewRecorder(pools.App)
	installer := appstore.NewInstaller(tdb, perms)
	gate := appstore.NewGate(tdb)
	deps := nexus.Deps{DB: tdb, Perms: perms, Audit: rec}

	authH := &handlers.Auth{Pool: pools.App, DB: tdb, Svc: authSvc, Audit: rec, Perms: perms}
	storeH := &handlers.Store{DB: tdb, Installer: installer, Gate: gate, Audit: rec}
	rbacH := &handlers.RBACH{DB: tdb, Pool: pools.App, Perms: perms, Audit: rec}
	miscH := &handlers.Misc{DB: tdb, Perms: perms}
	adminH := &handlers.Admin{Pool: pools.Admin}

	r := chi.NewRouter()
	r.Use(middleware.RealIP, middleware.Logger, middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	// Нээлттэй.
	r.Get("/api/setup", authH.SetupState)
	r.Post("/api/setup", authH.SetupCreate)
	r.Post("/api/signup", authH.Signup)
	r.Post("/api/login", authH.Login)
	r.Post("/api/logout", authH.Logout)

	// Нэвтэрсэн (tenant сонгоогүй байж болно).
	r.Group(func(g chi.Router) {
		g.Use(authSvc.RequireUser)
		g.Get("/api/me", authH.Me)
		g.Post("/api/session/tenant", authH.SelectTenant)
		g.Post("/api/tenants", authH.CreateTenant)
	})

	// Tenant сонгосон.
	r.Group(func(g chi.Router) {
		g.Use(authSvc.RequireTenant)
		g.Get("/api/menu", miscH.Menu)
		g.Get("/api/store/apps", storeH.List)
		g.Get("/api/permissions", rbacH.Permissions)

		g.With(nexus.RequirePermission(perms, "core.apps.manage")).
			Post("/api/store/apps/{id}/install", storeH.Install)
		g.With(nexus.RequirePermission(perms, "core.apps.manage")).
			Post("/api/store/apps/{id}/enable", storeH.SetStatus("enabled"))
		g.With(nexus.RequirePermission(perms, "core.apps.manage")).
			Post("/api/store/apps/{id}/disable", storeH.SetStatus("disabled"))

		g.With(nexus.RequirePermission(perms, "core.roles.manage")).Get("/api/roles", rbacH.Roles)
		g.With(nexus.RequirePermission(perms, "core.roles.manage")).Post("/api/roles", rbacH.CreateRole)
		g.With(nexus.RequirePermission(perms, "core.roles.manage")).Put("/api/roles/{id}/grants", rbacH.SetGrants)

		g.With(nexus.RequirePermission(perms, "core.members.manage")).Get("/api/members", rbacH.Members)
		g.With(nexus.RequirePermission(perms, "core.members.manage")).Post("/api/members", rbacH.AddMember)
		g.With(nexus.RequirePermission(perms, "core.members.manage")).Put("/api/members/{id}/roles", rbacH.SetMemberRoles)
		g.With(nexus.RequirePermission(perms, "core.members.manage")).Delete("/api/members/{id}", rbacH.RemoveMember)

		g.With(nexus.RequirePermission(perms, "core.audit.read")).Get("/api/audit", miscH.Audit)
		g.With(nexus.RequirePermission(perms, "core.audit.read")).Get("/api/audit/verify", miscH.AuditVerify)
	})

	// Платформын админ панель (admin pool → бүх tenant харагдана).
	r.Group(func(g chi.Router) {
		g.Use(authSvc.RequirePlatformAdmin)
		g.Get("/api/admin/overview", adminH.Overview)
		g.Get("/api/admin/tenants", adminH.Tenants)
		g.Get("/api/admin/users", adminH.Users)
		g.Get("/api/admin/apps", adminH.Apps)
		g.Get("/api/admin/audit", adminH.Audit)
	})

	// Модулиуд: урьдчилан хамгаалагдсан router (docs/02-rbac.md #5) —
	// tenant auth + "апп идэвхтэй" gate платформ өөрөө тавьдаг.
	for _, m := range nexus.Registered() {
		mod := m
		r.Route("/api/apps/"+mod.ShortID(), func(sub chi.Router) {
			sub.Use(authSvc.RequireTenant)
			sub.Use(gate.Middleware(mod.ID()))
			mod.RegisterRoutes(sub, deps)
		})
	}

	log.Printf("nexus-mini API :%s (%d модуль)", cfg.Port, len(nexus.Registered()))
	if err := http.ListenAndServe(":"+cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
