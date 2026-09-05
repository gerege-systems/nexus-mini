// Package ssoclient — энэ instance ГАДНЫ OIDC provider-ийн relying party болно:
// Google, өөр nexus-mini (federation), дурын OIDC issuer. Authorization code
// + PKCE + state + nonce; id_token-ийг issuer-ийн JWKS-ээр (RS256) шалгана.
//
// Тохиргоо env: SSO_ISSUER / SSO_CLIENT_ID / SSO_CLIENT_SECRET / SSO_NAME
// (ерөнхий provider), GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET (Google).
// SSO_AUTO_SIGNUP=true бол танигдаагүй имэйлд данс үүсгэнэ (JIT), үгүй бол
// зөвхөн бүртгэлтэй хэрэглэгч нэвтэрнэ (default — хаалттай).
package ssoclient

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Provider struct {
	Key, Name, Issuer, ClientID, ClientSecret string
	// TrustEmail — issuer-ийн email claim-ийг email_verified=false байсан ч
	// баталгаажсан гэж үзэх. Зөвхөн операторын ил шийдвэр (SSO_TRUST_EMAIL):
	// nexus-mini өөрөө provider болохдоо email_verified=false өгдөг тул
	// federation-д JIT/имэйлээр холбохын тулд хэрэгтэй. Default false.
	TrustEmail bool
}

type discovery struct {
	Issuer   string `json:"issuer"`
	AuthURL  string `json:"authorization_endpoint"`
	TokenURL string `json:"token_endpoint"`
	JWKSURL  string `json:"jwks_uri"`
	Userinfo string `json:"userinfo_endpoint"`
	fetched  time.Time
}

type Client struct {
	providers map[string]Provider
	order     []string
	http      *http.Client
	mu        sync.Mutex
	disc      map[string]*discovery
	jwks      map[string]jwksEntry
}

