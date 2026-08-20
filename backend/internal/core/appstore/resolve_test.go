package appstore

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

type fakeMod struct {
	id   string
	deps []nexus.Dependency
}

func (m fakeMod) ID() string                                { return m.id }
func (m fakeMod) ShortID() string                           { return strings.ReplaceAll(m.id, ".", "_") }
func (m fakeMod) Name() string                              { return m.id }
func (m fakeMod) Version() string                           { return "1.0.0" }
func (m fakeMod) Dependencies() []nexus.Dependency          { return m.deps }
func (m fakeMod) Permissions() []nexus.PermissionDefinition { return nil }
func (m fakeMod) Menus() []nexus.MenuDefinition             { return nil }
func (m fakeMod) Migrations() fs.FS                         { return nil }
func (m fakeMod) RegisterRoutes(chi.Router, nexus.Deps)     {}

func modmap(ms ...nexus.Module) map[string]nexus.Module {
	out := map[string]nexus.Module{}
	for _, m := range ms {
		out[m.ID()] = m
	}
	return out
}

func ids(ms []nexus.Module) []string {
	var out []string
	for _, m := range ms {
		out = append(out, m.ID())
	}
	return out
}

func TestResolveOrder(t *testing.T) {
	// c → b → a гинж: a эхэлж суух ёстой.
	a := fakeMod{id: "t.a"}
	b := fakeMod{id: "t.b", deps: []nexus.Dependency{{ID: "t.a"}}}
	c := fakeMod{id: "t.c", deps: []nexus.Dependency{{ID: "t.b"}}}
	order, err := ResolveOrder(modmap(a, b, c), c)
	if err != nil {
		t.Fatal(err)
	}
	got := ids(order)
	want := []string{"t.a", "t.b", "t.c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("дараалал буруу: %v", got)
		}
	}
}

func TestResolveOrderDiamond(t *testing.T) {
	// d → {b, c} → a: a нэг л удаа, хамгийн түрүүнд.
	a := fakeMod{id: "t.a"}
	b := fakeMod{id: "t.b", deps: []nexus.Dependency{{ID: "t.a"}}}
	c := fakeMod{id: "t.c", deps: []nexus.Dependency{{ID: "t.a"}}}
	d := fakeMod{id: "t.d", deps: []nexus.Dependency{{ID: "t.b"}, {ID: "t.c"}}}
	order, err := ResolveOrder(modmap(a, b, c, d), d)
	if err != nil {
		t.Fatal(err)
	}
	if len(order) != 4 || order[0].ID() != "t.a" || order[3].ID() != "t.d" {
		t.Fatalf("diamond дараалал буруу: %v", ids(order))
	}
}

func TestResolveOrderCycle(t *testing.T) {
	a := fakeMod{id: "t.a", deps: []nexus.Dependency{{ID: "t.b"}}}
	b := fakeMod{id: "t.b", deps: []nexus.Dependency{{ID: "t.a"}}}
	if _, err := ResolveOrder(modmap(a, b), a); err == nil {
		t.Fatal("мөчлөг илрээгүй")
	}
}

func TestResolveOrderMissingDep(t *testing.T) {
	a := fakeMod{id: "t.a", deps: []nexus.Dependency{{ID: "t.missing"}}}
	if _, err := ResolveOrder(modmap(a), a); err == nil {
		t.Fatal("дутуу хамаарал илрээгүй")
	}
}
