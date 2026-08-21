# Модуль хөгжүүлэх гарын авлага

Модуль бол `pkg/nexus.Module` interface-ийг хэрэгжүүлсэн Go package.
Хамгийн сайн заавар бол ажиллаж байгаа жишээ —
[`backend/apps/devices`](../backend/apps/devices/module.go)-ийг
нээгээд зэрэгцүүлж уншаарай.

## Модуль юу хийдэг, платформ юу хийдэг вэ

| Модуль | Платформ |
|---|---|
| Permission-оо тунхаглана | Tenant тусгаарлалт (RLS) |
| Цэсээ зарлана | Нэвтрэлт, session |
| Route-уудаа бүртгэнэ | Суулгалт, хамаарлын шийдэл |
| Өөрийн хүснэгт, миграц | RBAC default оноолт, шалгалт |
| Бизнес логик | Audit гинж, app store |

## Файлын бүтэц

Жижиг модуль нэг файлаас эхэлж болно; өсөхөөрөө ингэж хуваана
(devices нь яг энэ хуваарийн амьд жишээ):

```
backend/apps/<нэр>/           (эсвэл өөрийн репо — доорх «Өөрийн дистрибуц»)
  module.go              модулийн ГЭРЭЭ: ID, permission, цэс, миграц,
                         route↔permission холболт — бүгд нэг дор
  types.go               хүсэлт/хариултын struct + validation
  handlers.go            HTTP handler-ууд (нэг resource = нэг файл)
  reports_handlers.go    хоёр дахь resource нэмэгдвэл тусдаа файл
  reports_types.go
  migrations/            модулийн goose миграцууд
  ui/
    pages/               portal хуудсууд → app/(portal)/<нэр>/ руу хуулагдана
    i18n.ts              модулийн толь (en: {...}) → цөмийн толинд нэгдэнэ
```

`organisation` модуль нь олон resource-тэй хувилбарын жишээ
(`departments.go`, `people.go`, `ui/pages/departments/`, `ui/pages/people/`).

Зарчим: **route бүр аль permission-ээр хамгаалагдаж байгаа нь module.go-д
нэг дор харагдана**; handler файлууд зөвхөн бизнес логик агуулна. SQL нь
handler дотроо explicit байхад mini-д хангалттай — query олон газар давхардаж
эхэлбэл л store.go гэж тусгаарлана.

## Алхамууд

### 1. Package үүсгэх

`backend/apps/<нэр>/module.go` (өөр репод бол өөрийн module path,
`pkg/nexus`-аас л хамаарна):

```go
type Module struct{}

func (m *Module) ID() string      { return "mn.танай.<нэр>" } // reverse-DNS, глобал давтагдашгүй
func (m *Module) ShortID() string { return "<нэр>" }          // permission prefix + URL зам
func (m *Module) Name() string    { return "Хүний нэр" }
func (m *Module) Version() string { return "1.0.0" }
```

### 2. Permission тунхаглах

```go
func (m *Module) Permissions() []nexus.PermissionDefinition {
    return []nexus.PermissionDefinition{
        {Code: "<нэр>.read", Name: "...", DefaultRoles: []string{"manager", "user"}},
        {Code: "<нэр>.manage", Name: "...", OwnScope: true,
         DefaultRoles: []string{"manager", "user:own"}},
    }
}
```

Дүрмүүд (зөрчвөл бинари асахгүй):

- Код заавал `<ShortID>.`-ээр эхэлнэ — өөр модулийн эрхийг булааж чадахгүй
- `DefaultRoles` нь суулгах үед хэн авахыг **тунхагладаг** (suffix-ийн ид
  шид байхгүй): `admin` үргэлж бүгдийг авна, жагсаалтад бичсэн нь нэмж авна,
  `"user:own"` нь тухайн role-д зөвхөн өөрийн мөрийн эрх өгнө
- `DefaultRoles` хоосон = зөвхөн admin (аюулгүй default)

### 3. Миграц

`migrations/00001_<нэр>.sql` — goose формат, embed:

```go
//go:embed migrations/*.sql
var migrations embed.FS
func (m *Module) Migrations() fs.FS { return migrations }
```

Хүснэгтийн дүрмүүд:

- `tenant_id uuid NOT NULL` + RLS policy (`app_tenant_id()`) — жишээг
  devices-ээс хуул
- `OwnScope` ашиглах бол `created_by uuid` багана заавал
- Бүх string баганад урттай хязгаар (varchar(n)) — задгай text хориотой
- Төгсгөлд нь `GRANT ... TO nexus_app, nexus_admin`

