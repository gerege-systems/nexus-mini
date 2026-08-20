package nexus

import (
	"context"
	"errors"
	"net/http"
)

// Scope — permission-ий хамрах хүрээ.
type ScopeKind string

const (
	// ScopeAll — tenant-ийн бүх мөр.
	ScopeAll ScopeKind = "all"
	// ScopeOwn — зөвхөн created_by нь өөрөө байх мөрүүд.
	ScopeOwn ScopeKind = "own"
)

// Grant — нэг permission-ий шийдэгдсэн эрх: олгогдсон эсэх + scope.
// Нэг хэрэглэгч олон role-оос ижил permission авбал өргөн нь (all) ялна.
type Grant struct {
	Allowed bool
	Scope   ScopeKind
}

// PermissionStore нь "энэ хэрэглэгч энэ tenant дотор юу хийж болох вэ"
// гэдэгт хариулна. Платформ хэрэгжүүлж модульд Deps-ээр өгнө.
type PermissionStore interface {
	// UserGrants нь permission код → Grant map буцаана.
	UserGrants(ctx context.Context, tenantID, userID string) (map[string]Grant, error)
}

// ErrForbidden — permission шалгалтын татгалзал.
var ErrForbidden = errors.New("permission denied")

type ctxKey int

const (
	ctxTenantID ctxKey = iota
	ctxUserID
	ctxScope
)

// WithIdentity нь платформын auth middleware-ийн context-д таних мэдээллээ
// хийдэг цэг. Модуль үүнийг дуудахгүй — зөвхөн уншина.
func WithIdentity(ctx context.Context, tenantID, userID string) context.Context {
	ctx = context.WithValue(ctx, ctxTenantID, tenantID)
	return context.WithValue(ctx, ctxUserID, userID)
}

// TenantID нь хүсэлтийн tenant. Платформын gate-ийн цаана үргэлж байна.
func TenantID(ctx context.Context) string {
	s, _ := ctx.Value(ctxTenantID).(string)
	return s
}

// UserID нь хүсэлт гаргагч хэрэглэгч.
func UserID(ctx context.Context) string {
	s, _ := ctx.Value(ctxUserID).(string)
	return s
}

// Scope нь RequirePermission-ий шийдсэн scope. RequirePermission-гүй route
// дээр ScopeAll буцаана (тэнд permission огт шалгаагүй тул модуль өөрөө
// мэднэ).
func Scope(ctx context.Context) ScopeKind {
	if s, ok := ctx.Value(ctxScope).(ScopeKind); ok {
		return s
	}
	return ScopeAll
}

// RequirePermission — тухайн permission-ийг шаарддаг middleware. Олгогдсон
// grant-ийн scope-ийг context-д хийдэг тул handler нь Scope(ctx)-оор уншиж
// ScopeOwn үед created_by шүүлтээ нэмнэ.
func RequirePermission(store PermissionStore, code string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			grants, err := store.UserGrants(ctx, TenantID(ctx), UserID(ctx))
			if err != nil {
				http.Error(w, `{"error":"permission lookup failed"}`, http.StatusInternalServerError)
				return
			}
			g, ok := grants[code]
			if !ok || !g.Allowed {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(ctx, ctxScope, g.Scope)))
		})
	}
}
