// Package envfile — энгийн KEY=VALUE файл уншиж орчинд ачаална.
// `make migrate`/`serve`-ийн уншдаг (nexus-mini.env) nexus-mini.env-ийг serve/migrate/admin
// коммандууд автоматаар олж уншдаг.
package envfile

import (
	"os"
	"strings"
)

// Load нь файлын мөр бүрийг KEY=VALUE гэж уншаад, орчинд ХАРААХАН
// байхгүй түлхүүрүүдийг тохируулна (орчны хувьсагч файлаас дээгүүр).
// Файл байхгүй бол чимээгүй алгасна.
func Load(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"`)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
	return nil
}
