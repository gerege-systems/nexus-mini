package apps

import (
	"strings"
	"testing"
)

// All — бинарид орох модулиудын жагсаалт: давхардалгүй, ID/ShortID зөв.
func TestAllModulesUnique(t *testing.T) {
	mods := All()
	if len(mods) == 0 {
		t.Fatal("модуль алга")
	}
	ids, shorts := map[string]bool{}, map[string]bool{}
	for _, m := range mods {
		if ids[m.ID()] || shorts[m.ShortID()] {
			t.Fatalf("давхардсан: %s / %s", m.ID(), m.ShortID())
		}
		ids[m.ID()], shorts[m.ShortID()] = true, true
		if !strings.HasPrefix(m.ID(), "mn.") || m.Version() == "" {
			t.Errorf("%s: ID/Version буруу", m.ID())
		}
	}
}
