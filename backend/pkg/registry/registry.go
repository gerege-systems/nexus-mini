// Package registry — app store-ийн НИЙТИЙН гэрээ: модулийн манифест,
// registry index, Ed25519 гарын үсэг, татах/кэшлэх. internal/ биш — гадны
// нийтлэгч, дистрибуц, CLI бүгд үүнийг ашиглана.
//
// Registry = статик файл: index.json (+ index.json.sig). Git репо + raw URL
// хангалттай, сервер код хэрэггүй. Модулийн код өөрөө Go module тул түүний
// бүрэн бүтэн байдлыг go.sum/proxy баталгаажуулна; энд зөвхөн "хэн юуг аль
// хувилбараар нийтэлсэн" гэсэн индексийг гарын үсэглэнэ.
package registry

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime/debug"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

// DefaultURL — nexus.*.com-ийн registry (gerege-systems/nexus-registry репо).
const DefaultURL = "https://raw.githubusercontent.com/gerege-systems/nexus-registry/main/index.json"

// DefaultKeys — DefaultURL-ийн index-ийг гарын үсэглэдэг нийтийн түлхүүр(үүд),
// base64. Өөрийн registry-тэй бол REGISTRY_KEYS env-ээр өгнө.
var DefaultKeys = []string{"zCzgMubR/3snvVZsrkCAZGoWd44xGI+p5W6cDO2I05U="}

type Permission struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	OwnScope     bool     `json:"own_scope,omitempty"`
	DefaultRoles []string `json:"default_roles,omitempty"`
}

type Menu struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Path  string `json:"path"`
	Icon  string `json:"icon,omitempty"`
	Order int    `json:"order,omitempty"`
}

// Manifest — нэг модулийн нэг хувилбар. `nexus-mini manifest` кодоос
// үүсгэнэ — гараар бичихгүй (drift байхгүй).
type Manifest struct {
	ID      string `json:"id"`
	ShortID string `json:"short_id"`
	Name    string `json:"name"`
	Version string `json:"version"`
	// GoModule — Go МОДУЛИЙН үндэс (`go get <GoModule>@v<Version>`).
	GoModule string `json:"go_module"`
	// Import — модулийн Go пакежийн зам (main.go-д импортлох). Ихэвчлэн
	// GoModule-тэй ижил; дэд пакеж бол урт. Хоосон бол GoModule.
	Import      string       `json:"import,omitempty"`
	Description string       `json:"description,omitempty"`
	Publisher   string       `json:"publisher,omitempty"`
	MinCore     string       `json:"min_core,omitempty"` // "1.0.0" — шаардах цөмийн доод хувилбар
	Permissions []Permission `json:"permissions"`
	Menus       []Menu       `json:"menus,omitempty"`
}

// ImportPath — main.go-д импортлох зам.
func (m Manifest) ImportPath() string {
	if m.Import != "" {
		return m.Import
	}
	return m.GoModule
}

// Index — registry-ийн бүтэн агуулга. GeneratedAt нь гарын үсгийн дотор
// тул хуучин index-ийг дахин тулгах (replay) боломжгүй.
type Index struct {
	GeneratedAt time.Time  `json:"generated_at"`
	Apps        []Manifest `json:"apps"`
}

// Describer — модуль тайлбар/нийтлэгчээ зарлаж болно (заавал биш).
type Describer interface {
	Description() string
	Publisher() string
}

// FromModule — бүртгэгдсэн модулиас манифест. Import нь пакежийн зам
// (reflect), GoModule нь тэр пакежийг агуулах МОДУЛИЙН үндэс (build info) —
// хоёрыг ялгахгүй бол `go get <пакеж>@v1.0.0` нь агуулагч модулийг (жишээ нь
// цөмийг) тэр таг руу БУУЛГАНА.
func FromModule(m nexus.Module) Manifest {
	mf := Manifest{ID: m.ID(), ShortID: m.ShortID(), Name: m.Name(), Version: m.Version(), MinCore: "1.0.0"}
	if t := reflect.TypeOf(m); t != nil {
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		mf.Import = t.PkgPath()
		mf.GoModule = moduleRoot(mf.Import)
	}
	if d, ok := m.(Describer); ok {
		mf.Description, mf.Publisher = d.Description(), d.Publisher()
	}
	for _, p := range m.Permissions() {
		mf.Permissions = append(mf.Permissions, Permission{Code: p.Code, Name: p.Name, Description: p.Description,
			OwnScope: p.OwnScope, DefaultRoles: p.DefaultRoles})
	}
	for _, mn := range m.Menus() {
		mf.Menus = append(mf.Menus, Menu{ID: mn.ID, Label: mn.Label, Path: mn.Path, Icon: mn.Icon, Order: mn.Order})
	}
	return mf
}

