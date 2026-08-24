package main

// nexus CLI-ийн бүтэн урсгал ЛОКАЛ registry-тэй, сүлжээгүйгээр: list → add →
// (main.go + modules.json + UI хуулбар) → build → upgrade → remove.
// Цөмийн дотоод модуль тул `go get` алгасагдана (цөмийг буулгахгүй).

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gerege-systems/nexus-mini/backend/pkg/registry"
)

// localDist — бодит цөм рүү replace хийсэн дистрибуц (сүлжээгүй build).
func localDist(t *testing.T) *dist {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain алга")
	}
	backendDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	must(os.MkdirAll(filepath.Join(root, "backend"), 0o755))
	must(os.MkdirAll(filepath.Join(root, "frontend"), 0o755))
	gomod := "module testdist/backend\n\ngo 1.25\n\nrequire " + coreModule + " v1.2.0\n\nreplace " + coreModule + " => " + backendDir + "\n"
	must(os.WriteFile(filepath.Join(root, "backend", "go.mod"), []byte(gomod), 0o644))
	must(os.WriteFile(filepath.Join(root, "backend", "main.go"), []byte(mainTemplate), 0o644))
	must(os.WriteFile(filepath.Join(root, "frontend", "modules.json"), []byte("[]\n"), 0o644))
	// go.sum-ыг replace орлуулна; кэшнээс шийднэ.
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = filepath.Join(root, "backend")
	cmd.Env = append(os.Environ(), "GOFLAGS=-mod=mod", "GOPROXY=off")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("офлайн go mod tidy боломжгүй: %v\n%s", err, out)
	}
	return &dist{root: root}
}

