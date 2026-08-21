// Package identity — DB-ийн RLS context-ийг тэжээдэг ДОТООД identity.
//
// Өмнө нь TenantDB нь pkg/nexus-ийн WithIdentity-ээр тавьсан утгыг уншдаг
// байсан — тэр нь экспортлогдсон тул модуль хүсэлтийнхээ ctx-д өөр tenant
// тавиад RLS-ийг нээх онолын зам байлаа (аудитын шүүмж). Одоо DB зөвхөн
// энэ internal package-ийн утгыг уншина; pkg/nexus-ийн TenantID/UserID нь
// модулиудад зориулсан ЗӨВХӨН УНШИХ хувилбар хэвээр.
package identity

import "context"

type ctxKey int

const (
	keyTenant ctxKey = iota
	keyUser
	keyImpersonator
)

// WithImpersonator — платформын админ хэрэглэгчийн нэрийн өмнөөс орсон
// session (handover). Audit бүртгэл бүрд impersonated_by хавсарна.
func WithImpersonator(ctx context.Context, adminID string) context.Context {
	return context.WithValue(ctx, keyImpersonator, adminID)
}

func Impersonator(ctx context.Context) string {
	s, _ := ctx.Value(keyImpersonator).(string)
	return s
}

// With — auth middleware болон платформын дотоод урсгалууд л дуудна.
func With(ctx context.Context, tenantID, userID string) context.Context {
	ctx = context.WithValue(ctx, keyTenant, tenantID)
	return context.WithValue(ctx, keyUser, userID)
}

func TenantID(ctx context.Context) string {
	s, _ := ctx.Value(keyTenant).(string)
	return s
}

func UserID(ctx context.Context) string {
	s, _ := ctx.Value(keyUser).(string)
	return s
}
