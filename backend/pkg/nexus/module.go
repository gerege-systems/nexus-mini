// Package nexus нь платформ ба түүн дээр суух модулиудын хоорондох гэрээ.
//
// Модуль зөвхөн энэ package-аас хамаарна — internal/ доторх юуг ч импортлохгүй.
// Модуль гэдэг нь Module interface-ийг хэрэгжүүлээд Register-ээр бүртгүүлсэн
// Go package: permission-оо тунхаглаж, цэсээ зарлаад, урьдчилан хамгаалагдсан
// router дээрээ route-уудаа суулгана. Tenant тусгаарлалт, нэвтрэлт, суулгалт,
// RBAC оноолт, audit — бүгдийг платформ хийнэ.
package nexus

import (
	"context"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
)

// PermissionDefinition — модулийн зарладаг нэг эрх.
//
// Code нь заавал "<модулийн short ID>."-ээр эхэлнэ — Register үүнийг шалгаад
// зөрчилтэй бол бинари асахаас татгалзана. DefaultRoles нь суулгах үед энэ
// эрхийг default-оор авах role-уудын код ("manager", "user"); admin role
// үргэлж бүх эрхийг авдаг тул бичих шаардлагагүй. DefaultRoles хоосон бол
// зөвхөн admin — аюулгүй тал руугаа. OwnScope үнэн бол энэ эрхийг "зөвхөн
// өөрийн үүсгэсэн мөр" scope-той олгож болно (модуль created_by конвенц
// баримталсан байх ёстой).
type PermissionDefinition struct {
	Code         string   `json:"code"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	DefaultRoles []string `json:"default_roles,omitempty"`
	OwnScope     bool     `json:"own_scope,omitempty"`
}

// MenuDefinition — модулийн зарладаг цэсний нэг мөр. Labels нь locale-ийн
// override (ISO 639-1 түлхүүрээр); сервер хэрэглэгчийн хэлээр шийдэж өгдөг
// тул клиент серверийн контентыг орчуулах шаардлагагүй.
type MenuDefinition struct {
	ID     string            `json:"id"`
	Label  string            `json:"label"`
	Labels map[string]string `json:"-"`
	Path   string            `json:"path"`
	Icon   string            `json:"icon"`
	Order  int               `json:"order"`
}

// LocalizedLabel нь хүссэн хэлний label, байхгүй бол default Label.
func (m MenuDefinition) LocalizedLabel(locale string) string {
	if l, ok := m.Labels[locale]; ok && l != "" {
		return l
	}
	return m.Label
}

// Dependency — өөр модулиас хамаарах хамаарал. Version нь semver constraint
// ("^1.0.0" маягийн); хоосон бол дурын хувилбар.
type Dependency struct {
	ID      string `json:"id"`
	Version string `json:"version,omitempty"`
}

// Module — compile-time модуль бүрийн хэрэгжүүлэх гэрээ.
//
// ID нь глобал давтагдашгүй reverse-DNS ("io.gerege.devices"); ShortID нь
// permission prefix болон URL зам болдог богино нэр ("devices").
// RegisterRoutes-д өгөгдөх router нь /api/apps/<ShortID>/ дор аль хэдийн
// байрлаж, tenant auth + "энэ tenant-д апп суусан эсэх" gate давхарласан
// байдаг — модуль зөвхөн нарийн permission middleware-ээ нэмнэ.
// Migrations нь модулийн өөрийн goose миграцууд (embed.FS, "migrations/"
// дэд хавтаст); модуль бүр өөрийн goose хүснэгттэй тул цөмтэй мөргөлдөхгүй.
type Module interface {
	ID() string
	ShortID() string
	Name() string
	Version() string
	Dependencies() []Dependency
	Permissions() []PermissionDefinition
	Menus() []MenuDefinition
	Migrations() fs.FS
	RegisterRoutes(r chi.Router, deps Deps)
}

// Deps — платформоос модульд өгөх зүйлс. DB нь nexus_app pool-ийн
// хийсвэрлэл: гүйлгээ бүр RLS-ийн tenant context-той аль хэдийн тохирсон
// байдаг.
type Deps struct {
	DB    DB
	Perms PermissionStore
	Audit AuditRecorder
}

// AuditRecorder — модулиуд үйлдлээ платформын audit гинжид бичих гэрээ.
type AuditRecorder interface {
	Record(ctx context.Context, action, object string, details map[string]any)
}

var (
	registry   []Module
	shortIDRe  = regexp.MustCompile(`^[a-z][a-z0-9_]{1,31}$`)
	idRe       = regexp.MustCompile(`^[a-z][a-z0-9._]{2,127}$`)
	permCodeRe = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
)

// Register нь модулийг бинарид бүртгэнэ. Модулийн package-ийн init() эсвэл
// cmd/api-ийн модулийн жагсаалтаас дуудагдана. Зөрчил илэрвэл panic —
// эдгээр нь бүгд хөгжүүлэгчийн алдаа тул boot дээр унах нь runtime дээр
// чимээгүй цоорхой үлдээхээс дээр.
func Register(m Module) {
	if !idRe.MatchString(m.ID()) {
		panic(fmt.Sprintf("nexus: модулийн ID %q буруу форматтай", m.ID()))
	}
	if !shortIDRe.MatchString(m.ShortID()) {
		panic(fmt.Sprintf("nexus: модулийн ShortID %q буруу форматтай", m.ShortID()))
	}
	// Платформын нөөцөлсөн нэрс — permission prefix/URL зам булаахгүй.
	switch m.ShortID() {
	case "core", "api", "admin", "platform", "store":
		panic(fmt.Sprintf("nexus: ShortID %q нь платформын нөөцөлсөн нэр", m.ShortID()))
	}
	for _, r := range registry {
		if r.ID() == m.ID() || r.ShortID() == m.ShortID() {
			panic(fmt.Sprintf("nexus: %q модуль давхардаж бүртгэгдэв", m.ID()))
		}
	}
	prefix := m.ShortID() + "."
	for _, p := range m.Permissions() {
		if !permCodeRe.MatchString(p.Code) {
			panic(fmt.Sprintf("nexus: %s модулийн permission код %q буруу форматтай", m.ID(), p.Code))
		}
		if !strings.HasPrefix(p.Code, prefix) {
			panic(fmt.Sprintf("nexus: %s модулийн permission %q нь %q prefix-ээр эхлэх ёстой",
				m.ID(), p.Code, prefix))
		}
		for _, dr := range p.DefaultRoles {
			if strings.HasSuffix(dr, ":own") && !p.OwnScope {
				// Чимээгүй all болгочихвол fail-open — boot дээр унагана.
				panic(fmt.Sprintf("nexus: %s permission %q нь OwnScope=false атлаа %q default-той",
					m.ID(), p.Code, dr))
			}
		}
	}
	registry = append(registry, m)
}

// Registered нь бүртгэгдсэн модулиудыг ID-гаар эрэмбэлж буцаана.
func Registered() []Module {
	out := make([]Module, len(registry))
	copy(out, registry)
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}