Модуль бүр өөрийн goose хүснэгттэй (`goose_<shortid>`) тул цөм болон бусад
модультай мөргөлдөхгүй.

### 4. Route-ууд

```go
func (m *Module) RegisterRoutes(r chi.Router, deps nexus.Deps) {
    h := &handler{deps: deps}
    r.With(nexus.RequirePermission(deps.Perms, "<нэр>.read")).Get("/", h.list)
    r.With(nexus.RequirePermission(deps.Perms, "<нэр>.manage")).Post("/", h.create)
}
```

Танд өгөгдөх `r` нь **аль хэдийн хамгаалагдсан**: `/api/apps/<ShortID>/`
дор байрладаг, нэвтрээгүй хүн 401, апп суулгаагүй tenant 403 авчихсан
байдаг. Та зөвхөн permission middleware-ээ нэмнэ.

Handler дотор:

- `nexus.TenantID(ctx)`, `nexus.UserID(ctx)` — хүсэлтийн identity
- `nexus.Scope(ctx)` — `RequirePermission`-ий шийдсэн scope; `ScopeOwn` бол
  query-дээ `created_by = <user>` шүүлт нэм (devices-ийн `ownFilter` жишээ)
- `deps.DB` — RLS context нь автоматаар тохирдог холболт. SQL-даа
  `tenant_id = $1` гэж бас бичиж бай: RLS хамгаална, WHERE нь индекс
  ашиглуулна
- `deps.Audit.Record(ctx, "<нэр>.үйлдэл", объект, details)` — чухал
  үйлдлээ audit гинжид бич

### 5. Цэс

```go
func (m *Module) Menus() []nexus.MenuDefinition {
    return []nexus.MenuDefinition{{
        ID: "<нэр>.list", Label: "Монгол нэр",
        Labels: map[string]string{"en": "English"},
        Path: "/<нэр>", Icon: "device", Order: 10,
    }}
}
```

### 6. UI хуудас (portal)

UI нь **модулийн хавтаст** амьдарна: `ui/pages/` доторх файлууд portal
build үед `frontend/app/(portal)/<нэр>/` руу хуулагдана (`scripts/sync-modules.mjs`,
`pnpm build`/`dev`-ийн өмнө автоматаар; хуулагдсан хавтас git-д ордоггүй).
Цэсэндээ зарласан `Path` нь `/<нэр>/...` байх тул хуудасны зам нь
`ui/pages/page.tsx` → `/<нэр>`, `ui/pages/reports/page.tsx` → `/<нэр>/reports`.
Бэлэн загвар: [devices](../backend/apps/devices/ui/pages/page.tsx),
олон хуудастай нь [organisation](../backend/apps/organisation/ui/pages/).

Толь: `ui/i18n.ts` — түлхүүр нь монгол текст, утга нь орчуулга
(`{ en: { "Төхөөрөмжүүд": "Devices" } }`). Цөмийн `lib/i18n.tsx`-д **гар
хүрэхгүй** — ингэж байж цөмийг шинэчлэхэд мөргөлдөөн гарахгүй.

```tsx
"use client";

export default function NamePage() {
  const { me } = useShell();   // хэрэглэгч + permissions
  const { t } = useT();        // хэл (mn/en)
  const manage = me.permissions["<нэр>.manage"]; // undefined | "all" | "own"
  // api.get(`/api/apps/<нэр>/...`) — cookie автоматаар, 401 бол login руу
}
```

Дүрмүүд:

- **Эрхээр UI-гаа нуу**: `me.permissions["<нэр>.manage"]` байхгүй бол
  товчоо бүү харуул. Энэ нь UX — жинхэнэ хамгаалалт серверт (RequirePermission).
- «Өөрийн» scope-той хэрэглэгчид засах/устгах товчийг
  `created_by === me.user.id` үед л харуул (devices-ийн `canEdit` жишээ).
- Цэсэнд зарласан icon нэрээ `frontend/components/icons.tsx`-ийн map-д
  нэм (lucide icon).
- Бэлэн загварууд: `card / table / btn / field / badge / modal`
  (globals.css); амжилтад `toast(...)`, текстэд `t(...)` (шинэ текстээ
  модулийнхаа `ui/i18n.ts`-д нэмнэ).
- Хуудас бүр loading/хоосон/алдааны төлөвтэй байх (devices-ийн `empty`
  блок жишээ).

