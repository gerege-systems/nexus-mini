package core

// Цөмийн HTTP middleware-үүдийн unit тест (DB шаардахгүй): rate limit/лог-ийн
// IP-г клиентийн толгойгоор хуурч болохгүй, CSRF-ийн хоёр давхарга, аюулгүй
// байдлын толгойнууд, OAuth2-ийн CORS.

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func ok(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func TestClientIPIgnoresSpoofableHeaders(t *testing.T) {
	cases := []struct {
		name, remote, xreal, trueClient, xff, want string
	}{
		{"гаднаас X-Real-IP хуурмаг", "203.0.113.9:1234", "1.2.3.4", "", "", "203.0.113.9"},
		{"True-Client-IP хэзээ ч биш", "203.0.113.9:1234", "", "1.2.3.4", "", "203.0.113.9"},
		{"X-Forwarded-For хэзээ ч биш", "10.0.0.5:1234", "", "", "1.2.3.4", "10.0.0.5"},
		{"loopback proxy-гоос X-Real-IP", "127.0.0.1:5555", "198.51.100.7", "1.2.3.4", "9.9.9.9", "198.51.100.7"},
		{"private proxy-гоос X-Real-IP", "10.0.0.2:5555", "198.51.100.7", "", "", "198.51.100.7"},
		{"proxy-гоос ирсэн ч утга буруу", "127.0.0.1:5555", "not-an-ip", "", "", "127.0.0.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got string
			h := clientIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.RemoteAddr
			}))
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = c.remote
			if c.xreal != "" {
				r.Header.Set("X-Real-IP", c.xreal)
			}
			if c.trueClient != "" {
				r.Header.Set("True-Client-IP", c.trueClient)
			}
			if c.xff != "" {
				r.Header.Set("X-Forwarded-For", c.xff)
			}
			h.ServeHTTP(httptest.NewRecorder(), r)
			if host, _, _ := splitHostPortSafe(got); host != c.want {
				t.Fatalf("RemoteAddr = %q, хүлээсэн IP %q", got, c.want)
			}
		})
	}
}

func splitHostPortSafe(addr string) (string, string, error) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
	}
	return addr, "", nil
}

func TestSameOriginOnly(t *testing.T) {
	cases := []struct {
		name, method, path, origin, fwdHost, secFetch string
		want                                          int
	}{
		{"GET үргэлж нээлттэй", http.MethodGet, "/api/me", "https://evil.mn", "", "cross-site", 200},
		{"Origin-гүй (curl)", http.MethodPost, "/api/tenants", "", "", "", 200},
		{"ижил origin", http.MethodPost, "/api/tenants", "http://example.com", "", "same-origin", 200},
		{"өөр origin", http.MethodPost, "/api/tenants", "https://evil.mn", "", "", 403},
		{"X-Forwarded-Host-той (Docker rewrite)", http.MethodPost, "/api/tenants", "https://portal.mn", "portal.mn", "", 200},
		{"Sec-Fetch-Site: cross-site", http.MethodPost, "/api/tenants", "", "", "cross-site", 403},
		{"handover чөлөөлөгдсөн", http.MethodPost, "/api/auth/handover", "https://admin.mn", "", "cross-site", 200},
		{"oauth2 token чөлөөлөгдсөн", http.MethodPost, "/api/oauth2/token", "https://spa.mn", "", "cross-site", 200},
		{"oauth2 consent чөлөөлөгдөөгүй", http.MethodPost, "/api/oauth2/consent", "https://evil.mn", "", "", 403},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest(c.method, c.path, nil)
			r.Host = "example.com"
			if c.origin != "" {
				r.Header.Set("Origin", c.origin)
			}
			if c.fwdHost != "" {
				r.Header.Set("X-Forwarded-Host", c.fwdHost)
			}
			if c.secFetch != "" {
				r.Header.Set("Sec-Fetch-Site", c.secFetch)
			}
			w := httptest.NewRecorder()
			sameOriginOnly(http.HandlerFunc(ok)).ServeHTTP(w, r)
			if w.Code != c.want {
				t.Fatalf("код = %d, хүлээсэн %d", w.Code, c.want)
			}
		})
	}
}

func TestSecurityHeaders(t *testing.T) {
	for _, prod := range []bool{false, true} {
		w := httptest.NewRecorder()
		securityHeaders(prod)(http.HandlerFunc(ok)).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/me", nil))
		h := w.Header()
		for k, want := range map[string]string{
			"X-Content-Type-Options":  "nosniff",
			"X-Frame-Options":         "DENY",
			"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
			"Cache-Control":           "no-store",
		} {
			if h.Get(k) != want {
				t.Errorf("prod=%v %s = %q, хүлээсэн %q", prod, k, h.Get(k), want)
			}
		}
		if hsts := h.Get("Strict-Transport-Security"); (hsts != "") != prod {
			t.Errorf("prod=%v HSTS = %q", prod, hsts)
		}
	}
}

func TestOAuthCORS(t *testing.T) {
	// Token endpoint: CORS нээлттэй + preflight.
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodOptions, "/api/oauth2/token", nil)
	oauthCORS(http.HandlerFunc(ok)).ServeHTTP(w, r)
	if w.Code != http.StatusNoContent || w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("preflight = %d, ACAO = %q", w.Code, w.Header().Get("Access-Control-Allow-Origin"))
	}
	// Consent нь cookie-тэй тул CORS нээхгүй.
	w = httptest.NewRecorder()
	r = httptest.NewRequest(http.MethodGet, "/api/oauth2/consent", nil)
	oauthCORS(http.HandlerFunc(ok)).ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("consent-д CORS нээгдсэн байна")
	}
}

// requestLog query string-ийг (имэйл, код, JWT) логлохгүй.
func TestRequestLogOmitsQuery(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(prev) })
	h := requestLog(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) }))
	r := httptest.NewRequest(http.MethodGet, "/api/members/lookup?email=secret%40x.mn&id_token_hint=eyJ", nil)
	h.ServeHTTP(httptest.NewRecorder(), r)
	out := buf.String()
	if !strings.Contains(out, "GET /api/members/lookup 204") {
		t.Fatalf("лог = %q", out)
	}
	if strings.Contains(out, "secret") || strings.Contains(out, "eyJ") || strings.Contains(out, "?") {
		t.Fatalf("query string логт орсон: %q", out)
	}
}
