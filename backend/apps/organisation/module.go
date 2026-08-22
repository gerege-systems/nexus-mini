// Package organisation — Байгууллагын бүтэц: хэлтэс/нэгжийн мод, ажилтны
// хэлтэс + албан тушаал. Цөм биш, модуль: байгууллага бүр хэрэгтэй бол
// store-оос суулгана (OGN ч үүнийг эцэст нь апп болгосон).
//
//	module.go        — гэрээ: ID, permission, цэс, миграц, route wiring
//	types.go         — хүсэлт/хариултын бүтцүүд + validation
//	departments.go   — хэлтсийн handler-ууд
//	people.go        — ажилтны handler-ууд
package organisation

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

func (m *Module) ID() string      { return "mn.gerege.nexus_mini.organisation" }
func (m *Module) ShortID() string { return "organisation" }
func (m *Module) Name() string    { return "Байгууллагын бүтэц" }
func (m *Module) Version() string { return "1.0.0" }

func (m *Module) Description() string {
	return "Хэлтэс, нэгжийн мод (дээд нэгж, менежер) болон ажилтан бүрийн хэлтэс, албан тушаал. Байгууллагын дотоод бүтцээ зурах модуль."
}
func (m *Module) Publisher() string { return "gerege-systems" }

func (m *Module) Dependencies() []nexus.Dependency { return nil }

func (m *Module) Permissions() []nexus.PermissionDefinition {
	return []nexus.PermissionDefinition{
		{
			Code:         "organisation.read",
			Name:         "Бүтэц харах",
			Description:  "Хэлтэс, нэгж, ажилтнуудын байршлыг харах",
			DefaultRoles: []string{"manager", "user"},
		},
		{
			Code:         "organisation.manage",
			Name:         "Бүтэц удирдах",
			Description:  "Хэлтэс үүсгэх/засах, ажилтны хэлтэс, албан тушаал тохируулах",
			DefaultRoles: []string{"manager"},
		},
	}
}

func (m *Module) Menus() []nexus.MenuDefinition {
	return []nexus.MenuDefinition{
		{ID: "organisation.departments", Label: "Хэлтэс, нэгж", Labels: map[string]string{"en": "Departments"},
			Path: "/organisation/departments", Icon: "building", Order: 10},
		{ID: "organisation.people", Label: "Ажилтнууд", Labels: map[string]string{"en": "People"},
			Path: "/organisation/people", Icon: "users", Order: 20},
	}
}

func (m *Module) Migrations() fs.FS { return migrations }

func (m *Module) RegisterRoutes(r chi.Router, deps nexus.Deps) {
	h := &handler{deps: deps}
	read := nexus.RequirePermission(deps.Perms, "organisation.read")
	manage := nexus.RequirePermission(deps.Perms, "organisation.manage")

	r.With(read).Get("/departments", h.listDepartments)
	r.With(manage).Post("/departments", h.createDepartment)
	r.With(manage).Put("/departments/{id}", h.updateDepartment)
	r.With(manage).Delete("/departments/{id}", h.deleteDepartment)

	r.With(read).Get("/people", h.listPeople)
	r.With(manage).Put("/people/{membership_id}", h.updatePosition)
}
