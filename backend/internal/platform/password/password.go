// Package password — argon2id hash.
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	timeCost   = 2
	memoryCost = 64 * 1024
	threads    = 2
	keyLen     = 32
	saltLen    = 16
)

// argon2 нэг дуудлагадаа 64MB эзэлдэг — зэрэг ажиллах тоог хязгаарлахгүй
// бол login-ий шуурга RAM-аар DoS хийнэ. Нэг зэрэг ≤4 (256MB тааз).
var sem = make(chan struct{}, 4)

func idKey(pass, salt []byte, t, m uint32, p uint8, l uint32) []byte {
	sem <- struct{}{}
	defer func() { <-sem }()
	return argon2.IDKey(pass, salt, t, m, p, l)
}

func Hash(plain string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := idKey([]byte(plain), salt, timeCost, memoryCost, threads, keyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		memoryCost, timeCost, threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key)), nil
}

func Verify(plain, encoded string) bool {
	// base64 нь өөрөө $ агуулдаггүй тул Split аюулгүй (docs/01-lessons.md #8
	// нь base64url доторх '_'-ийн тухай — тэнд SplitN хэрэглэнэ).
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var m, t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false
	}
	got := idKey([]byte(plain), salt, t, m, p, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}
