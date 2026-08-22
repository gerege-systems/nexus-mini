package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/internal/core/httpx"
	"github.com/gerege-systems/nexus-mini/backend/internal/core/ssoclient"
	"github.com/jackc/pgx/v5"
)

// SSO — гадны OIDC provider-оор нэвтрэх (Google, өөр nexus-mini, дурын issuer).
// Урсгал: GET /api/auth/sso/{key}/start → provider → GET /api/auth/sso/{key}/callback.
// PKCE verifier/state/nonce-ийг HMAC-тэй, 10 мин, HttpOnly cookie-д хадгална.
type SSO struct {
	Client     *ssoclient.Client
	Auth       *Auth
	PortalURL  string
	AutoSignup bool
	Secret     []byte // cookie HMAC (nexus_auth pool-ийн нууцаас гаралтай)
	secure     bool
}

func NewSSO(c *ssoclient.Client, a *Auth, portalURL string, autoSignup, secure bool, secret string) *SSO {
	sum := sha256.Sum256([]byte("sso-state:" + secret))
	return &SSO{Client: c, Auth: a, PortalURL: strings.TrimRight(portalURL, "/"), AutoSignup: autoSignup, Secret: sum[:], secure: secure}
}

type ssoState struct {
	Provider, State, Nonce, Verifier, Next string
	Exp                                    int64
}

const ssoCookie = "nexus_sso"

func (s *SSO) sign(b []byte) string {
	m := hmac.New(sha256.New, s.Secret)
	m.Write(b)
	return base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *SSO) verify(v string) (*ssoState, bool) {
	i := strings.LastIndex(v, ".")
	if i < 0 {
		return nil, false
	}
	b, err := base64.RawURLEncoding.DecodeString(v[:i])
	if err != nil {
		return nil, false
	}
	m := hmac.New(sha256.New, s.Secret)
	m.Write(b)
	want := base64.RawURLEncoding.EncodeToString(m.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(v[i+1:])) {
		return nil, false
	}
	var st ssoState
	if json.Unmarshal(b, &st) != nil || time.Now().Unix() > st.Exp {
		return nil, false
	}
	return &st, true
}

// GET /api/auth/sso/providers — нэвтрэх хуудасны товчнууд.
func (s *SSO) Providers(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"providers": s.Client.Public()})
}

func (s *SSO) redirectURI(key string) string {
	return s.PortalURL + "/api/auth/sso/" + key + "/callback"
}

// GET /api/auth/sso/{key}/start?next=/dashboard
func (s *SSO) Start(w http.ResponseWriter, r *http.Request) {
	key := providerKey(r)
	p, ok := s.Client.Get(key)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "sso provider олдсонгүй")
		return
	}
	verifier, challenge := ssoclient.NewPKCE()
	st := ssoState{Provider: key, State: ssoclient.RandString(), Nonce: ssoclient.RandString(), Verifier: verifier,
		Next: safeNext(r.URL.Query().Get("next")), Exp: time.Now().Add(10 * time.Minute).Unix()}
	u, err := s.Client.AuthURL(r.Context(), p, s.redirectURI(key), st.State, st.Nonce, challenge)
	if err != nil {
		log.Printf("sso start %s: %v", key, err)
		httpx.Error(w, http.StatusBadGateway, "sso provider хүрэхгүй байна")
		return
	}
	b, _ := json.Marshal(st)
	http.SetCookie(w, &http.Cookie{Name: ssoCookie, Value: s.sign(b), Path: "/api/auth/sso", HttpOnly: true,
		Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: 600})
	http.Redirect(w, r, u, http.StatusFound)
}

// GET /api/auth/sso/{key}/callback?code&state
func (s *SSO) Callback(w http.ResponseWriter, r *http.Request) {
	key := providerKey(r)
	p, ok := s.Client.Get(key)
	if !ok {
		httpx.Error(w, http.StatusNotFound, "sso provider олдсонгүй")
		return
	}
	clear := &http.Cookie{Name: ssoCookie, Value: "", Path: "/api/auth/sso", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteLaxMode, MaxAge: -1}
	defer http.SetCookie(w, clear)
	c, err := r.Cookie(ssoCookie)
	if err != nil {
		s.fail(w, r, "sso state алга (cookie)")
		return
	}
	st, ok := s.verify(c.Value)
	if !ok || st.Provider != key || st.State != r.URL.Query().Get("state") {
		s.fail(w, r, "sso state зөрүү")
		return
	}
	if e := r.URL.Query().Get("error"); e != "" {
		s.fail(w, r, "provider: "+e)
		return
	}
	id, err := s.Client.Exchange(r.Context(), p, s.redirectURI(key), r.URL.Query().Get("code"), st.Verifier, st.Nonce)
	if err != nil {
		log.Printf("sso callback %s: %v", key, err)
		s.fail(w, r, "sso баталгаажуулалт амжилтгүй")
		return
	}
	if !id.EmailVerified && key == "google" {
		s.fail(w, r, "имэйл баталгаажаагүй")
		return
	}
	// Бүртгэлтэй хэрэглэгч (имэйлээр) → session; үгүй бол JIT (тохиргоотой).
	var uid, hash, name string
	var isAdmin bool
	err = s.Auth.Pool.QueryRow(r.Context(),
		`SELECT id, password_hash, name, platform_admin FROM auth_user_by_email($1::varchar(255))`, id.Email).
		Scan(&uid, &hash, &name, &isAdmin)
	if errors.Is(err, pgx.ErrNoRows) {
		if !s.AutoSignup {
			s.fail(w, r, "энэ имэйл бүртгэлгүй — админаас урилга ав (SSO_AUTO_SIGNUP хаалттай)")
			return
		}
		n := strings.TrimSpace(id.Name)
		if n == "" {
			n = strings.SplitN(id.Email, "@", 2)[0]
		}
		if len(n) > 120 {
			n = n[:120]
		}
		// Нууц үггүй данс: argon2-гүй санамсаргүй hash — SSO-оор л нэвтэрнэ.
		if err := s.Auth.Pool.QueryRow(r.Context(),
			`SELECT auth_signup($1::varchar(255), $2::varchar(255), $3::varchar(120))`,
			id.Email, "sso:"+ssoclient.RandString(), n).Scan(&uid); err != nil {
			s.fail(w, r, "данс үүсгэж чадсангүй")
			return
		}
	} else if err != nil {
		s.fail(w, r, "login failed")
		return
	}
	if _, err := s.Auth.Svc.StartSession(r.Context(), w, uid); err != nil {
		s.fail(w, r, "session failed")
		return
	}
	_, _ = s.Auth.Svc.LoginResult(r.Context(), id.Email, true)
	http.Redirect(w, r, s.PortalURL+st.Next, http.StatusSeeOther)
}

func (s *SSO) fail(w http.ResponseWriter, r *http.Request, msg string) {
	http.Redirect(w, r, s.PortalURL+"/login?error="+url.QueryEscape(msg), http.StatusSeeOther)
}

func providerKey(r *http.Request) string {
	// /api/auth/sso/{key}/(start|callback)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

func safeNext(n string) string {
	if strings.HasPrefix(n, "/") && !strings.HasPrefix(n, "//") && len(n) < 2000 {
		return n
	}
	return "/dashboard"
}
