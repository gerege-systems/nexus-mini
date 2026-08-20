// Package modules — энэ бинарид компиллогдох модулиудын жагсаалт.
// `nexus-mini add` CLI (үе 2) энэ файлд мөр нэмдэг.
package modules

import (
	"github.com/gerege-systems/nexus-mini/backend/internal/apps/devices"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

// RegisterAll — cmd/api ба cmd/migrate хоёул дуудна (миграц нь модулиудын
// өөрсдийн migration FS-ийг мэдэх ёстой).
func RegisterAll() {
	nexus.Register(devices.New())
}