// fileRegistry — гарын үсэггүй локал index (LoadFile зам).
func fileRegistry(t *testing.T, manifests ...registry.Manifest) string {
	t.Helper()
	ix := registry.Index{GeneratedAt: time.Now().UTC(), Apps: manifests}
	raw, err := json.MarshalIndent(ix, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(t.TempDir(), "index.json")
	if err := os.WriteFile(p, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func devicesManifest() registry.Manifest {
	return registry.Manifest{
		ID: "mn.gerege.nexus_mini.devices", ShortID: "devices", Name: "Төхөөрөмж", Version: "1.0.0",
		GoModule: coreModule, Import: coreModule + "/apps/devices", MinCore: "1.0.0",
		Permissions: []registry.Permission{{Code: "devices.read", Name: "харах", DefaultRoles: []string{"user"}}},
		Menus:       []registry.Menu{{ID: "devices.list", Label: "Төхөөрөмж", Path: "/devices", Icon: "device"}},
	}
}

func runCLI(t *testing.T, d *dist, args ...string) error {
	t.Helper()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(d.root); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOFLAGS", "-mod=mod")
	t.Setenv("GOPROXY", "off")
	defer func() { _ = os.Chdir(cwd) }()
	switch args[0] {
	case "add":
		return cmdAdd(args[1:], false)
	case "upgrade":
		return cmdAdd(args[1:], true)
	case "remove":
		return cmdRemove(args[1:])
	case "list":
		return cmdList(args[1:])
	}
	t.Fatalf("үл мэдэх комманд %q", args[0])
	return nil
}

func TestCLIAddUpgradeRemove(t *testing.T) {
	d := localDist(t)
	reg := fileRegistry(t, devicesManifest())

	// list — бүртгэлтэй апп харагдана.
	if err := runCLI(t, d, "list", "-registry", reg); err != nil {
		t.Fatalf("list: %v", err)
	}
	// add — цөмийн дотоод модуль тул go get алгасна.
	if err := runCLI(t, d, "add", "-registry", reg, "devices"); err != nil {
		t.Fatalf("add: %v", err)
	}
	// go.mod-ын цөмийн хувилбар ХЭВЭЭР (буулгаагүй).
	gomod, err := os.ReadFile(filepath.Join(d.root, "backend", "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(gomod), coreModule+" v1.2.0") {
		t.Fatalf("цөмийн хувилбар өөрчлөгдсөн:\n%s", gomod)
	}
	// main.go-д импорт + модуль.
	mainSrc, _ := os.ReadFile(filepath.Join(d.root, "backend", "main.go"))
	if !strings.Contains(string(mainSrc), coreModule+"/apps/devices") || !strings.Contains(string(mainSrc), "mod_devices.New()") {
		t.Fatalf("main.go:\n%s", mainSrc)
	}
	// modules.json + UI хуулбар + манифест.
	list, err := d.readModules()
	if err != nil || len(list) != 1 || list[0].ShortID != "devices" {
		t.Fatalf("modules.json = %v %v", list, err)
	}
	uiDir := filepath.Join(d.root, "frontend", "modules", "devices", "ui")
	if _, err := os.Stat(filepath.Join(uiDir, "i18n.ts")); err != nil {
		t.Fatalf("UI толь хуулагдаагүй: %v", err)
	}
	if _, err := os.Stat(filepath.Join(uiDir, "pages", "page.tsx")); err != nil {
		t.Fatalf("UI хуудас хуулагдаагүй: %v", err)
	}
	if _, err := os.Stat(filepath.Join(d.root, "frontend", "modules", "devices", "manifest.json")); err != nil {
		t.Fatalf("манифест хадгалагдаагүй: %v", err)
	}
	// Дахин add — татгалзана.
	if err := runCLI(t, d, "add", "-registry", reg, "devices"); err == nil {
		t.Fatal("давхар add амжилттай болов")
	}
	// Байхгүй апп.
	if err := runCLI(t, d, "add", "-registry", reg, "байхгүй"); err == nil {
		t.Fatal("байхгүй апп нэмэгдэв")
	}
	// upgrade — permission өөрчлөгдөөгүй тул зөвшөөрөл шаардахгүй.
	if err := runCLI(t, d, "upgrade", "-registry", reg, "devices"); err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	// permission ӨРГӨССӨН хувилбар — -approve-гүй бол зогсоно.
	wider := devicesManifest()
	wider.Version = "1.1.0"
	wider.Permissions = append(wider.Permissions, registry.Permission{Code: "devices.manage", Name: "удирдах", DefaultRoles: []string{"manager"}})
	reg2 := fileRegistry(t, wider)
	err = runCLI(t, d, "upgrade", "-registry", reg2, "devices")
	if err == nil || !strings.Contains(err.Error(), "approve") {
		t.Fatalf("permission өргөссөн upgrade = %v", err)
	}
	if err := runCLI(t, d, "upgrade", "-registry", reg2, "-approve", "devices"); err != nil {
		t.Fatalf("-approve-тэй upgrade: %v", err)
	}
	// remove — main.go, modules.json, UI цэвэрлэгдэнэ.
	if err := runCLI(t, d, "remove", "devices"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	mainSrc, _ = os.ReadFile(filepath.Join(d.root, "backend", "main.go"))
	if strings.Contains(string(mainSrc), "mod_devices") {
		t.Fatal("remove-ийн дараа main.go-д үлдсэн")
	}
	if list, _ := d.readModules(); len(list) != 0 {
		t.Fatalf("modules.json = %v", list)
	}
	if _, err := os.Stat(filepath.Join(d.root, "frontend", "modules", "devices")); !os.IsNotExist(err) {
		t.Fatal("UI хавтас устгагдаагүй")
	}
	// Байхгүй модулийг хасах.
	if err := runCLI(t, d, "remove", "devices"); err == nil {
		t.Fatal("байхгүй модуль хасагдав")
	}
}

func TestLoadRegistryFromFileAndBadKeys(t *testing.T) {
	reg := fileRegistry(t, devicesManifest())
	ix, err := loadRegistry(reg, "")
	if err != nil || len(ix.Apps) != 1 {
		t.Fatalf("файлаас = %v %v", ix, err)
	}
	if ix2, err := loadRegistry("file://"+reg, ""); err != nil || len(ix2.Apps) != 1 {
		t.Fatalf("file:// = %v %v", ix2, err)
	}
	// HTTP registry + буруу түлхүүр.
	if _, err := loadRegistry("https://example.invalid/index.json", "тийм-биш"); err == nil {
		t.Fatal("буруу түлхүүр хүлээн авагдав")
	}
	// findManifest — short_id, id, go_module-аар олдоно.
	m := devicesManifest()
	fix := &registry.Index{Apps: []registry.Manifest{m}}
	for _, spec := range []string{"devices", m.ID, m.GoModule} {
		if findManifest(fix, spec) == nil {
			t.Errorf("findManifest(%q) = nil", spec)
		}
	}
	if findManifest(fix, "байхгүй") != nil {
		t.Error("байхгүй апп олдов")
	}
	// envOr.
	t.Setenv("ТЕСТ_ENV", "")
	if envOr("ТЕСТ_ENV", "default") != "default" {
		t.Error("envOr default")
	}
	t.Setenv("ТЕСТ_ENV", "утга")
	if envOr("ТЕСТ_ENV", "default") != "утга" {
		t.Error("envOr env")
	}
}

func TestCopyTreeRejectsSymlink(t *testing.T) {
	src, dst := t.TempDir(), filepath.Join(t.TempDir(), "out")
	if err := os.WriteFile(filepath.Join(src, "a.tsx"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dst, "a.tsx")); err != nil {
		t.Fatal(err)
	}
	// Symlink — татгалзана.
	if err := os.Symlink("/etc/passwd", filepath.Join(src, "link")); err != nil {
		t.Skip("symlink дэмжигдэхгүй")
	}
	if err := copyTree(src, filepath.Join(t.TempDir(), "out2")); err == nil {
		t.Fatal("symlink хуулагдав")
	}
}

// TestCmdInitOffline — init-ийг ЛОКАЛ tarball сервертэй (сүлжээгүй) шалгана.
func TestCmdInitOffline(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go toolchain алга")
	}
	// Цөмийн frontend/admin/.env.example/deploy-ийг агуулсан tar.gz.
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	add := func(name, body string) {
		t.Helper()
		if err := tw.WriteHeader(&tar.Header{Name: "nexus-mini-1.2.0/" + name, Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	add("frontend/package.json", `{"name":"web"}`)
	add("frontend/lib/i18n.tsx", "// толь\n")
	add("admin/package.json", `{"name":"admin"}`)
	add(".env.example", "DATABASE_URL=\n")
	add("deploy/01-roles.sql", "-- roles\n")
	add("backend/main.go", "// орохгүй\n")            // backend хуулагдахгүй
	add("frontend/node_modules/x/index.js", "skip\n") // node_modules алгасна
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/backend/v1.2.0.tar.gz") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(buf.Bytes())
	}))
	defer srv.Close()
	old := archiveBase
	archiveBase = srv.URL
	defer func() { archiveBase = old }()

	root := filepath.Join(t.TempDir(), "my-dist")
	if err := cmdInit([]string{"-core", "v1.2.0", root}); err != nil {
		t.Fatalf("cmdInit: %v", err)
	}
	for _, want := range []string{"backend/go.mod", "backend/main.go", "makefile", ".gitignore",
		"frontend/package.json", "frontend/lib/i18n.tsx", "frontend/modules.json", "admin/package.json",
		".env.example", "deploy/01-roles.sql"} {
		if _, err := os.Stat(filepath.Join(root, want)); err != nil {
			t.Errorf("%s үүсээгүй", want)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "backend", "main.go")); err != nil {
		t.Fatal(err)
	}
	// Tarball доторх backend/main.go биш, ЗАГВАР main.go байх ёстой.
	mainSrc, _ := os.ReadFile(filepath.Join(root, "backend", "main.go"))
	if !strings.Contains(string(mainSrc), markModulesBegin) {
		t.Fatalf("main.go загвар биш:\n%s", mainSrc)
	}
	if _, err := os.Stat(filepath.Join(root, "frontend", "node_modules")); !os.IsNotExist(err) {
		t.Error("node_modules хуулагдав")
	}
	// go.mod-д цөм зөв хувилбартай.
	gomod, _ := os.ReadFile(filepath.Join(root, "backend", "go.mod"))
	if !strings.Contains(string(gomod), coreModule+" v1.2.0") {
		t.Fatalf("go.mod:\n%s", gomod)
	}
	// makefile-д add/upgrade/admin target-ууд.
	mk, _ := os.ReadFile(filepath.Join(root, "makefile"))
	for _, want := range []string{"migrate:", "serve:", "admin:", "add:", "upgrade:"} {
		if !strings.Contains(string(mk), want) {
			t.Errorf("makefile-д %q алга", want)
		}
	}
	// Байгаа хавтас дээр дахин init — татгалзана.
	if err := cmdInit([]string{"-core", "v1.2.0", root}); err == nil {
		t.Fatal("байгаа хавтас дээр init амжилттай болов")
	}
	// Хувилбаргүй, сүлжээгүй үед.
	archiveBase = "http://127.0.0.1:1"
	if err := cmdInit([]string{"-core", "v9.9.9", filepath.Join(t.TempDir(), "x")}); err == nil {
		t.Fatal("татагдахгүй tarball дээр амжилттай болов")
	}
	usage() // тусламжийн текст — panic хийхгүй
}
