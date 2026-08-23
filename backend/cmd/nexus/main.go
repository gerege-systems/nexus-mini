// nexus — дистрибуцийн эзний CLI: цөмийг fork хийлгүй, registry-ээс модуль
// нэмж/шинэчилж/хасна.
//
//	go run github.com/gerege-systems/nexus-mini/backend/cmd/nexus@latest init my-dist
//	go run github.com/gerege-systems/nexus-mini/backend/cmd/nexus@latest add inventory[@1.2.0]
//	... upgrade [inventory] [--approve]   · remove inventory · list
//
// Дистрибуцийн бүтэц (init үүсгэнэ):
//
//	backend/main.go        core.Main(modules()...) — маркер хооронд модулиуд
//	backend/go.mod         require nexus-mini/backend + модулиуд
//	frontend/              цөмийн portal хуулбар; modules/<id>/ui — модулийн UI (add хуулна)
//	frontend/modules.json  add/remove засна
//
// Registry: REGISTRY_URL / REGISTRY_KEYS env эсвэл -registry/-keys флаг;
// default gerege-systems/nexus-registry (GitHub raw). Манифестийн permission ӨРГӨСВӨЛ upgrade -approve шаардана.
package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/pkg/registry"
)

const coreModule = "github.com/gerege-systems/nexus-mini/backend"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "init":
		err = cmdInit(os.Args[2:])
	case "add":
		err = cmdAdd(os.Args[2:], false)
	case "upgrade":
		err = cmdAdd(os.Args[2:], true)
	case "remove":
		err = cmdRemove(os.Args[2:])
	case "list":
		err = cmdList(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		err = fmt.Errorf("үл мэдэх комманд %q", os.Args[1])
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "алдаа:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`nexus — nexus-mini дистрибуцийн хэрэгсэл (цөмийг fork хийхгүй)

  init <dir> [-core vX.Y.Z]     шинэ дистрибуц (backend/main.go, go.mod, frontend/, admin/, makefile, .env.example)
  add <short_id|go_module>[@v]  registry-ээс модуль нэмнэ: go get + main.go + frontend/modules.json + ui хуулбар
  upgrade [short_id] [-approve] шинэ хувилбар; permission ӨРГӨССӨН бол -approve шаардана
  remove <short_id>             модуль хасна
  list                          registry дэх аппууд ба дистрибуцийн төлөв

Флаг: -registry URL  -keys b64[,b64]   (эсвэл REGISTRY_URL / REGISTRY_KEYS env)
`)
}

// ─── registry ─────────────────────────────────────────────────────────

func regFlags(fs *flag.FlagSet) (*string, *string) {
	url := fs.String("registry", envOr("REGISTRY_URL", registry.DefaultURL), "registry index.json URL")
	keys := fs.String("keys", os.Getenv("REGISTRY_KEYS"), "нийтийн түлхүүр(үүд), base64")
	return url, keys
}

func loadRegistry(url, keys string) (*registry.Index, error) {
	if strings.HasPrefix(url, "file://") || filepath.IsAbs(url) {
		return registry.LoadFile(strings.TrimPrefix(url, "file://"))
	}
	if keys == "" && url == registry.DefaultURL {
		keys = strings.Join(registry.DefaultKeys, ",")
	}
	pubs, err := registry.ParseKeys(keys)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cache := filepath.Join(os.TempDir(), "nexus-registry-cache")
	ix, err := registry.Fetch(ctx, url, pubs, cache)
	if err != nil && ix == nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "анхаар:", err)
	}
	return ix, nil
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// ─── дистрибуцийн файлууд ─────────────────────────────────────────────

const (
	markImportsBegin = "// nexus:imports:begin"
	markImportsEnd   = "// nexus:imports:end"
	markModulesBegin = "// nexus:modules:begin"
	markModulesEnd   = "// nexus:modules:end"
)

const mainTemplate = `// Энэ дистрибуцийн оролт. Модулиудыг "nexus add/remove" маркер хооронд
// засна — гараар ч засаж болно.
package main

import (
	"github.com/gerege-systems/nexus-mini/backend/core"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	` + markImportsBegin + `
	` + markImportsEnd + `
)

func main() { core.Main(modules()...) }

func modules() []nexus.Module {
	return []nexus.Module{
		` + markModulesBegin + `
		` + markModulesEnd + `
	}
}
`

const makefileTemplate = `# nexus-mini дистрибуц — бүх команд Makefile-аар.
ENV_FLAG := $(if $(ENV_FILE),--env $(ENV_FILE),)
.PHONY: build migrate serve web admin manifest check add upgrade

build:
	cd backend && go build -o bin/nexus.new . && { [ -f bin/nexus ] && cp bin/nexus bin/nexus.prev || true; } && mv -f bin/nexus.new bin/nexus
migrate: build
	cd backend && ./bin/nexus migrate $(ENV_FLAG)
serve: build
	cd backend && ./bin/nexus serve $(ENV_FLAG)
manifest: build
	cd backend && ./bin/nexus manifest $(MOD)
web:
	cd frontend && pnpm dev
admin:
	cd admin && pnpm dev
check:
	cd backend && GOOS=linux GOARCH=amd64 go build ./... && go vet ./...
# Модуль: make add MOD=inventory · make upgrade MOD=inventory APPROVE=1
add:
	go run ` + coreModule + `/cmd/nexus@latest add $(MOD)
upgrade:
	go run ` + coreModule + `/cmd/nexus@latest upgrade $(MOD) $(if $(APPROVE),-approve,)
`

type dist struct{ root string }

func findDist() (*dist, error) {
	d, _ := os.Getwd()
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(d, "backend", "main.go")); err == nil {
			if _, err := os.Stat(filepath.Join(d, "frontend", "modules.json")); err == nil {
				return &dist{root: d}, nil
			}
		}
		d = filepath.Dir(d)
	}
	return nil, errors.New("дистрибуцийн язгуур олдсонгүй (backend/main.go + frontend/modules.json) — `nexus init` хийсэн үү?")
}

type modEntry struct {
	ShortID string `json:"short_id"`
	UI      string `json:"ui"`
}

func (d *dist) readModules() ([]modEntry, error) {
	var out []modEntry
	raw, err := os.ReadFile(filepath.Join(d.root, "frontend", "modules.json"))
	if err != nil {
		return nil, err
	}
	return out, json.Unmarshal(raw, &out)
}

func (d *dist) writeModules(list []modEntry) error {
	sort.Slice(list, func(i, j int) bool { return list[i].ShortID < list[j].ShortID })
	raw, _ := json.MarshalIndent(list, "", "  ")
	return os.WriteFile(filepath.Join(d.root, "frontend", "modules.json"), append(raw, '\n'), 0o644)
}

var aliasRe = regexp.MustCompile(`[^a-z0-9_]`)

func alias(shortID string) string { return "mod_" + aliasRe.ReplaceAllString(shortID, "_") }

// setMain — main.go-ийн маркер хоорондын мөрүүдийг (импорт + модуль) бүхэлд
// нь дахин бичнэ: одоогийн жагсаалтаас.
func (d *dist) setMain(mods map[string]string /* shortID → goModule */) error {
	p := filepath.Join(d.root, "backend", "main.go")
	raw, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	ids := make([]string, 0, len(mods))
	for id := range mods {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	var imps, news []string
	for _, id := range ids {
		imps = append(imps, fmt.Sprintf("\t%s %q", alias(id), mods[id]))
		news = append(news, fmt.Sprintf("\t\t%s.New(),", alias(id)))
	}
	s := string(raw)
	s, err = replaceBetween(s, markImportsBegin, markImportsEnd, "\t", imps)
	if err != nil {
		return err
	}
	s, err = replaceBetween(s, markModulesBegin, markModulesEnd, "\t\t", news)
	if err != nil {
		return err
	}
	return os.WriteFile(p, []byte(s), 0o644)
}

func replaceBetween(s, begin, end, indent string, lines []string) (string, error) {
	i := strings.Index(s, begin)
	j := strings.Index(s, end)
	if i < 0 || j < 0 || j < i {
		return "", fmt.Errorf("backend/main.go-д %s / %s маркер байхгүй", begin, end)
	}
	body := ""
	if len(lines) > 0 {
		body = strings.Join(lines, "\n") + "\n"
	}
	return s[:i+len(begin)] + "\n" + body + indent + s[j:], nil
}

// currentModules — main.go-ийн маркер доторх импортуудаас (alias → path).
func (d *dist) currentModules() (map[string]string, error) {
	raw, err := os.ReadFile(filepath.Join(d.root, "backend", "main.go"))
	if err != nil {
		return nil, err
	}
	s := string(raw)
	i, j := strings.Index(s, markImportsBegin), strings.Index(s, markImportsEnd)
	if i < 0 || j < 0 {
		return nil, errors.New("main.go маркер байхгүй")
	}
	out := map[string]string{}
	re := regexp.MustCompile(`mod_([a-z0-9_]+) "([^"]+)"`)
	for _, m := range re.FindAllStringSubmatch(s[i:j], -1) {
		out[m[1]] = m[2]
	}
	return out, nil
}

