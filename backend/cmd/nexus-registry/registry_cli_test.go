package main

// Registry эзэмшигчийн хэрэгсэл: keygen → build → verify гинж, гарын үсгийн
// зөрчил, дүрэм зөрчсөн манифестыг барих.

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/pkg/registry"
)

func writeManifest(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

const goodManifest = `{"id":"mn.test.inv","short_id":"inv","name":"Inv","version":"1.0.0",
 "go_module":"example.com/m","import":"example.com/m/inv","permissions":[{"code":"inv.read","name":"r"}]}`

func TestBuildAndVerify(t *testing.T) {
	dir := t.TempDir()
	manifests := filepath.Join(dir, "manifests")
	if err := os.MkdirAll(manifests, 0o755); err != nil {
		t.Fatal(err)
	}
	writeManifest(t, manifests, "inv.json", goodManifest)

	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	keyPath := filepath.Join(dir, "key")
	if err := os.WriteFile(keyPath, []byte(base64.StdEncoding.EncodeToString(priv)), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := build([]string{"-manifests", manifests, "-key", keyPath, "-out", dir}); err != nil {
		t.Fatal(err)
	}
	pubB64 := base64.StdEncoding.EncodeToString(pub)
	if err := verify([]string{"-dir", dir, "-keys", pubB64}); err != nil {
		t.Fatalf("verify: %v", err)
	}
	// Өөр түлхүүрээр — унана.
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	if err := verify([]string{"-dir", dir, "-keys", base64.StdEncoding.EncodeToString(otherPub)}); err == nil {
		t.Fatal("өөр түлхүүрээр батлагдав")
	}
	// index.json-ийг өөрчилвөл — унана.
	idxPath := filepath.Join(dir, "index.json")
	raw, _ := os.ReadFile(idxPath)
	var ix registry.Index
	if err := json.Unmarshal(raw, &ix); err != nil {
		t.Fatal(err)
	}
	if len(ix.Apps) != 1 || ix.Apps[0].ShortID != "inv" {
		t.Fatalf("index = %+v", ix)
	}
	tampered := append([]byte{}, raw...)
	tampered[len(tampered)-3] ^= 1
	if err := os.WriteFile(idxPath, tampered, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verify([]string{"-dir", dir, "-keys", pubB64}); err == nil {
		t.Fatal("өөрчилсөн index батлагдав")
	}
}

func TestBuildRejectsInvalidManifest(t *testing.T) {
	dir := t.TempDir()
	manifests := filepath.Join(dir, "manifests")
	if err := os.MkdirAll(manifests, 0o755); err != nil {
		t.Fatal(err)
	}
	// permission нь short_id-ийн prefix-гүй.
	writeManifest(t, manifests, "bad.json", `{"id":"mn.test.inv","short_id":"inv","name":"Inv","version":"1.0.0",
	 "go_module":"example.com/m","permissions":[{"code":"өөр.read","name":"r"}]}`)
	if err := build([]string{"-manifests", manifests, "-out", dir}); err == nil {
		t.Fatal("дүрэм зөрчсөн манифест хүлээн авагдав")
	}
	// Хоосон хавтас.
	empty := filepath.Join(dir, "empty")
	_ = os.MkdirAll(empty, 0o755)
	if err := build([]string{"-manifests", empty, "-out", dir}); err == nil {
		t.Fatal("хоосон хавтас хүлээн авагдав")
	}
	// Давхардсан апп.
	dup := filepath.Join(dir, "dup")
	_ = os.MkdirAll(dup, 0o755)
	writeManifest(t, dup, "a.json", goodManifest)
	writeManifest(t, dup, "b.json", goodManifest)
	if err := build([]string{"-manifests", dup, "-out", dir}); err == nil {
		t.Fatal("давхардсан апп хүлээн авагдав")
	}
}

func TestBuildWithoutKeyLeavesUnsigned(t *testing.T) {
	dir := t.TempDir()
	manifests := filepath.Join(dir, "manifests")
	_ = os.MkdirAll(manifests, 0o755)
	writeManifest(t, manifests, "inv.json", goodManifest)
	if err := build([]string{"-manifests", manifests, "-out", dir}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "index.json.sig")); err == nil {
		t.Fatal("түлхүүргүй байхад .sig үүсэв")
	}
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	if err := verify([]string{"-dir", dir, "-keys", base64.StdEncoding.EncodeToString(pub)}); err == nil {
		t.Fatal("гарын үсэггүй index батлагдав")
	}
}

func TestReadPrivRejectsBadKey(t *testing.T) {
	p := filepath.Join(t.TempDir(), "k")
	if err := os.WriteFile(p, []byte("тийм-биш"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPriv(p); err == nil {
		t.Fatal("буруу түлхүүр хүлээн авагдав")
	}
	if _, err := readPriv(filepath.Join(t.TempDir(), "байхгүй")); err == nil {
		t.Fatal("байхгүй файл хүлээн авагдав")
	}
}

func TestKeygenPrintsUsablePair(t *testing.T) {
	// keygen нь stdout руу бичдэг — түр дамжуулж авна.
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = keygen()
	_ = w.Close()
	os.Stdout = old
	if err != nil {
		t.Fatal(err)
	}
	out, _ := io.ReadAll(r)
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 4 {
		t.Fatalf("keygen гаралт:\n%s", out)
	}
	priv, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[1]))
	if err != nil || len(priv) != ed25519.PrivateKeySize {
		t.Fatalf("private key = %d байт, %v", len(priv), err)
	}
	pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(lines[3]))
	if err != nil || len(pub) != ed25519.PublicKeySize {
		t.Fatalf("public key = %d байт, %v", len(pub), err)
	}
	// Хос болох эсэх: гарын үсэг зурж баталгаажуулна.
	msg := []byte("тест")
	sig := registry.Sign(ed25519.PrivateKey(priv), msg)
	if err := registry.Verify([]ed25519.PublicKey{ed25519.PublicKey(pub)}, msg, sig); err != nil {
		t.Fatalf("түлхүүрийн хос таарахгүй: %v", err)
	}
}