### 7. Бүртгэх ба асаах

Хоёр газар нэг нэг мөр:

```go
// backend/apps/apps.go — бинарид орох модулиуд
func All() []nexus.Module { return []nexus.Module{ devices.New(), <нэр>.New() } }
```
```json
// frontend/modules.json — portal-д орох UI
{ "short_id": "<нэр>", "ui": "../backend/apps/<нэр>/ui" }
```

`make migrate && make serve` (+ `make web`) — модуль чинь store-д гарч ирнэ.

### 8. Store-д нийтлэх

Энэ репогийн (nexus.*.com) store-д оруулах бол `catalog/apps.json`-д
бүртгэлээ нэмээд PR илгээнэ. Өөрийн store-той бол өөрийн каталог/регистр
(доор). Үе 2-т төв registry + `nexus-mini add` CLI ирнэ — тэр үед
`go_module` замаар тань шууд татна.

## Өөрийн дистрибуц — цөмийг fork хийхгүй

Та өөрийн компанид nexus-mini ашиглаж, өөрийн модулиудтай, өөрийн store-той,
өөрийн харилцагчидтай (tenant) платформ ажиллуулж болно. **Цөмийн репог
хуулбарлаж (fork) засахгүй** — хамаарал болгон ашиглана. Ингэж байж цөмийн
шинэчлэлтийг merge-гүй, мөргөлдөөнгүй авна.

Хоёр репо хангалттай:

```
your-company/nexus-inventory        ← модуль (энэ гарын авлагын дагуу)
  go.mod:  require github.com/gerege-systems/nexus-mini/backend v1.x.y
  module.go · handlers.go · migrations/ · ui/

your-company/nexus-dist             ← дистрибуц (цөм + сонгосон модулиуд)
  backend/
    go.mod:  require nexus-mini/backend v1.x.y, nexus-inventory v0.4.0
    main.go
  frontend/                         ← цөмийн frontend-ийн хуулбар + modules.json
  nexus-mini.env · makefile · deploy/
```

`backend/main.go` бүхэлдээ:

```go
package main

import (
	"github.com/gerege-systems/nexus-mini/backend/core"
	"your-company/nexus-inventory"
)

func main() { core.Main(inventory.New()) }
```

`core.Main` нь migrate/serve коммандууд, env файл, миграц, анхны админ,
permission sync, сервер — бүгдийг агуулна; та модулиудаа л өгнө. Цөмийн
`apps/devices`-ийг хүсвэл мөн импортолж нэмнэ, хүсэхгүй бол үгүй.

`frontend/modules.json`-д модулийнхаа `ui/` замыг заана. Модуль Go-гийн
cache-д байгаа бол замыг нь `cd backend && go list -m -f '{{.Dir}}'
your-company/nexus-inventory` гэж олно (үе 2-ын `nexus-mini add` үүнийг
автоматаар хийнэ).

**Цөмийг шинэчлэх:**

```bash
cd backend && go get github.com/gerege-systems/nexus-mini/backend@v1.5.0 && go mod tidy
cd ../frontend && git subtree pull --prefix frontend https://github.com/gerege-systems/nexus-mini main --squash
make check && make build
```

Backend — merge байхгүй, зөвхөн хувилбар. Frontend — цөмийн файлд та гар
хүрээгүй (модулийн UI `ui/`-д, толь `ui/i18n.ts`-д) тул subtree pull
мөргөлдөхгүй. Цөмд алдаа олбол өөр дээрээ засахгүй — upstream руу PR.

**SDK-ийн амлалт:** `pkg/nexus` (Module interface, Deps, RequirePermission,
Scope, web helpers) болон `core.Main` нь semver — `v1.x` дотор эвдэхгүй.
Цөмийн `internal/*` хэдийд ч өөрчлөгдөж болно; модуль түүнээс юу ч
импортолж чадахгүй (`make check` барина) тул танд хамаагүй.

**Таны харилцагчид:** таны instance дээр tenant болж бүртгүүлнэ, таны
store-оос суулгана; та платформ админ (`ADMIN_*` env) — түдгэлзүүлэх,
impersonation, бүх админ хэрэгсэл таных. Харилцагч модуль өөрөө татаж
чадахгүй (compile-time) — та бинаридаа оруулсан л бол гарч ирнэ.

## Тест

SQL parse/encode бүх логикт unit тест бичнэ. `make check` нь linux
cross-build + vet + test — push бүрийн өмнө заавал.
