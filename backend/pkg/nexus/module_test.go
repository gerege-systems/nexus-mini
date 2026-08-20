package nexus

import (
	"io/fs"
	"testing"

	"github.com/go-chi/chi/v5"
)

type fakeModule struct {
	id, short string
	perms     []PermissionDefinition
}

func (m fakeModule) ID() string                          { return m.id }
func (m fakeModule) ShortID() string                     { return m.short }
func (m fakeModule) Name() string                        { return "Fake" }
func (m fakeModule) Version() string                     { return "1.0.0" }
func (m fakeModule) Dependencies() []Dependency          { return nil }
func (m fakeModule) Permissions() []PermissionDefinition { return m.perms }
func (m fakeModule) Menus() []MenuDefinition             { return nil }
func (m fakeModule) Migrations() fs.FS                   { return nil }
func (m fakeModule) RegisterRoutes(chi.Router, Deps)     {}

func expectPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s: panic хүлээж байсан", name)
		}
	}()
	fn()
}

func TestRegisterValidation(t *testing.T) {
	defer func() { registry = nil }()

	// Зөв модуль бүртгэгдэнэ.
	Register(fakeModule{id: "mn.test.good", short: "good",
		perms: []PermissionDefinition{{Code: "good.read", Name: "x"}}})

	// Prefix зөрчил — модулийн short ID-гаар эхлээгүй permission.
	expectPanic(t, "prefix", func() {
		Register(fakeModule{id: "mn.test.bad", short: "bad",
			perms: []PermissionDefinition{{Code: "other.read", Name: "x"}}})
	})

	// Давхардсан ID.
	expectPanic(t, "duplicate", func() {
		Register(fakeModule{id: "mn.test.good", short: "good2"})
	})

	// Буруу short ID (том үсэг).
	expectPanic(t, "shortid", func() {
		Register(fakeModule{id: "mn.test.upper", short: "Upper"})
	})
}

func TestMenuLocalizedLabel(t *testing.T) {
	m := MenuDefinition{Label: "Төхөөрөмжүүд", Labels: map[string]string{"en": "Devices"}}
	if m.LocalizedLabel("en") != "Devices" {
		t.Fatal("en label буруу")
	}
	if m.LocalizedLabel("mn") != "Төхөөрөмжүүд" {
		t.Fatal("default label буруу")
	}
	if m.LocalizedLabel("fr") != "Төхөөрөмжүүд" {
		t.Fatal("fallback ажилласангүй")
	}
}