// moduleRoot — пакежийн замыг агуулах модулийн үндсийг build info-оос олно
// (хамгийн урт тохирох модуль). Олдохгүй бол пакежийн зам өөрөө.
func moduleRoot(pkg string) string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return pkg
	}
	best := ""
	consider := func(path string) {
		if path == "" || len(path) > len(pkg) {
			return
		}
		if pkg == path || strings.HasPrefix(pkg, path+"/") {
			if len(path) > len(best) {
				best = path
			}
		}
	}
	consider(bi.Main.Path)
	for _, d := range bi.Deps {
		consider(d.Path)
	}
	if best == "" {
		return pkg
	}
	return best
}

var (
	idRe      = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
	shortRe   = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)
	versionRe = regexp.MustCompile(`^\d+\.\d+\.\d+([-+][0-9A-Za-z.-]+)?$`)
)

// Validate — Register-ийн дүрмүүдийн registry талын хувилбар (build хийхгүй
// шалгаж болох бүхэн).
func (m Manifest) Validate() error {
	switch {
	case !idRe.MatchString(m.ID):
		return fmt.Errorf("%s: id буруу", m.ID)
	case !shortRe.MatchString(m.ShortID):
		return fmt.Errorf("%s: short_id буруу", m.ID)
	case strings.TrimSpace(m.Name) == "" || len(m.Name) > 160:
		return fmt.Errorf("%s: name 1..160", m.ID)
	case !versionRe.MatchString(m.Version):
		return fmt.Errorf("%s: version semver биш: %q", m.ID, m.Version)
	case m.GoModule == "" || len(m.GoModule) > 255:
		return fmt.Errorf("%s: go_module шаардлагатай", m.ID)
	case m.Import != "" && m.Import != m.GoModule && !strings.HasPrefix(m.Import, m.GoModule+"/"):
		return fmt.Errorf("%s: import (%s) нь go_module (%s) дотор байх ёстой", m.ID, m.Import, m.GoModule)
	case len(m.Description) > 1000 || len(m.Publisher) > 120:
		return fmt.Errorf("%s: description ≤1000, publisher ≤120", m.ID)
	}
	for _, p := range m.Permissions {
		if !strings.HasPrefix(p.Code, m.ShortID+".") {
			return fmt.Errorf("%s: permission %q нь %s. prefix-тэй байх ёстой", m.ID, p.Code, m.ShortID)
		}
	}
	for _, mn := range m.Menus {
		if mn.Path != "/"+m.ShortID && !strings.HasPrefix(mn.Path, "/"+m.ShortID+"/") {
			return fmt.Errorf("%s: цэс %q нь /%s/... замтай байх ёстой", m.ID, mn.Path, m.ShortID)
		}
	}
	return nil
}

// Validate — index: манифест бүр зөв, id/short_id давхардаагүй.
func (ix Index) Validate() error {
	ids, shorts := map[string]bool{}, map[string]bool{}
	for _, m := range ix.Apps {
		if err := m.Validate(); err != nil {
			return err
		}
		if ids[m.ID] || shorts[m.ShortID] {
			return fmt.Errorf("%s / %s давхардсан", m.ID, m.ShortID)
		}
		ids[m.ID], shorts[m.ShortID] = true, true
	}
	return nil
}

// ─── Гарын үсэг ───────────────────────────────────────────────────────

// Sign — index-ийн ТҮҮХИЙ байтуудыг гарын үсэглэнэ (base64). Хүлээн авагч
// яг тэр байтуудыг шалгана — JSON-ийг дахин хэвлэх/форматлахгүй.
func Sign(priv ed25519.PrivateKey, raw []byte) string {
	return base64.StdEncoding.EncodeToString(ed25519.Sign(priv, raw))
}

// Verify — аль нэг түлхүүрээр батлагдвал ok.
func Verify(pubs []ed25519.PublicKey, raw []byte, sigB64 string) error {
	sig, err := base64.StdEncoding.DecodeString(strings.TrimSpace(sigB64))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return errors.New("гарын үсгийн формат буруу")
	}
	for _, p := range pubs {
		if ed25519.Verify(p, raw, sig) {
			return nil
		}
	}
	return errors.New("гарын үсэг батлагдсангүй (REGISTRY_KEYS зөв үү?)")
}

// ParseKeys — "b64,b64" → нийтийн түлхүүрүүд.
func ParseKeys(csv string) ([]ed25519.PublicKey, error) {
	var out []ed25519.PublicKey
	for _, s := range strings.Split(csv, ",") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		b, err := base64.StdEncoding.DecodeString(s)
		if err != nil || len(b) != ed25519.PublicKeySize {
			return nil, fmt.Errorf("нийтийн түлхүүр буруу: %q", s)
		}
		out = append(out, ed25519.PublicKey(b))
	}
	return out, nil
}

