package registry

// Fuzz: гадны (registry) JSON-ыг задлахад panic гарахгүй; Parse амжилттай
// бол Validate-ын инвариантууд биелнэ; гарын үсэг байтаас хамаарна.

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add(`{"generated_at":"2026-01-01T00:00:00Z","apps":[]}`)
	f.Add(`{"apps":[{"id":"mn.a.inv","short_id":"inv","name":"N","version":"1.0.0","go_module":"m","permissions":[{"code":"inv.read"}]}]}`)
	f.Add(`{`)
	f.Add(``)
	f.Fuzz(func(t *testing.T, raw string) {
		ix, err := Parse([]byte(raw))
		if err != nil {
			return
		}
		for _, m := range ix.Apps {
			if m.ID == "" || m.ShortID == "" || m.Version == "" || m.GoModule == "" {
				t.Fatalf("хоосон талбартай манифест нэвтэрлээ: %+v", m)
			}
			for _, p := range m.Permissions {
				if !strings.HasPrefix(p.Code, m.ShortID+".") {
					t.Fatalf("prefix зөрчсөн permission нэвтэрлээ: %q", p.Code)
				}
			}
			if m.Import != "" && m.Import != m.GoModule && !strings.HasPrefix(m.Import, m.GoModule+"/") {
				t.Fatalf("import модулиас гадуур: %q ⊄ %q", m.Import, m.GoModule)
			}
		}
	})
}

func FuzzVerifySignature(f *testing.F) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	raw := []byte(`{"generated_at":"2026-01-01T00:00:00Z","apps":[]}`)
	f.Add(raw, Sign(priv, raw))
	f.Add([]byte("x"), "")
	f.Add([]byte(""), "!!!не-base64")
	f.Fuzz(func(t *testing.T, body []byte, sig string) {
		err := Verify([]ed25519.PublicKey{pub}, body, sig)
		if err == nil && sig != Sign(priv, body) {
			t.Fatalf("буруу гарын үсэг батлагдав: %q", sig)
		}
	})
}
