// nexus-registry — registry эзэмшигчийн хэрэгсэл: түлхүүр үүсгэх, манифест
// хавтаснаас index.json + .sig бүтээх, шалгах.
//
//	go run ./cmd/nexus-registry keygen                 → stdout: private/public (base64)
//	go run ./cmd/nexus-registry build -manifests DIR -key FILE -out DIR
//	go run ./cmd/nexus-registry verify -dir DIR -keys b64[,b64]
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/pkg/registry"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "nexus-registry keygen | build | verify")
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "keygen":
		err = keygen()
	case "build":
		err = build(os.Args[2:])
	case "verify":
		err = verify(os.Args[2:])
	default:
		err = fmt.Errorf("үл мэдэх комманд %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "алдаа:", err)
		os.Exit(1)
	}
}

func keygen() error {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	fmt.Printf("private (нууц, файлд хадгал; mode 600):\n%s\npublic (REGISTRY_KEYS):\n%s\n",
		base64.StdEncoding.EncodeToString(priv), base64.StdEncoding.EncodeToString(pub))
	return nil
}

func readPriv(path string) (ed25519.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	k, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(b)))
	if err != nil || len(k) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key буруу (%s)", path)
	}
	return ed25519.PrivateKey(k), nil
}

func build(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	dir := fs.String("manifests", "manifests", "манифест JSON файлуудын хавтас")
	key := fs.String("key", "", "private key файл (base64)")
	out := fs.String("out", ".", "index.json + .sig бичих хавтас")
	if err := fs.Parse(args); err != nil {
		return err
	}
	files, _ := filepath.Glob(filepath.Join(*dir, "*.json"))
	if len(files) == 0 {
		return fmt.Errorf("%s: манифест олдсонгүй", *dir)
	}
	sort.Strings(files)
	ix := registry.Index{GeneratedAt: time.Now().UTC().Truncate(time.Second)}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		var m registry.Manifest
		if err := json.Unmarshal(raw, &m); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		if err := m.Validate(); err != nil {
			return fmt.Errorf("%s: %w", f, err)
		}
		ix.Apps = append(ix.Apps, m)
	}
	if err := ix.Validate(); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(*out, "index.json"), raw, 0o644); err != nil {
		return err
	}
	if *key == "" {
		fmt.Println("АНХААР: -key өгөөгүй — index.json гарын үсэггүй (зөвхөн локал fallback-д хэрэглэгдэнэ)")
		return nil
	}
	priv, err := readPriv(*key)
	if err != nil {
		return err
	}
	sig := registry.Sign(priv, raw)
	if err := os.WriteFile(filepath.Join(*out, "index.json.sig"), []byte(sig+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Printf("index.json (%d апп) + index.json.sig бичигдлээ\n", len(ix.Apps))
	return nil
}

func verify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	dir := fs.String("dir", ".", "index.json + .sig байгаа хавтас")
	keys := fs.String("keys", "", "нийтийн түлхүүр(үүд), base64, таслалаар")
	if err := fs.Parse(args); err != nil {
		return err
	}
	raw, err := os.ReadFile(filepath.Join(*dir, "index.json"))
	if err != nil {
		return err
	}
	sig, err := os.ReadFile(filepath.Join(*dir, "index.json.sig"))
	if err != nil {
		return err
	}
	pubs, err := registry.ParseKeys(*keys)
	if err != nil {
		return err
	}
	if err := registry.Verify(pubs, raw, string(sig)); err != nil {
		return err
	}
	ix, err := registry.Parse(raw)
	if err != nil {
		return err
	}
	fmt.Printf("ok: %d апп, generated_at %s\n", len(ix.Apps), ix.GeneratedAt.Format(time.RFC3339))
	return nil
}
