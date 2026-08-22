package oidc

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/json"
	"strings"
)

// signJWT — RS256 sign-only (header alg/kid, payload claims). Баталгаажуулах
// талыг (SSO client) ssoclient package хийнэ.
func signJWT(k *signingKey, claims map[string]any) (string, error) {
	hdr, _ := json.Marshal(map[string]string{"alg": "RS256", "typ": "JWT", "kid": k.kid})
	body, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signing := b64url(hdr) + "." + b64url(body)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, k.priv, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return signing + "." + b64url(sig), nil
}

// verifyJWT — энэ provider-ийн өөрийн түлхүүрээр (end_session-ийн id_token_hint).
func verifyJWT(k *signingKey, token string) (map[string]any, bool) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, false
	}
	var hdr struct{ Alg, Kid string }
	hb, err := b64urlDecode(parts[0])
	if err != nil || json.Unmarshal(hb, &hdr) != nil || hdr.Alg != "RS256" || hdr.Kid != k.kid {
		return nil, false
	}
	sig, err := b64urlDecode(parts[2])
	if err != nil {
		return nil, false
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if rsa.VerifyPKCS1v15(&k.priv.PublicKey, crypto.SHA256, sum[:], sig) != nil {
		return nil, false
	}
	pb, err := b64urlDecode(parts[1])
	if err != nil {
		return nil, false
	}
	var claims map[string]any
	if json.Unmarshal(pb, &claims) != nil {
		return nil, false
	}
	return claims, true
}