// ─── Татах / кэш ──────────────────────────────────────────────────────

// Parse — түүхий байтуудаас index (гарын үсэггүй — локал файлд).
func Parse(raw []byte) (*Index, error) {
	var ix Index
	if err := json.Unmarshal(raw, &ix); err != nil {
		return nil, fmt.Errorf("index JSON: %w", err)
	}
	if err := ix.Validate(); err != nil {
		return nil, err
	}
	return &ix, nil
}

// LoadFile — локал index файл (registry-гүй fallback; гарын үсэг шаардахгүй —
// дистрибуцийн эзний файл).
func LoadFile(path string) (*Index, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(raw)
}

const maxIndexBytes = 4 << 20

// Fetch — URL-аас index + .sig татаж, гарын үсэг + replay (generated_at
// кэшээс хуучин биш, ирээдүйд биш) шалгаад cacheDir-т хадгална. Татаж
// чадахгүй бол сүүлийн сайн кэшээ буцаана (offline-д ч ажиллана).
func Fetch(ctx context.Context, url string, keys []ed25519.PublicKey, cacheDir string) (*Index, error) {
	if len(keys) == 0 {
		return nil, errors.New("registry: нийтийн түлхүүр байхгүй (REGISTRY_KEYS)")
	}
	cachedRaw, cachedSig, _ := readCache(cacheDir)
	var cached *Index
	if cachedRaw != nil && Verify(keys, cachedRaw, cachedSig) == nil {
		cached, _ = Parse(cachedRaw)
	}
	raw, sig, err := download(ctx, url, cacheDir)
	if err != nil {
		if cached != nil {
			return cached, nil
		}
		return nil, err
	}
	if err := Verify(keys, raw, sig); err != nil {
		if cached != nil {
			return cached, fmt.Errorf("registry: %w — кэш ашиглав", err)
		}
		return nil, fmt.Errorf("registry: %w", err)
	}
	ix, err := Parse(raw)
	if err != nil {
		if cached != nil {
			return cached, err
		}
		return nil, err
	}
	if ix.GeneratedAt.After(time.Now().Add(time.Hour)) {
		return nil, errors.New("registry: generated_at ирээдүйд")
	}
	if cached != nil && ix.GeneratedAt.Before(cached.GeneratedAt) {
		return cached, errors.New("registry: хуучин index (replay) — кэш ашиглав")
	}
	_ = writeCache(cacheDir, raw, sig)
	return ix, nil
}

func download(ctx context.Context, url, cacheDir string) ([]byte, string, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	get := func(u string, etag string) ([]byte, string, int, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, "", 0, err
		}
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", 0, err
		}
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNotModified {
			return nil, etag, resp.StatusCode, nil
		}
		if resp.StatusCode != http.StatusOK {
			return nil, "", resp.StatusCode, fmt.Errorf("%s: HTTP %d", u, resp.StatusCode)
		}
		b, err := io.ReadAll(io.LimitReader(resp.Body, maxIndexBytes+1))
		if err != nil {
			return nil, "", 0, err
		}
		if len(b) > maxIndexBytes {
			return nil, "", 0, fmt.Errorf("%s: %d байтаас том", u, maxIndexBytes)
		}
		return b, resp.Header.Get("ETag"), resp.StatusCode, nil
	}
	etag := ""
	if cacheDir != "" {
		if b, err := os.ReadFile(filepath.Join(cacheDir, "etag")); err == nil {
			etag = strings.TrimSpace(string(b))
		}
	}
	raw, newETag, code, err := get(url, etag)
	if err != nil {
		return nil, "", err
	}
	if code == http.StatusNotModified {
		cr, cs, cerr := readCache(cacheDir)
		if cerr != nil {
			return nil, "", errors.New("304 гэвч кэш байхгүй")
		}
		return cr, cs, nil
	}
	sigB, _, _, err := get(url+".sig", "")
	if err != nil {
		return nil, "", err
	}
	if cacheDir != "" && newETag != "" {
		_ = os.MkdirAll(cacheDir, 0o755)
		_ = os.WriteFile(filepath.Join(cacheDir, "etag"), []byte(newETag), 0o644)
	}
	return raw, string(sigB), nil
}

func readCache(dir string) ([]byte, string, error) {
	if dir == "" {
		return nil, "", errors.New("no cache")
	}
	raw, err := os.ReadFile(filepath.Join(dir, "index.json"))
	if err != nil {
		return nil, "", err
	}
	sig, err := os.ReadFile(filepath.Join(dir, "index.json.sig"))
	if err != nil {
		return nil, "", err
	}
	return raw, string(sig), nil
}

func writeCache(dir string, raw []byte, sig string) error {
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "index.json"), raw, 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "index.json.sig"), []byte(sig), 0o644)
}
