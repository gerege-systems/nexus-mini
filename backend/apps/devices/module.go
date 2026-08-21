// Package devices — Төхөөрөмжийн бүртгэл: SDK-г баталдаг жишээ модуль.
//
// Модуль хэрхэн бичихийн загвар болдог тул docs/03-module-guide.md-тэй
// хамт уншина. Файлын хуваарь нь мөн жишиг:
//
//	module.go    — модулийн ГЭРЭЭ: ID, permission, цэс, миграц, route wiring
//	types.go     — хүсэлт/хариултын бүтцүүд + validation
//	handlers.go  — HTTP handler-ууд (нэг resource = нэг файл; олон
//	               resource-тэй модуль reports_handlers.go гэх мэтээр нэмнэ)
//	migrations/  — модулийн өөрийн goose миграцууд
package devices

import (
	"embed"
	"io/fs"

	"github.com/gerege-systems/nexus-mini/backend/pkg/nexus"
	"github.com/go-chi/chi/v5"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) ID() string      { return "mn.gerege.nexus_mini.devices" }
func (m *Module) ShortID() string { return "devices" }
func (m *Module) Name() string    { return "Төхөөрөмжийн бүртгэл" }
func (m *Module) Version() string { return "1.0.0" }

func (m *Module) Dependencies() []nexus.Dependency { return nil }

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{
			Code:         "devices.read",
			Name:         "Төхөөрөмж харах",
			Description:  "Байгууллагын төхөөрөмжийн жагсаалтыг харах",
			DefaultRoles: []string{"manager", "user"},
			OwnScope:     true, // "own" өгвөл зөвхөн өөрийн бүртгэсэн мөрүүд
		},
		{
			Code:         "devices.manage",
			Name:         "Төхөөрөмж бүртгэх, засах",
			Description:  "Төхөөрөмж нэмэх, засах, устгах",
			OwnScope:     true,
			DefaultRoles: []string{"manager", "user:own"},
		},
	}
}

func (m *Module) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{
			ID:     "devices.list",
			Label:  "Төхөөрөмжүүд",
			Labels: map[string]string{"en": "Devices"},
			Path:   "/devices",
			Icon:   "device",
			Order:  10,
		},
	}
}

func (m *Module) Migrations() fs.FS { return migrations }

// RegisterRoutes — route бүрийг permission-тэй нь ЭНД, нэг дор холбоно:
// модулийн бүх "хэн юу хийж болох" нэг файлд харагдана. Handler-ийн бие
// нь handlers.go-д.
func (m *Module) RegisterRoutes(r chi.Router, deps nexus.Deps) {
	h := &handler{deps: deps}
	read := nexus.RequirePermission(deps.Perms, "devices.read")
	manage := nexus.RequirePermission(deps.Perms, "devices.manage")

	r.With(read).Get("/", h.list)
	r.With(manage).Post("/", h.create)
	r.With(manage).Put("/{id}", h.update)
	r.With(manage).Delete("/{id}", h.remove)
}
