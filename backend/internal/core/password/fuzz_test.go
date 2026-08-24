package password

// Fuzz: Validate ямар ч оролтод panic хийхгүй, батлагдсан нууц үг үргэлж
// hash/verify-д тэнцэнэ, дүрмийн инвариантууд зөрчигдөхгүй.

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzValidate(f *testing.F) {
	for _, s := range []string{"", "Abc123!x", "Нууцүг123!", "password-12", "  ", "a1!", strings.Repeat("x", 300), "\x00\x01", "🙂1aA!"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, p string) {
		err := Validate(p)
		if err != nil {
			return
		}
		// Батлагдсан бол: 8..128 тэмдэгт, зөвхөн ASCII 0x21..0x7e, гурван анги.
		n := utf8.RuneCountInString(p)
		if n < MinLen || n > MaxLen {
			t.Fatalf("уртаар зөрчсөн: %d", n)
		}
		var l, d, s bool
		for _, r := range p {
			if r <= 0x20 || r >= 0x7f {
				t.Fatalf("ASCII биш тэмдэгт нэвтэрлээ: %q (%U)", p, r)
			}
			switch {
			case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
				l = true
			case r >= '0' && r <= '9':
				d = true
			default:
				s = true
			}
		}
		if !l || !d || !s {
			t.Fatalf("ангиар зөрчсөн: %q (үсэг=%v тоо=%v тэмдэгт=%v)", p, l, d, s)
		}
		// Батлагдсан нууц үг hash/verify-д тэнцэнэ.
		h, err := Hash(p)
		if err != nil {
			t.Fatalf("Hash: %v", err)
		}
		if !Verify(p, h) {
			t.Fatalf("Verify(%q) = false", p)
		}
	})
}

func FuzzVerifyDoesNotPanic(f *testing.F) {
	h, _ := Hash("Abc123!x")
	f.Add("Abc123!x", h)
	f.Add("", "")
	f.Add("x", "$argon2id$v=19$m=65536,t=2,p=2$aaa$bbb")
	f.Add("x", "$argon2id$гэмтсэн")
	f.Fuzz(func(t *testing.T, plain, hash string) {
		_ = Verify(plain, hash) // panic хийхгүй байх нь л шалгуур
	})
}