type jwksEntry struct {
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

func New(providers []Provider) *Client {
	c := &Client{providers: map[string]Provider{}, http: &http.Client{Timeout: 10 * time.Second},
		disc: map[string]*discovery{}, jwks: map[string]jwksEntry{}}
	for _, p := range providers {
		if p.Issuer != "" && p.ClientID != "" {
			c.providers[p.Key] = p
			c.order = append(c.order, p.Key)
		}
	}
	return c
}

// Public — нэвтрэх хуудасны товчнууд.
func (c *Client) Public() []map[string]string {
	out := []map[string]string{}
	for _, k := range c.order {
		out = append(out, map[string]string{"key": k, "name": c.providers[k].Name})
	}
	return out
}

func (c *Client) Get(key string) (Provider, bool) { p, ok := c.providers[key]; return p, ok }

func (c *Client) discover(ctx context.Context, p Provider) (*discovery, error) {
	c.mu.Lock()
	d := c.disc[p.Key]
	c.mu.Unlock()
	if d != nil && time.Since(d.fetched) < time.Hour {
		return d, nil
	}
	var nd discovery
	if err := c.getJSON(ctx, strings.TrimRight(p.Issuer, "/")+"/.well-known/openid-configuration", &nd); err != nil {
		return nil, fmt.Errorf("discovery %s: %w", p.Key, err)
	}
	if nd.Issuer != p.Issuer && nd.Issuer != strings.TrimRight(p.Issuer, "/") {
		return nil, fmt.Errorf("discovery: issuer зөрүү (%s ≠ %s)", nd.Issuer, p.Issuer)
	}
	nd.fetched = time.Now()
	c.mu.Lock()
	c.disc[p.Key] = &nd
	c.mu.Unlock()
	return &nd, nil
}

func (c *Client) getJSON(ctx context.Context, u string, v any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(v)
}

// AuthURL — redirect хийх URL (PKCE verifier/state/nonce-ийг дуудагч cookie-д хадгална).
func (c *Client) AuthURL(ctx context.Context, p Provider, redirectURI, state, nonce, challenge string) (string, error) {
	d, err := c.discover(ctx, p)
	if err != nil {
		return "", err
	}
	q := url.Values{
		"response_type": {"code"}, "client_id": {p.ClientID}, "redirect_uri": {redirectURI},
		"scope": {"openid email profile"}, "state": {state}, "nonce": {nonce},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"},
	}
	sep := "?"
	if strings.Contains(d.AuthURL, "?") {
		sep = "&"
	}
	return d.AuthURL + sep + q.Encode(), nil
}

// Identity — баталгаажсан id_token-оос.
type Identity struct {
	Subject, Email, Name string
	EmailVerified        bool
}

// Exchange — код → токен → id_token шалгалт (iss, aud, exp, nonce, RS256 JWKS).
func (c *Client) Exchange(ctx context.Context, p Provider, redirectURI, code, verifier, nonce string) (*Identity, error) {
	d, err := c.discover(ctx, p)
	if err != nil {
		return nil, err
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {redirectURI},
		"code_verifier": {verifier}, "client_id": {p.ClientID}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, d.TokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if p.ClientSecret != "" {
		req.SetBasicAuth(url.QueryEscape(p.ClientID), url.QueryEscape(p.ClientSecret))
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var tok struct {
		IDToken string `json:"id_token"`
		Error   string `json:"error"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tok); err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 || tok.IDToken == "" {
		return nil, fmt.Errorf("token endpoint: HTTP %d %s", resp.StatusCode, tok.Error)
	}
	claims, err := c.verifyIDToken(ctx, p, d, tok.IDToken)
	if err != nil {
		return nil, err
	}
	if n, _ := claims["nonce"].(string); n != nonce {
		return nil, errors.New("nonce зөрүү")
	}
	id := &Identity{}
	id.Subject, _ = claims["sub"].(string)
	id.Email, _ = claims["email"].(string)
	id.Name, _ = claims["name"].(string)
	id.EmailVerified, _ = claims["email_verified"].(bool)
	if id.Subject == "" || id.Email == "" {
		return nil, errors.New("id_token-д sub/email алга")
	}
	id.Email = strings.ToLower(strings.TrimSpace(id.Email))
	return id, nil
}

func (c *Client) keys(ctx context.Context, p Provider, d *discovery, force bool) (map[string]*rsa.PublicKey, error) {
	c.mu.Lock()
	e, ok := c.jwks[p.Key]
	c.mu.Unlock()
	if ok && !force && time.Since(e.fetched) < time.Hour {
		return e.keys, nil
	}
	var set struct {
		Keys []struct{ Kty, Kid, N, E, Alg, Use string } `json:"keys"`
	}
	if err := c.getJSON(ctx, d.JWKSURL, &set); err != nil {
		return nil, err
	}
	out := map[string]*rsa.PublicKey{}
	for _, k := range set.Keys {
		if k.Kty != "RSA" || (k.Use != "" && k.Use != "sig") {
			continue
		}
		nb, err1 := base64.RawURLEncoding.DecodeString(k.N)
		eb, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil {
			continue
		}
		out[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(new(big.Int).SetBytes(eb).Int64())}
	}
	c.mu.Lock()
	c.jwks[p.Key] = jwksEntry{keys: out, fetched: time.Now()}
	c.mu.Unlock()
	return out, nil
}

func (c *Client) verifyIDToken(ctx context.Context, p Provider, d *discovery, token string) (map[string]any, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("id_token формат")
	}
	hb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}
	var hdr struct{ Alg, Kid string }
	if json.Unmarshal(hb, &hdr) != nil || hdr.Alg != "RS256" {
		return nil, errors.New("зөвхөн RS256")
	}
	keys, err := c.keys(ctx, p, d, false)
	if err != nil {
		return nil, err
	}
	pub, ok := keys[hdr.Kid]
	if !ok {
		// Түлхүүр эргэлдсэн байж магадгүй — нэг удаа дахин татна.
		if keys, err = c.keys(ctx, p, d, true); err != nil {
			return nil, err
		}
		if pub, ok = keys[hdr.Kid]; !ok {
			return nil, errors.New("kid олдсонгүй")
		}
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], sig); err != nil {
		return nil, errors.New("гарын үсэг буруу")
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var claims map[string]any
	if json.Unmarshal(pb, &claims) != nil {
		return nil, errors.New("claims")
	}
	if iss, _ := claims["iss"].(string); iss != d.Issuer && iss != strings.TrimRight(p.Issuer, "/") && iss != p.Issuer {
		return nil, errors.New("iss зөрүү")
	}
	if !audContains(claims["aud"], p.ClientID) {
		return nil, errors.New("aud зөрүү")
	}
	if exp, _ := claims["exp"].(float64); exp == 0 || time.Now().Unix() > int64(exp)+60 {
		return nil, errors.New("id_token хугацаа дууссан")
	}
	return claims, nil
}

func audContains(aud any, id string) bool {
	switch v := aud.(type) {
	case string:
		return v == id
	case []any:
		for _, x := range v {
			if s, _ := x.(string); s == id {
				return true
			}
		}
	}
	return false
}

// PKCE тусламж.
func NewPKCE() (verifier, challenge string) {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	return verifier, base64.RawURLEncoding.EncodeToString(sum[:])
}

func RandString() string {
	b := make([]byte, 24)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