func run(dir string, name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Stdout, c.Stderr = os.Stdout, os.Stderr
	return c.Run()
}

func runOut(dir string, name string, args ...string) (string, error) {
	c := exec.Command(name, args...)
	c.Dir = dir
	c.Stderr = os.Stderr
	b, err := c.Output()
	return strings.TrimSpace(string(b)), err
}

// ─── init ─────────────────────────────────────────────────────────────

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	core := fs.String("core", "", "цөмийн хувилбар (vX.Y.Z); хоосон бол хамгийн сүүлийн tag")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 {
		return errors.New("init <dir>")
	}
	root := fs.Arg(0)
	if _, err := os.Stat(root); err == nil {
		return fmt.Errorf("%s аль хэдийн байна", root)
	}
	ver := *core
	if ver == "" {
		out, err := runOut(os.TempDir(), "go", "list", "-m", "-versions", coreModule)
		if err == nil {
			f := strings.Fields(out)
			if len(f) > 1 {
				ver = f[len(f)-1]
			}
		}
	}
	if ver == "" {
		return errors.New("цөмийн хувилбар олдсонгүй — -core vX.Y.Z өг")
	}
	fmt.Printf("цөм %s %s\n", coreModule, ver)
	if err := os.MkdirAll(filepath.Join(root, "backend"), 0o755); err != nil {
		return err
	}
	name := filepath.Base(root)
	gomod := fmt.Sprintf("module %s/backend\n\ngo 1.25\n\nrequire %s %s\n", name, coreModule, ver)
	if err := os.WriteFile(filepath.Join(root, "backend", "go.mod"), []byte(gomod), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "main.go"), []byte(mainTemplate), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "makefile"), []byte(makefileTemplate), 0o644); err != nil {
		return err
	}
	// Цөмийн frontend + .env.example + deploy/ SQL-ийг tag-ийн tarball-аас.
	if err := fetchCoreFiles(root, ver); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(root, "frontend", "modules.json"), []byte("[]\n"), 0o644); err != nil {
		return err
	}
	gi := "node_modules/\n.next/\n*.env\n!.env.example\nbackend/bin/\nfrontend/lib/modules/\nfrontend/lib/i18n.modules.ts\n.registry-cache/\n"
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gi), 0o644); err != nil {
		return err
	}
	if err := run(filepath.Join(root, "backend"), "go", "mod", "tidy"); err != nil {
		return err
	}
	fmt.Printf(`
дистрибуц бэлэн: %s
  1. cp .env.example backend/nexus-mini.env  (DB URL + ADMIN_*; DB role: deploy/01-roles.sql)
  2. nexus add <short_id>            — registry-ээс модуль
  3. make migrate && make serve && (cd frontend && pnpm install && pnpm dev)
`, root)
	return nil
}

