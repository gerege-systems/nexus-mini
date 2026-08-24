package registry

// Registry татах давхарга: гарын үсэг заавал, өөрчилсөн байт татгалзана,
// хуучин index (replay) кэшийг дардаггүй, офлайн үед кэш ажиллана, ETag 304,
// хэмжээний хязгаар.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type idpServer struct {
	srv     *httptest.Server
	raw     atomic.Value // []byte
	sig     atomic.Value // string
	etag    string
	hits    atomic.Int32
	sigHits atomic.Int32
}

func newRegistryServer(t *testing.T, raw []byte, sig string) *idpServer {
	t.Helper()
	s := &idpServer{etag: `W/"1"`}
	s.raw.Store(raw)
	s.sig.Store(sig)
	mux := http.NewServeMux()
	mux.HandleFunc("/index.json", func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		if r.Header.Get("If-None-Match") == s.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", s.etag)
		_, _ = w.Write(s.raw.Load().([]byte))
	})
	mux.HandleFunc("/index.json.sig", func(w http.ResponseWriter, r *http.Request) {
		s.sigHits.Add(1)
		_, _ = w.Write([]byte(s.sig.Load().(string)))
	})
	s.srv = httptest.NewServer(mux)
	t.Cleanup(s.srv.Close)
	return s
}

func sampleIndex(t *testing.T, when time.Time, version string) ([]byte, Index) {
	t.Helper()
	ix := Index{GeneratedAt: when.UTC().Truncate(time.Second), Apps: []Manifest{{
		ID: "mn.test.inv", ShortID: "inv", Name: "Inv", Version: version,
		GoModule: "example.com/mod", Import: "example.com/mod/inv",
		Permissions: []Permission{{Code: "inv.read", Name: "r"}},
	}}}
	raw, err := json.Marshal(ix)
	if err != nil {
		t.Fatal(err)
	}
	return raw, ix
}

func TestFetchVerifiesSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	raw, _ := sampleIndex(t, time.Now(), "1.0.0")
	s := newRegistryServer(t, raw, Sign(priv, raw))
	cache := t.TempDir()
	ctx := context.Background()

	ix, err := Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, cache)
	if err != nil || ix == nil || len(ix.Apps) != 1 {
		t.Fatalf("Fetch = %v, %v", ix, err)
	}
	if _, err := os.Stat(filepath.Join(cache, "index.json")); err != nil {
		t.Fatal("кэш бичигдээгүй")
	}

	// Түлхүүргүй — огт татахгүй.
	if _, err := Fetch(ctx, s.srv.URL+"/index.json", nil, cache); err == nil {
		t.Fatal("түлхүүргүй Fetch амжилттай болов")
	}

	// Өөр түлхүүрээр гарын үсэглэсэн байт — кэш рүү буцна.
	otherPub, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
	s.etag = `W/"2"`
	bad, _ := sampleIndex(t, time.Now().Add(time.Minute), "9.9.9")
	s.raw.Store(bad)
	s.sig.Store(Sign(otherPriv, bad))
	ix, err = Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, cache)
	if err == nil {
		t.Fatal("буруу гарын үсэг хүлээн авагдав")
	}
	if ix == nil || ix.Apps[0].Version != "1.0.0" {
		t.Fatalf("кэшийн хувилбар руу буцаагүй: %+v", ix)
	}
	// Тэр түлхүүрийг зөвшөөрвөл дамжина.
	if ix, err = Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub, otherPub}, cache); err != nil || ix.Apps[0].Version != "9.9.9" {
		t.Fatalf("хоёр түлхүүрийн нэг нь таарахад: %v %v", ix, err)
	}
}

func TestFetchRejectsReplayAndFuture(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	now := time.Now()
	raw, _ := sampleIndex(t, now, "2.0.0")
	s := newRegistryServer(t, raw, Sign(priv, raw))
	cache := t.TempDir()
	ctx := context.Background()
	if _, err := Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, cache); err != nil {
		t.Fatal(err)
	}
	// Хуучин (replay) — гарын үсэг зөв ч кэшээс хуучин.
	old, _ := sampleIndex(t, now.Add(-24*time.Hour), "1.0.0")
	s.etag = `W/"old"`
	s.raw.Store(old)
	s.sig.Store(Sign(priv, old))
	ix, err := Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, cache)
	if err == nil || ix == nil || ix.Apps[0].Version != "2.0.0" {
		t.Fatalf("replay хүлээн авагдав: %v %v", ix, err)
	}
	// Ирээдүйн огноо.
	future, _ := sampleIndex(t, now.Add(48*time.Hour), "3.0.0")
	s.etag = `W/"future"`
	s.raw.Store(future)
	s.sig.Store(Sign(priv, future))
	if _, err := Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, filepath.Join(t.TempDir())); err == nil {
		t.Fatal("ирээдүйн generated_at хүлээн авагдав")
	}
}

