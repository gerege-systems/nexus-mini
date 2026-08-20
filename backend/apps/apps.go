// Package apps — энэ бинарид компиллогдох модулиудын жагсаалт.
// `nexus-mini add` CLI (үе 2) энэ файлд мөр нэмдэг. Модулиуд internal
// биш тул гадны репо импортолж, хооронд нь хамаарч болно; цөм рүү
// (internal/core) хүрэхийг Go-гийн internal дүрэм + make check хориглоно.
package apps

import (
	"github.com/gerege-systems/nexus-mini/backend/apps/devices"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

// RegisterAll — cmd/nexus-mini дуудна (миграц нь модулиудын
// өөрсдийн migration FS-ийг мэдэх ёстой).
func RegisterAll() {
	nexus.Register(devices.New())
}
