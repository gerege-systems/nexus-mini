// Package auth — session-д суурилсан нэвтрэлт.
//
// Token нь 32 санамсаргүй байт (base64url) → cookie; DB-д зөвхөн sha256
// hex нь хадгалагдана. Session-ий бүх хайлт SECURITY DEFINER функцээр —
// tenant тодорхойлогдохоос өмнө ажилладаг тул (docs/01-lessons.md #1).
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/platform/identity"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	CookieName = "nexus_session"
	sessionTTL = 14 * 24 * time.Hour
)

// Principal — танигдсан хүсэлт гаргагч.
type Principal struct {
	SessionID     string
	UserID        string
	TenantID      string // хоосон = tenant сонгоогүй
	Name          string
	Email         string
	PlatformAdmin bool
}

type principalKey struct{}

func PrincipalFrom(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}

type Service struct {
	pool   *pgxpool.Pool // app pool — definer функцууд дуудагдана
	secure bool
}

func NewService(pool *pgxpool.Pool, secureCookies bool) *Service {
	return &Service{pool: pool, secure: secureCookies}
}

func newToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	return plain, hex.EncodeToString(sum[:]), nil
}

func hashToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

// StartSession нь session үүсгээд cookie тавьж, session id-гаа буцаана.
func (s *Service) StartSession(ctx context.Context, w http.ResponseWriter, userID string) (string, error) {
	plain, th, err := newToken()
	if err != nil {
		return "", err
	}
	var sid string
	err = s.pool.QueryRow(ctx,
		`SELECT auth_session_create($1::uuid, $2::char(64), $3::timestamptz)`,
		userID, th, time.Now().Add(sessionTTL)).Scan(&sid)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    plain,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionTTL.Seconds()),
	})
	return sid, nil
}

func (s *Service) EndSession(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	// Ижил нэртэй бүх cookie-гийн session-ийг устгана (Resolve-той ижил
	// логик — нэгийг нь л устгавал үлдсэнээрээ нэвтэрсэн хэвээр үлдэнэ).
	for _, c := range r.CookiesNamed(CookieName) {
		if c.Value != "" {
			_, _ = s.pool.Exec(ctx, `SELECT auth_session_delete($1::char(64))`, hashToken(c.Value))
		}
	}
	http.SetCookie(w, &http.Cookie{
		Name: CookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

// Resolve нь cookie-гоос Principal тодорхойлно; олдохгүй бол ok=false.
//
// Браузер ижил нэртэй ХЭД ХЭДЭН cookie зэрэг явуулж болдог (хуучирсан
// domain-cookie шинэ host-cookie-гийн урд жагсдаг тохиолдол бодитоор
// гарсан) тул эхнийхээр нь биш, бүгдээр нь оролдоно.
func (s *Service) Resolve(ctx context.Context, r *http.Request) (Principal, bool) {
	for _, c := range r.CookiesNamed(CookieName) {
		if c.Value == "" {
			continue
		}
		var p Principal
		var tenantID *string
		err := s.pool.QueryRow(ctx,
			`SELECT session_id, user_id, tenant_id, platform_admin, name, email
			   FROM auth_session_lookup($1::char(64))`, hashToken(c.Value)).
			Scan(&p.SessionID, &p.UserID, &tenantID, &p.PlatformAdmin, &p.Name, &p.Email)
		if err != nil {
			continue // хуучирсан/танигдахгүй token — дараагийнхыг үзнэ
		}
		if tenantID != nil {
			p.TenantID = *tenantID
		}
		return p, true
	}
	return Principal{}, false
}

// SetTenant нь session-ий идэвхтэй tenant-ийг сольж, гишүүнчлэлийг DB
// талд шалгана.
func (s *Service) SetTenant(ctx context.Context, sessionID, tenantID string) (bool, error) {
	var ok bool
	var arg any
	if tenantID != "" {
		arg = tenantID
	}
	err := s.pool.QueryRow(ctx,
		`SELECT auth_session_set_tenant($1::uuid, $2::uuid)`, sessionID, arg).Scan(&ok)
	return ok, err
}

// RequireUser — нэвтэрсэн байхыг шаардана (tenant сонгоогүй байж болно).
func (s *Service) RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, ok := s.Resolve(r.Context(), r)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), principalKey{}, p)
		// identity нь DB-ийн RLS context (дотоод), nexus нь модулиудын
		// зөвхөн-унших хувилбар.
		ctx = identity.With(ctx, p.TenantID, p.UserID)
		ctx = nexus.WithIdentity(ctx, p.TenantID, p.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequireTenant — нэвтэрсэн + идэвхтэй tenant сонгосон байхыг шаардана.
func (s *Service) RequireTenant(next http.Handler) http.Handler {
	return s.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		if p.TenantID == "" {
			http.Error(w, `{"error":"no tenant selected"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}

// RequirePlatformAdmin — платформын админ байхыг шаардана.
func (s *Service) RequirePlatformAdmin(next http.Handler) http.Handler {
	return s.RequireUser(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p, _ := PrincipalFrom(r.Context())
		if !p.PlatformAdmin {
			http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
