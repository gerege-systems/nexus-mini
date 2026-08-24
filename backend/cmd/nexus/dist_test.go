package main

// Дистрибуцийн CLI-ийн цэвэр логик (сүлжээ/Go toolchain шаардахгүй хэсэг):
// main.go-ийн маркер засвар, alias, modules.json, аюулгүй байдлын шалгалтууд.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/pkg/registry"
)

func newDist(t *testing.T) *dist {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "backend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "frontend"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "backend", "main.go"), []byte(mainTemplate), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "frontend", "modules.json"), []byte("[]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return &dist{root: root}
}

func TestSetMainAndCurrentModules(t *testing.T) {
	d := newDist(t)
	mods := map[string]string{
		"devices":     "github.com/gerege-systems/nexus-mini/backend/apps/devices",
		"inventory":   "bold.mn/nexus-inventory",
		"deep_module": "example.com/x/deep/module",
	}
	if err := d.setMain(mods); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(d.root, "backend", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(raw)
	for id, path := range mods {
		if !strings.Contains(src, alias(id)+" \""+path+"\"") {
			t.Errorf("импорт алга: %s", id)
		}
		if !strings.Contains(src, alias(id)+".New(),") {
			t.Errorf("modules() мөр алга: %s", id)
		}
	}
	// Дахин уншихад ижил зураглал.
	got, err := d.currentModules()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(mods) {
		t.Fatalf("currentModules = %v", got)
	}
	for id, path := range mods {
		if got[id] != path {
			t.Errorf("%s = %q", id, got[id])
		}
	}
	// Хасалт — маркер хоорондоо цэвэрлэгдэнэ.
	delete(mods, "inventory")
	if err := d.setMain(mods); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(d.root, "backend", "main.go"))
	if strings.Contains(string(raw), "bold.mn/nexus-inventory") {
		t.Fatal("хасагдсан модуль main.go-д үлдсэн")
	}
	// Бүгдийг хасахад ч файл эвдрэхгүй (маркерууд үлдэнэ).
	if err := d.setMain(map[string]string{}); err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(filepath.Join(d.root, "backend", "main.go"))
	if !strings.Contains(string(raw), markImportsBegin) || !strings.Contains(string(raw), markModulesEnd) {
		t.Fatal("маркерууд алдагдсан")
	}
}

func TestSetMainWithoutMarkersFails(t *testing.T) {
	d := newDist(t)
	if err := os.WriteFile(filepath.Join(d.root, "backend", "main.go"), []byte("package main\nfunc main(){}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := d.setMain(map[string]string{"x": "example.com/x"}); err == nil {
		t.Fatal("маркергүй main.go дээр алдаа гарсангүй")
	}
}

func TestModulesJSONRoundTrip(t *testing.T) {
	d := newDist(t)
	if err := d.writeModules([]modEntry{{ShortID: "b", UI: "./modules/b/ui"}, {ShortID: "a", UI: "./modules/a/ui"}}); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(filepath.Join(d.root, "frontend", "modules.json"))
	var list []modEntry
	if err := json.Unmarshal(raw, &list); err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].ShortID != "a" {
		t.Fatalf("эрэмбэлэгдээгүй: %v", list)
	}
	got, err := d.readModules()
	if err != nil || len(got) != 2 {
		t.Fatalf("readModules = %v %v", got, err)
	}
}

func TestAliasIsSafeIdentifier(t *testing.T) {
	for _, id := range []string{"devices", "my_app", "app9"} {
		a := alias(id)
		if !strings.HasPrefix(a, "mod_") || strings.ContainsAny(a, "-. /") {
			t.Errorf("alias(%q) = %q", id, a)
		}
	}
	// Тусгай тэмдэгт цэвэрлэгдэнэ (Go identifier эвдэхгүй).
	if a := alias("a-b.c"); strings.ContainsAny(a, "-.") {
		t.Errorf("alias цэвэрлээгүй: %q", a)
	}
}

func TestPermissionsWidened(t *testing.T) {
	d := newDist(t)
	dir := filepath.Join(d.root, "frontend", "modules", "inv")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	old := `{"id":"mn.a.inv","short_id":"inv","name":"Inv","version":"1.0.0","go_module":"example.com/m",
	 "permissions":[{"code":"inv.read","name":"r","default_roles":["user"]},{"code":"inv.manage","name":"m"}]}`
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	same := parseManifest(t, old)
	if got := permissionsWidened(d, &same); len(got) != 0 {
		t.Fatalf("өөрчлөлтгүй хувилбар өргөссөн гэв: %v", got)
	}
	// Шинэ permission.
	added := parseManifest(t, `{"id":"mn.a.inv","short_id":"inv","name":"Inv","version":"1.1.0","go_module":"example.com/m",
	 "permissions":[{"code":"inv.read","name":"r","default_roles":["user"]},{"code":"inv.manage","name":"m"},{"code":"inv.export","name":"e"}]}`)
	if got := permissionsWidened(d, &added); len(got) != 1 || !strings.Contains(got[0], "inv.export") {
		t.Fatalf("шинэ permission илрээгүй: %v", got)
	}
	// Өргөссөн DefaultRoles.
	wider := parseManifest(t, `{"id":"mn.a.inv","short_id":"inv","name":"Inv","version":"1.1.0","go_module":"example.com/m",
	 "permissions":[{"code":"inv.read","name":"r","default_roles":["user","manager"]},{"code":"inv.manage","name":"m"}]}`)
	if got := permissionsWidened(d, &wider); len(got) != 1 || !strings.Contains(got[0], "manager") {
		t.Fatalf("өргөссөн DefaultRoles илрээгүй: %v", got)
	}
	// Өмнөх манифест байхгүй бол харьцуулах юмгүй.
	d2 := newDist(t)
	if got := permissionsWidened(d2, &added); got != nil {
		t.Fatalf("манифестгүй үед = %v", got)
	}
}

func parseManifest(t *testing.T, raw string) registry.Manifest {
	t.Helper()
	var m registry.Manifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestFindDistRequiresBothFiles(t *testing.T) {
	root := t.TempDir()
	cwd, _ := os.Getwd()
	t.Cleanup(func() { _ = os.Chdir(cwd) })
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	if _, err := findDist(); err == nil {
		t.Fatal("хоосон хавтасд дистрибуц олдов")
	}
}
