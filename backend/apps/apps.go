// Package apps — энэ бинарид компиллогдох модулиудын жагсаалт.
// `nexus-mini add` CLI (үе 2) энэ файлд мөр нэмдэг. Модулиуд internal
// биш тул гадны репо импортолж, хооронд нь хамаарч болно; цөм рүү
// (internal/core) хүрэхийг Go-гийн internal дүрэм + make check хориглоно.
package apps

import (
	"github.com/gerege-systems/nexus-mini/backend/apps/devices"
	"github.com/gerege-systems/nexus-mini/backend/apps/organisation"
	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
)

// All — cmd/nexus-mini (core.Main) дуудна (миграц нь модулиудын
// өөрсдийн migration FS-ийг мэдэх ёстой).
// All — энэ репогийн дистрибуцид орох модулиуд. Шинэ модуль = нэг мөр.
func All() []nexus.Module {
	return []nexus.Module{
		devices.New(),
		organisation.New(),
	}
}