func TestFetchUsesETagAndOfflineCache(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	raw, _ := sampleIndex(t, time.Now(), "1.2.3")
	s := newRegistryServer(t, raw, Sign(priv, raw))
	cache := t.TempDir()
	ctx := context.Background()
	if _, err := Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, cache); err != nil {
		t.Fatal(err)
	}
	before := s.sigHits.Load()
	// Хоёр дахь удаа: 304 → .sig дахин татахгүй.
	ix, err := Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, cache)
	if err != nil || ix.Apps[0].Version != "1.2.3" {
		t.Fatalf("304 зам: %v %v", ix, err)
	}
	if s.sigHits.Load() != before {
		t.Fatal("304 үед .sig дахин татагдав")
	}
	// Сервер унтарсан — кэшээс.
	s.srv.Close()
	ix, err = Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, cache)
	if err != nil || ix == nil || ix.Apps[0].Version != "1.2.3" {
		t.Fatalf("офлайн кэш: %v %v", ix, err)
	}
	// Кэшгүй + офлайн = алдаа.
	if _, err := Fetch(ctx, s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, t.TempDir()); err == nil {
		t.Fatal("офлайн, кэшгүй үед амжилттай болов")
	}
}

func TestFetchRejectsHugeIndex(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	huge := append([]byte(`{"generated_at":"2026-01-01T00:00:00Z","apps":[],"_pad":"`),
		append([]byte(strings.Repeat("x", 5<<20)), []byte(`"}`)...)...)
	s := newRegistryServer(t, huge, Sign(priv, huge))
	if _, err := Fetch(context.Background(), s.srv.URL+"/index.json", []ed25519.PublicKey{pub}, t.TempDir()); err == nil {
		t.Fatal("хэт том index хүлээн авагдав")
	}
}

func TestParseKeysAndLoadFile(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	keys, err := ParseKeys(" " + base64.StdEncoding.EncodeToString(pub) + " , ")
	if err != nil || len(keys) != 1 {
		t.Fatalf("ParseKeys = %v %v", keys, err)
	}
	if _, err := ParseKeys("тийм-биш"); err == nil {
		t.Fatal("буруу түлхүүр хүлээн авагдав")
	}
	// LoadFile — гарын үсэггүй локал fallback.
	raw, _ := sampleIndex(t, time.Now(), "1.0.0")
	p := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if ix, err := LoadFile(p); err != nil || len(ix.Apps) != 1 {
		t.Fatalf("LoadFile = %v %v", ix, err)
	}
	bad := filepath.Join(t.TempDir(), "bad.json")
	_ = os.WriteFile(bad, []byte(`{"apps":[{"id":"x","short_id":"x","name":"n","version":"1.0.0","go_module":"m","permissions":[{"code":"өөр.read"}]}]}`), 0o644)
	if _, err := LoadFile(bad); err == nil {
		t.Fatal("дүрэм зөрчсөн манифест LoadFile-аар оров")
	}
}

func TestManifestValidateTable(t *testing.T) {
	base := Manifest{ID: "mn.a.inv", ShortID: "inv", Name: "N", Version: "1.0.0", GoModule: "example.com/m", Import: "example.com/m/inv",
		Permissions: []Permission{{Code: "inv.read"}}, Menus: []Menu{{ID: "inv.l", Path: "/inv"}}}
	if err := base.Validate(); err != nil {
		t.Fatalf("зөв манифест унав: %v", err)
	}
	mut := func(f func(m *Manifest)) Manifest {
		m := base
		m.Permissions = append([]Permission{}, base.Permissions...)
		m.Menus = append([]Menu{}, base.Menus...)
		f(&m)
		return m
	}
	cases := map[string]Manifest{
		"id буруу":               mut(func(m *Manifest) { m.ID = "Bad" }),
		"short_id буруу":         mut(func(m *Manifest) { m.ShortID = "Inv-1" }),
		"нэр хоосон":             mut(func(m *Manifest) { m.Name = "" }),
		"version semver биш":     mut(func(m *Manifest) { m.Version = "v1" }),
		"go_module хоосон":       mut(func(m *Manifest) { m.GoModule = "" }),
		"import модулиас гадуур": mut(func(m *Manifest) { m.Import = "example.org/өөр" }),
		"permission prefix":      mut(func(m *Manifest) { m.Permissions[0].Code = "өөр.read" }),
		"цэсний зам":             mut(func(m *Manifest) { m.Menus[0].Path = "/өөр" }),
	}
	for name, m := range cases {
		if err := m.Validate(); err == nil {
			t.Errorf("%s: алдаа хүлээсэн", name)
		}
	}
	// Index: давхардал.
	ix := Index{GeneratedAt: time.Now(), Apps: []Manifest{base, base}}
	if err := ix.Validate(); err == nil {
		t.Error("давхардсан апп хүлээн авагдав")
	}
	// ImportPath fallback.
	m := base
	m.Import = ""
	if m.ImportPath() != m.GoModule {
		t.Error("ImportPath fallback буруу")
	}
}
