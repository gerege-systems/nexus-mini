// Package oidc — OpenID Connect provider: authorization code + PKCE (заавал),
// refresh rotation (replay илэрвэл гэр бүлээр хүчингүй), client_credentials,
// RS256 id_token + JWKS, discovery, userinfo, introspect, revoke, end_session.
//
// Зориуд хийгээгүй: implicit/hybrid flow, JWT access token (opaque + introspect
// нь хүчингүй болгож чаддаг), dynamic client registration. JWT-г өөрсдөө
// бичдэг (зөвхөн RS256, sign-only) — alg confusion/none-ийн зам байхгүй.
package oidc

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type signingKey struct {
	kid  string
	priv *rsa.PrivateKey
}

// loadOrCreateKey — идэвхтэй RSA түлхүүрийг DB-ээс, байхгүй бол үүсгэж хадгална
// (олон процесс зэрэг асахад нэг нь л ялна — kid давхардвал дахин уншина).
func loadOrCreateKey(ctx context.Context, pool *pgxpool.Pool) (*signingKey, error) {
	for attempt := 0; attempt < 2; attempt++ {
		var kid, privPEM string
		err := pool.QueryRow(ctx,
			`SELECT kid, private_pem FROM oidc_keys WHERE active ORDER BY created_at DESC LIMIT 1`).Scan(&kid, &privPEM)
		if err == nil {
			block, _ := pem.Decode([]byte(privPEM))
			if block == nil {
				return nil, errors.New("oidc: private_pem decode")
			}
			priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("oidc: key parse: %w", err)
			}
			return &signingKey{kid: kid, priv: priv}, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, err
		}
		b := make([]byte, 8)
		_, _ = rand.Read(b)
		kid = hex.EncodeToString(b)
		privPEM = string(pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)}))
		pubDER, _ := x509.MarshalPKIXPublicKey(&priv.PublicKey)
		pubPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}))
		if _, err := pool.Exec(ctx,
			`INSERT INTO oidc_keys (kid, private_pem, public_pem) VALUES ($1::varchar(32), $2::varchar(4000), $3::varchar(1000))`,
			kid, privPEM, pubPEM); err != nil {
			continue // зэрэг үүсгэсэн бол нөгөөгөөр нь уншина
		}
		return &signingKey{kid: kid, priv: priv}, nil
	}
	return nil, errors.New("oidc: түлхүүр үүсгэж чадсангүй")
}

// JWK — RFC 7517 нийтийн түлхүүр.
func (k *signingKey) jwk() map[string]any {
	return map[string]any{
		"kty": "RSA", "use": "sig", "alg": "RS256", "kid": k.kid,
		"n": b64url(k.priv.PublicKey.N.Bytes()),
		"e": b64url(big.NewInt(int64(k.priv.PublicKey.E)).Bytes()),
	}
}

func b64url(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

func sha256hex(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func randToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// keyCache — процесс бүр нэг удаа уншина.
type keyCache struct {
	mu  sync.Mutex
	key *signingKey
}
