package core

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/appstore"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/audit"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/auth"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/bus"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/config"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/db"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/handlers"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/rbac"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/tenantstate"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
)

// cmdServe — make serve
func cmdServe(_ []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	pools, err := db.Connect(ctx, cfg.DatabaseURL, cfg.DatabaseURLAdmin, cfg.DatabaseURLAuth)
	cancel()
	if err != nil {
		return err
	}
	defer pools.Close()

	// Boot sync: permission ба апп каталог.
	syncCtx, syncCancel := context.WithTimeout(context.Background(), 30*time.Second)
	if err := appstore.Sync(syncCtx, pools.Admin, appstore.Source{URL: cfg.RegistryURL, Keys: cfg.RegistryKeys, CacheDir: cfg.RegistryCacheDir, FilePath: cfg.CatalogPath}); err != nil {
		syncCancel()
		return err
	}
	syncCancel()

	// Админгүй instance ажиллах л болно, гэхдээ сануулна.
	var hasAdmin bool
	_ = pools.Admin.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM users WHERE platform_admin)`).Scan(&hasAdmin)
	if !hasAdmin {
		log.Println("АНХААР: платформын админ алга — env-д ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD өгөөд `make migrate` ажиллуул")
	}

	tdb := db.NewTenantDB(pools.App)
	authSvc := auth.NewService(pools.Auth, cfg.CookieSecure)
	perms := rbac.NewStore(tdb)
	rec := audit.NewRecorder(tdb)
	installer := appstore.NewInstaller(tdb, perms)
	gate := appstore.NewGate(tdb)
	deps := nexus.Deps{DB: tdb, Perms: perms, Audit: rec}

	tstate := tenantstate.New(pools.Auth)
	authSvc.SetTenantState(func(ctx context.Context, tid string) (bool, bool, error) {
		st, err := tstate.Get(ctx, tid)
		return st.Suspended, st.ReadOnly, err
	})
	authH := &handlers.Auth{Pool: pools.Auth, DB: tdb, Svc: authSvc, Audit: rec, Perms: perms, State: tstate}
	storeH := &handlers.Store{DB: tdb, Installer: installer, Gate: gate, Audit: rec}
	rbacH := &handlers.RBACH{DB: tdb, Pool: pools.Auth, Perms: perms, Audit: rec}
	miscH := &handlers.Misc{DB: tdb, Perms: perms}
	adminH := &handlers.Admin{Pool: pools.Admin, Svc: authSvc, Rec: rec, State: tstate, PortalURL: cfg.PortalURL}

	// Процесс хоорондын cache invalidation — Postgres LISTEN/NOTIFY.
	b := bus.New(pools.App)
	perms.SetNotifier(func(t string) { b.Publish("grants:" + t) })
	gate.SetNotifier(func(t string) { b.Publish("gate:" + t) })
	tstate.SetNotifier(func(t string) { b.Publish("tenant:" + t) })
	b.Subscribe(func(payload string) {
		kind, tid, ok := strings.Cut(payload, ":")
		if !ok {
			return
		}
		switch kind {
		case "grants":
			perms.InvalidateLocal(tid)
		case "gate":
			gate.InvalidateLocal(tid)
		case "tenant":
			tstate.InvalidateLocal(tid)
		}
	})
	busCtx, busCancel := context.WithCancel(context.Background())
	defer busCancel()
	go b.Run(busCtx)

	r := chi.NewRouter()
	r.Use(clientIP, middleware.Logger, middleware.Recoverer, securityHeaders(cfg.Env == "production"))
	r.Use(middleware.Timeout(30 * time.Second))
	// CSRF (#5): SameSite=Lax нь same-site (*.домэйн) хоорондын хүсэлтээс
	// хамгаалдаггүй — бичих хүсэлт бүрд Origin-ийг хостой тулгана.
	r.Use(sameOriginOnly)

	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	})

	// Нээлттэй. Rate limit (#1): нууц үгийн endpoint-ууд argon2-ийн улмаас
	// үнэтэй тул IP тус бүрд хязгаарлана.
	authLimit := httprate.LimitByIP(10, time.Minute)
	r.With(httprate.LimitByIP(5, time.Minute)).Post("/api/signup", authH.Signup)
	r.With(authLimit).Post("/api/login", authH.Login)
	r.With(authLimit).Post("/api/auth/handover", authH.Handover)
	r.With(httprate.LimitByIP(30, time.Minute)).Post("/api/logout", authH.Logout)
	r.Get("/api/catalog", storeH.Catalog)

	// Нэвтэрсэн (tenant сонгоогүй байж болно).
	r.Group(func(g chi.Router) {
		g.Use(authSvc.RequireUser)
		g.Get("/api/me", authH.Me)
		g.Put("/api/me", authH.UpdateProfile)
		g.With(httprate.LimitByIP(10, time.Minute)).Post("/api/me/password", authH.ChangePassword)
		g.Post("/api/session/tenant", authH.SelectTenant)
		g.Post("/api/tenants", authH.CreateTenant)
	})

	// Tenant сонгосон.
	r.Group(func(g chi.Router) {
		g.Use(authSvc.RequireTenant)
		g.Get("/api/menu", miscH.Menu)
		g.Get("/api/store/apps", storeH.List)
		g.Get("/api/store/apps/{id}/history", storeH.History)
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

		g.Get("/api/tenant/profile", authH.TenantProfile)
		g.With(nexus.RequirePermission(perms, "core.settings.manage")).Put("/api/tenant/profile", authH.UpdateTenantProfile)
		g.With(nexus.RequirePermission(perms, "core.members.manage")).Get("/api/members", rbacH.Members)
		g.With(nexus.RequirePermission(perms, "core.members.manage")).Get("/api/members/lookup", rbacH.LookupMember)
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
		g.Get("/api/admin/tenants/{id}/members", adminH.TenantMembers)
		g.Post("/api/admin/impersonate", adminH.Impersonate)
		g.Put("/api/admin/tenants/{id}/state", adminH.SetTenantState)
		g.Post("/api/admin/tenants/{id}/delete", adminH.ScheduleDeletion)
		g.Post("/api/admin/tenants/{id}/delete/cancel", adminH.CancelDeletion)
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

	// Default нь loopback — гадагш nginx л нээнэ; Docker-т LISTEN_ADDR=0.0.0.0.
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = "127.0.0.1"
	}
	srv := &http.Server{
		Addr:              addr + ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      35 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	// Хугацаа дууссан session-уудыг цагийн зайцаар цэвэрлэнэ.
	purgeStop := make(chan struct{})
	defer close(purgeStop)
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-purgeStop:
				return
			case <-t.C:
				var n int
				if err := pools.Auth.QueryRow(context.Background(),
					`SELECT auth_sessions_purge()`).Scan(&n); err != nil {
					log.Printf("session purge: %v", err)
				} else if n > 0 {
					log.Printf("session purge: %d устгав", n)
				}
				if n, err := handlers.SweepDeletions(context.Background(), pools.Admin); err != nil {
					log.Printf("tenant deletion sweep: %v", err)
				} else if n > 0 {
					log.Printf("tenant deletion sweep: %d байгууллага устгав", n)
				}
			}
		}
	}()

	// Graceful shutdown: SIGTERM/SIGINT дээр идэвхтэй хүсэлтүүдээ дуусгаад
	// унтарна — ингэснээр defer pools.Close() ч бодитоор ажиллана.
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	log.Printf("nexus-mini API %s (%d модуль)", srv.Addr, len(nexus.Registered()))

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-stop:
		shCtx, shCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shCancel()
		return srv.Shutdown(shCtx)
	}
}

// sameOriginOnly — бичих хүсэлтийн Origin (байвал) хүсэлтийн хосттой
// таарахгүй бол 403. Origin-гүй клиент (curl, server-to-server) хэвээр.
// Next-ийн rewrite-ээр дамжсан хүсэлтийн Host нь destination болж
// солигддог (Docker compose: api:8084) тул X-Forwarded-Host-ийг мөн
// хүлээн зөвшөөрнө — түүнийг эцсийн proxy (Next/nginx) тавьдаг, гаднаас
// шууд ирсэн хүсэлтэд nginx хүчээр дарж бичдэг тул хуурч болохгүй.
// securityHeaders — API-ийн хариу бүрт (nginx-гүй дистрибуц/Docker-т ч).
// JSON API тул CSP 'none'; HTML-ийг Next өөрөө өгдөг.
func securityHeaders(prod bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			h.Set("Cache-Control", "no-store")
			if prod {
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// clientIP — rate limit/лог-ийн IP. chi-ийн RealIP нь True-Client-IP,
// X-Forwarded-For зэрэг клиентийн тавьж болох толгойг итгэдэг (GHSA-3fxj-6jh8-hvhx)
// тул хэрэглэхгүй. Зөвхөн шууд холбогч нь loopback/private (манай nginx)
// үед л nginx-ийн өөрөө тавьдаг X-Real-IP-г авна; өөр тохиолдолд RemoteAddr.
func clientIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			if ip := net.ParseIP(host); ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
				if real := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); real != nil {
					r.RemoteAddr = net.JoinHostPort(real.String(), "0")
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func sameOriginOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handover нь админ панелийн (өөр домэйн) form POST — token өөрөө
		// нэг удаагийн CSRF-ийн хамгаалалт.
		if r.Method != http.MethodGet && r.Method != http.MethodHead && r.URL.Path != "/api/auth/handover" {
			// Sec-Fetch-Site: браузер өөрөө тавьдаг (клиент скрипт өөрчилж
			// чадахгүй) — cross-site гэж хэлвэл Origin-оос үл хамааран 403.
			if sfs := r.Header.Get("Sec-Fetch-Site"); sfs == "cross-site" {
				http.Error(w, `{"error":"cross-site request"}`, http.StatusForbidden)
				return
			}
			if origin := r.Header.Get("Origin"); origin != "" {
				ok := origin == "https://"+r.Host || origin == "http://"+r.Host
				if fh := r.Header.Get("X-Forwarded-Host"); !ok && fh != "" {
					ok = origin == "https://"+fh || origin == "http://"+fh
				}
				if !ok {
					http.Error(w, `{"error":"origin mismatch"}`, http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