// fetchCoreFiles — github tarball (tag backend/<ver>) → frontend/, .env.example, deploy/*.sql.
func fetchCoreFiles(root, ver string) error {
	url := fmt.Sprintf("https://github.com/gerege-systems/nexus-mini/archive/refs/tags/backend/%s.tar.gz", ver)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s: HTTP %d", url, resp.StatusCode)
	}
	gz, err := gzip.NewReader(resp.Body)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	n := 0
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		// nexus-mini-backend-v1.0.0/frontend/... → frontend/...
		rel := h.Name[strings.Index(h.Name, "/")+1:]
		if !(strings.HasPrefix(rel, "frontend/") || strings.HasPrefix(rel, "admin/") || rel == ".env.example" || strings.HasPrefix(rel, "deploy/") && strings.HasSuffix(rel, ".sql")) {
			continue
		}
		if strings.Contains(rel, "node_modules/") || strings.Contains(rel, "/.next") || strings.HasPrefix(rel, "frontend/app/(portal)/devices") || strings.HasPrefix(rel, "frontend/app/(portal)/organisation") {
			continue
		}
		dst := filepath.Join(root, filepath.FromSlash(rel))
		if !strings.HasPrefix(dst, filepath.Clean(root)+string(os.PathSeparator)) {
			return fmt.Errorf("tar зам аюултай: %s", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			_ = os.MkdirAll(dst, 0o755)
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(dst), 0o755)
			f, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
			n++
		}
	}
	fmt.Printf("цөмийн frontend + admin: %d файл\n", n)
	return nil
}

