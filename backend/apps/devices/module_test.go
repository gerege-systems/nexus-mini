package devices

// Модулийн ГЭРЭЭ: SDK-ийн дүрмүүдийг хангасан эсэх (Register-ийн шалгалтууд
// бинари асахад panic хийдэг тул энд урьдчилж барина).

import (
	"io/fs"
	"strings"
	"testing"
)

func TestModuleContract(t *testing.T) {
	m := New()
	if m.ID() == "" || !strings.Contains(m.ID(), ".") {
		t.Fatalf("ID = %q (reverse-DNS байх ёстой)", m.ID())
	}
	if m.ShortID() != "devices" || m.Name() == "" || m.Version() != "1.0.0" {
		t.Fatalf("ShortID/Name/Version = %q %q %q", m.ShortID(), m.Name(), m.Version())
	}
	if m.Description() == "" || m.Publisher() == "" {
		t.Fatal("Description/Publisher хоосон (registry манифестад ордог)")
	}
	if len(m.Dependencies()) != 0 {
		t.Fatalf("Dependencies = %v", m.Dependencies())
	}
	// Permission бүр ShortID-ийн prefix-тэй.
	perms := m.Permissions()
	if len(perms) == 0 {
		t.Fatal("permission алга")
	}
	for _, p := range perms {
		if !strings.HasPrefix(p.Code, m.ShortID()+".") {
			t.Errorf("permission %q нь prefix-гүй", p.Code)
		}
		if p.Name == "" {
			t.Errorf("%s: Name хоосон", p.Code)
		}
		for _, dr := range p.DefaultRoles {
			if strings.HasSuffix(dr, ":own") && !p.OwnScope {
				t.Errorf("%s: OwnScope=false атал %q", p.Code, dr)
			}
		}
	}
	// Цэсний зам ShortID-тай таарна.
	for _, mn := range m.Menus() {
		if mn.Path != "/"+m.ShortID() && !strings.HasPrefix(mn.Path, "/"+m.ShortID()+"/") {
			t.Errorf("цэсний зам %q буруу", mn.Path)
		}
		if mn.ID == "" || mn.Label == "" || mn.Icon == "" {
			t.Errorf("цэс дутуу: %+v", mn)
		}
		if mn.Labels["en"] == "" {
			t.Errorf("%s: en орчуулга алга", mn.ID)
		}
	}
	// Миграц embed хийгдсэн, goose форматтай.
	files, err := fs.Glob(m.Migrations(), "migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("миграц алга: %v %v", files, err)
	}
	for _, f := range files {
		raw, err := fs.ReadFile(m.Migrations(), f)
		if err != nil {
			t.Fatal(err)
		}
		src := string(raw)
		for _, want := range []string{"-- +goose Up", "-- +goose Down"} {
			if !strings.Contains(src, want) {
				t.Errorf("%s: %q алга", f, want)
			}
		}
		// Хүснэгт үүсгэдэг миграц бүр RLS + tenant_id-тай байх ёстой.
		if strings.Contains(src, "CREATE TABLE") {
			for _, want := range []string{"ENABLE ROW LEVEL SECURITY", "tenant_id", "GRANT"} {
				if !strings.Contains(src, want) {
					t.Errorf("%s: %q алга", f, want)
				}
			}
		}
		if strings.Contains(src, " text") && !strings.Contains(src, "::text") {
			t.Errorf("%s: задгай text багана байна", f)
		}
	}
}