// ─── add / upgrade ────────────────────────────────────────────────────

func cmdAdd(args []string, upgrade bool) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	url, keys := regFlags(fs)
	approve := fs.Bool("approve", false, "permission өргөссөн шинэчлэлтийг зөвшөөрнө")
	if err := fs.Parse(args); err != nil {
		return err
	}
	d, err := findDist()
	if err != nil {
		return err
	}
	ix, err := loadRegistry(*url, *keys)
	if err != nil {
		return err
	}
	current, err := d.currentModules()
	if err != nil {
		return err
	}
	targets := fs.Args()
	if upgrade && len(targets) == 0 {
		for id := range current {
			targets = append(targets, id)
		}
		sort.Strings(targets)
	}
	if len(targets) == 0 {
		return errors.New("add <short_id|go_module>[@version]")
	}
	for _, t := range targets {
		spec, ver, _ := strings.Cut(t, "@")
		mf := findManifest(ix, spec)
		if mf == nil {
			return fmt.Errorf("registry-д %q олдсонгүй (nexus list)", spec)
		}
		if ver == "" {
			ver = mf.Version
		}
		if _, installed := current[mf.ShortID]; installed && !upgrade {
			return fmt.Errorf("%s аль хэдийн нэмэгдсэн — nexus upgrade %s", mf.ShortID, mf.ShortID)
		}
		if upgrade {
			if widened := permissionsWidened(d, mf); len(widened) > 0 && !*approve {
				return fmt.Errorf("%s: шинэ/өргөссөн permission %v — эрхийн өөрчлөлтийг хянаад -approve өг", mf.ShortID, widened)
			}
		}
		if mf.GoModule == coreModule {
			fmt.Printf("→ %s (%s) %s\n", mf.ShortID, mf.Name, mf.ImportPath())
		} else {
			fmt.Printf("→ %s (%s) %s@v%s\n", mf.ShortID, mf.Name, mf.GoModule, ver)
		}
		if mf.GoModule == coreModule {
			// Цөмийн дотоод модуль (apps/*) — цөмтэйгээ хамт ирдэг. `go get`
			// хийвэл цөмийг модулийн хувилбар руу БУУЛГАНА; зөвхөн импортлоно.
			fmt.Printf("   (цөмийн дотоод модуль — go get хэрэггүй)\n")
		} else if err := run(filepath.Join(d.root, "backend"), "go", "get", mf.GoModule+"@v"+ver); err != nil {
			return fmt.Errorf("go get: %w", err)
		}
		current[mf.ShortID] = mf.ImportPath()
		if err := d.setMain(current); err != nil {
			return err
		}
		if err := d.copyUI(mf); err != nil {
			return err
		}
	}
	if err := run(filepath.Join(d.root, "backend"), "go", "mod", "tidy"); err != nil {
		return err
	}
	if err := run(filepath.Join(d.root, "backend"), "go", "build", "./..."); err != nil {
		return fmt.Errorf("build: %w", err)
	}
	fmt.Println("ok — make migrate && make serve (portal: cd frontend && pnpm build)")
	return nil
}

func findManifest(ix *registry.Index, spec string) *registry.Manifest {
	for i := range ix.Apps {
		m := &ix.Apps[i]
		if m.ShortID == spec || m.GoModule == spec || m.ID == spec {
			return m
		}
	}
	return nil
}

// copyUI — модулийн ui/ хавтсыг (Go module cache-ээс) frontend/modules/<id>/ui
// руу хуулж, modules.json + манифестийн хуулбар (upgrade-ийн diff-д) бичнэ.
func (d *dist) copyUI(mf *registry.Manifest) error {
	pkgDir, err := runOut(filepath.Join(d.root, "backend"), "go", "list", "-f", "{{.Dir}}", mf.ImportPath())
	if err != nil {
		return fmt.Errorf("модулийн хавтас: %w", err)
	}
	dst := filepath.Join(d.root, "frontend", "modules", mf.ShortID)
	_ = os.RemoveAll(dst)
	src := filepath.Join(pkgDir, "ui")
	if st, err := os.Stat(src); err == nil && st.IsDir() {
		if err := copyTree(src, filepath.Join(dst, "ui")); err != nil {
			return err
		}
	} else {
		_ = os.MkdirAll(dst, 0o755)
	}
	raw, _ := json.MarshalIndent(mf, "", "  ")
	if err := os.WriteFile(filepath.Join(dst, "manifest.json"), append(raw, '\n'), 0o644); err != nil {
		return err
	}
	list, err := d.readModules()
	if err != nil {
		return err
	}
	found := false
	for i := range list {
		if list[i].ShortID == mf.ShortID {
			list[i].UI = "./modules/" + mf.ShortID + "/ui"
			found = true
		}
	}
	if !found {
		list = append(list, modEntry{ShortID: mf.ShortID, UI: "./modules/" + mf.ShortID + "/ui"})
	}
	return d.writeModules(list)
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlink хуулахгүй: %s", p)
		}
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
}

// permissionsWidened — хуучин манифест (frontend/modules/<id>/manifest.json)
// vs шинэ: шинэ код, own→all default, нэмэгдсэн DefaultRoles.
func permissionsWidened(d *dist, mf *registry.Manifest) []string {
	raw, err := os.ReadFile(filepath.Join(d.root, "frontend", "modules", mf.ShortID, "manifest.json"))
	if err != nil {
		return nil // өмнөх манифест байхгүй — харьцуулах юмгүй
	}
	var old registry.Manifest
	if json.Unmarshal(raw, &old) != nil {
		return nil
	}
	prev := map[string]registry.Permission{}
	for _, p := range old.Permissions {
		prev[p.Code] = p
	}
	var out []string
	for _, p := range mf.Permissions {
		o, ok := prev[p.Code]
		if !ok {
			out = append(out, p.Code+" (шинэ)")
			continue
		}
		oldRoles := map[string]bool{}
		for _, r := range o.DefaultRoles {
			oldRoles[r] = true
		}
		for _, r := range p.DefaultRoles {
			if !oldRoles[r] {
				out = append(out, p.Code+" → "+r)
			}
		}
	}
	return out
}

// ─── remove / list ────────────────────────────────────────────────────

func cmdRemove(args []string) error {
	if len(args) != 1 {
		return errors.New("remove <short_id>")
	}
	d, err := findDist()
	if err != nil {
		return err
	}
	current, err := d.currentModules()
	if err != nil {
		return err
	}
	if _, ok := current[args[0]]; !ok {
		return fmt.Errorf("%s нэмэгдээгүй байна", args[0])
	}
	delete(current, args[0])
	if err := d.setMain(current); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(d.root, "frontend", "modules", args[0]))
	list, err := d.readModules()
	if err != nil {
		return err
	}
	kept := list[:0]
	for _, m := range list {
		if m.ShortID != args[0] {
			kept = append(kept, m)
		}
	}
	if err := d.writeModules(kept); err != nil {
		return err
	}
	if err := run(filepath.Join(d.root, "backend"), "go", "mod", "tidy"); err != nil {
		return err
	}
	fmt.Printf("%s хасагдлаа (DB хүснэгт/оноолт хэвээр — миграц буцаахгүй)\n", args[0])
	return nil
}

func cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	url, keys := regFlags(fs)
	if err := fs.Parse(args); err != nil {
		return err
	}
	ix, err := loadRegistry(*url, *keys)
	if err != nil {
		return err
	}
	installed := map[string]string{}
	if d, err := findDist(); err == nil {
		installed, _ = d.currentModules()
	}
	fmt.Printf("registry: %s (%d апп, %s)\n", *url, len(ix.Apps), ix.GeneratedAt.Format("2006-01-02"))
	for _, m := range ix.Apps {
		state := ""
		if _, ok := installed[m.ShortID]; ok {
			state = "  [нэмэгдсэн]"
		}
		fmt.Printf("  %-14s %-8s %s — %s%s\n", m.ShortID, m.Version, m.Name, m.GoModule, state)
	}
	return nil
}
