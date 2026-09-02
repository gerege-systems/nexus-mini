# Хөгжүүлэгчийн гарын авлага — модуль, UI, дистрибуц, registry

> Вэб хувилбар (агуулгын бүх хэсэгтэй — эхлэх, интеграц OIDC/SSO, аюулгүй байдлын дүрэм, хувь нэмэр): https://nexus.craftzbay.com/developers · Интеграц: [04-integrations.md](04-integrations.md)

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
| Хувилбар + манифест (`make manifest`) | Registry, гарын үсэг, `nexus add/upgrade` |
| — | Түдгэлзүүлэлт / зөвхөн-унших, impersonation, lockout |

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

// Заавал биш — store/registry-ийн тайлбар кодоос (make manifest авдаг):
func (m *Module) Description() string { return "Юу хийдэг вэ…" }
func (m *Module) Publisher() string   { return "танай-байгууллага" }
```

`Version` нь semver; registry-д нийтлэх tag нь `v<Version>`. Permission
нэмсэн/өргөсгөсөн бол minor-оо өсгө — дистрибуц `nexus upgrade` хийхэд
`-approve` асуух шалтгаан нь энэ.

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
- `"role:own"` бичихийн тулд permission `OwnScope: true` байх ёстой — үгүй
  бол Register panic. Runtime-д ч: `own_scope=false` permission-д хэн ч «own»
  өгч чадахгүй (модуль шүүдэггүй тул чимээгүй бүрэн эрх болох байсан)
- Нөөцөлсөн ShortID: `core api admin platform store apps developers login
  signup dashboard members roles audit settings org`
- Шинэ хувилбарт permission нэмбэл цөм асахдаа суусан tenant бүрийн
  `admin`-д (+`DefaultRoles`) автоматаар оноодог (backfill); байгаа кодод
  хүрэхгүй — tenant-ийн гараар хассан оноолт сэргэхгүй

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
- Төгсгөлд нь `GRANT ... ON <table> TO nexus_app, nexus_admin` (функцэд
  автомат GRANT байхгүй — хэрэгтэй бол ил бич)
- Өөр хүснэгт рүү FK (`memberships`, өөрийн мод) заавал **same-tenant
  trigger**-тэй — FK шалгалт RLS-ийг давдаг тул uuid таамаглаж өөр
  tenant-ийн мөр холбож болдог. Загвар: `apps/organisation/migrations/00002_same_tenant.sql`
- Temp хүснэгт үүсгэх эрх апп role-д байхгүй (definer функц хамгаалалт);
  `users.password_hash` харагдахгүй; `auth_*` функцууд дуудагдахгүй — энэ
  бол зориуд

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
  үйлдлээ audit гинжид бич (impersonated session бол `impersonated_by`
  автоматаар хавсарна)
- `nexus.UUIDParam(w, r, "id")` / `nexus.IsUUID` — зам/биеийн id-г DB-д
  хүргэхээс өмнө (буруу бол 400, 500 биш); бүх string талбарын уртыг
  `valid()`-даа шалга
- Түдгэлзүүлсэн байгууллага → 403, зөвхөн-унших → бичих хүсэлт 503: платформ
  `RequireTenant`-д хийнэ, модуль юу ч мэдэх шаардлагагүй

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

`Path` заавал `/<ShortID>` эсвэл `/<ShortID>/...` — өөр зам Register panic
(portal-ийн middleware нийтийн замаас бусдыг хамгаалдаг, модуль түүнийг
тойрч чадахгүй). Icon нэр: `frontend/components/icons.tsx`-ийн map.

### 6. UI хуудас (portal)

UI нь **модулийн хавтаст** амьдарна: `ui/pages/` доторх файлууд portal
build үед `frontend/app/(portal)/<нэр>/` руу хуулагдана (`scripts/sync-modules.mjs`,
`pnpm build`/`dev`-ийн өмнө автоматаар; хуулагдсан хавтас git-д ордоггүй).
Цэсэндээ зарласан `Path` нь `/<нэр>/...` байх тул хуудасны зам нь
`ui/pages/page.tsx` → `/<нэр>`, `ui/pages/reports/page.tsx` → `/<нэр>/reports`.
Бэлэн загвар: [devices](../backend/apps/devices/ui/pages/page.tsx),
олон хуудастай нь [organisation](../backend/apps/organisation/ui/pages/).

Хуудасны нэр (browser tab): хуудас нь client component тул `metadata`-г
зэрэгцээ `layout.tsx`-аас өгнө — `export const metadata = { title: "Төхөөрөмжүүд" }`,
`export default function Layout({ children }) { return children; }`. Цөмийн
root layout нь `%s · nexus-mini` загвараар дүүргэнэ.

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
  нэм (утга нь lucide-ийн kebab нэр, ж: `building-2`).
- **UI бүхэлдээ `@gerege-systems/ui` дээр** (2026-08-31-ээс). Гараар CSS класс
  бичихгүй — `card / table / btn / field / badge / modal` зэрэг хуучин
  класснууд УСТСАН. Оронд нь сангийн компонентыг импортол:
  `Card`, `Table`/`TableHeader`/`TableRow`/`TableHead`/`TableCell`,
  `Button`, `IconButton`, `Input`, `Textarea`, `Select` (compound:
  `Select`+`SelectTrigger`+`SelectContent`+`SelectItem`), `Checkbox`,
  `Switch`, `Badge`, `Alert`, `Dialog` (+`DialogHeader/Content/Footer`),
  `ConfirmationDialog` (устгах гэх мэт эргэлт буцалтгүй үйлдэлд),
  `EmptyState`, `ErrorState`, `Spinner`, `Tooltip`.
- Дүрс `Icons.*`-ээс (`import { Icons } from '@gerege-systems/ui'`), тэнд
  байхгүй бол `import { Icon } from '@gerege-systems/ui/icon'` → `<Icon
  name="…" />`. **`lucide-react`-ийг шууд импортлохгүй.**
- Өнгө/зай/радиусыг гараар бүү бич — Tailwind токен класс
  (`text-foreground-muted`, `bg-background-muted`, `border-border`…).
- Амжилт/алдаанд `toast({ title, variant: 'success' | 'danger' })`
  (`@gerege-systems/ui`-ээс; хуучин `@/lib/toast` УСТСАН), текстэд `t(...)`
  (шинэ текстээ модулийнхаа `ui/i18n.ts`-д нэмнэ).
- Огноо `formatDate(...)` (`@gerege-systems/ui`) — `yyyy-MM-dd HH:mm`,
  Asia/Ulaanbaatar.
- Гарчигт `PageHead` (`@/components/states`) ашигла.
- Хуудас бүр loading/хоосон/алдааны төлөвтэй байх — `Spinner`,
  `EmptyState`, `Alert variant="danger"` (devices-ийн жишээ).
- `window.confirm` бүү ашигла — `ConfirmationDialog`.

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

### 8. Registry-д нийтлэх

Манифест **кодоос** үүснэ — гараар бичихгүй:

```bash
make manifest MOD=<нэр> > manifests/<нэр>.json
```

(`Description()`/`Publisher()` методыг модульдоо нэмбэл store-ийн тайлбар
мөн кодоос.) Дараа нь [gerege-systems/nexus-registry](https://github.com/gerege-systems/nexus-registry)
репод `manifests/<нэр>.json`-оо PR-аар илгээнэ; maintainer `index.json`-ийг
бүтээж Ed25519-ээр гарын үсэглэнэ. Модулийн код registry-д хадгалагдахгүй —
`go_module` зам + git tag (`v<version>`) тань байхад л хангалттай. Өөрийн
registry: репог хуулж, `nexus-registry keygen` → өөрийн URL + нийтийн
түлхүүрээ дистрибуцуудад `REGISTRY_URL`/`REGISTRY_KEYS`-ээр өгнө.

Registry-д орсон модулийг хэн ч `nexus add <нэр>` гэж дистрибуцдаа нэмнэ
(доор). Дистрибуцийн store хуудас registry-ийн бүх аппыг харуулж, бинарид
ороогүйд нь `nexus add` зааварчилгаа гаргана.

## Өөрийн дистрибуц — цөмийг fork хийхгүй

Та өөрийн компанид nexus-mini ашиглаж, өөрийн модулиудтай, өөрийн store-той,
өөрийн харилцагчидтай (tenant) платформ ажиллуулж болно. **Цөмийн репог
хуулбарлаж (fork) засахгүй** — хамаарал болгон ашиглана. Ингэж байж цөмийн
шинэчлэлтийг merge-гүй, мөргөлдөөнгүй авна.

`nexus` CLI бүгдийг хийнэ (`go run github.com/gerege-systems/nexus-mini/backend/cmd/nexus@latest …`):

```bash
nexus init my-dist              # backend/{go.mod,main.go}, frontend/ + admin/ (цөмийн хуулбар), makefile, .env.example, deploy/*.sql
cd my-dist
nexus add organisation          # registry → go get + main.go маркер + frontend/modules/organisation/ui + modules.json
nexus add inventory@0.4.0       # өөрийн модуль (registry-д нийтэлсэн)
nexus list · nexus upgrade [нэр] [-approve] · nexus remove нэр
make migrate && make serve && (cd frontend && pnpm install && pnpm build)
```

Үүссэн бүтэц:

```
my-dist/
  backend/main.go       core.Main(modules()...) — "// nexus:imports/modules:begin…end" маркер хооронд
  backend/go.mod        require nexus-mini/backend v1.x + модулиуд
  frontend/             цөмийн portal хуулбар; modules/<нэр>/{ui,manifest.json} — add хуулна (commit хийнэ)
  frontend/modules.json add/remove засна
```

`core.Main` нь migrate/serve/manifest коммандууд, env файл, миграц, анхны
админ, permission sync, сервер — бүгдийг агуулна. `upgrade` нь хуучин
манифест (`frontend/modules/<нэр>/manifest.json`) vs registry-ийн шинэ
манифестийг тулгаж, **permission шинээр нэмэгдсэн/өргөссөн** бол зогсоож
`-approve` шаардана — модулийн шинэчлэлт чимээгүй эрх авахгүй.

Registry: default gerege-systems/nexus-registry (GitHub raw) (түлхүүр цөмд); өөрийнх бол `REGISTRY_URL` +
`REGISTRY_KEYS` env (эсвэл `-registry/-keys` флаг). Бүх дистрибуц registry-д
байгаа аппуудыг store-доо харуулна; бинарид ороогүйд нь `nexus add` заавар.

**Цөмийг шинэчлэх:**

```bash
# backend — зөвхөн хувилбар
cd backend && go get github.com/gerege-systems/nexus-mini/backend@v1.5.0 && go mod tidy && cd ..
# frontend — цөмийн frontend-ийг tag-аас хуулж дарна; modules.json, modules/ таных хэвээр
git remote add upstream https://github.com/gerege-systems/nexus-mini   # нэг удаа
git fetch upstream --tags
git checkout backend/v1.5.0 -- frontend && git checkout HEAD -- frontend/modules.json frontend/modules
make check && make build
```

Backend — merge байхгүй. Frontend — цөмийн файлд та гар хүрээгүй (модулийн
UI `modules/<нэр>/ui`-д, толь `ui/i18n.ts`-д) тул `git checkout <tag> --
frontend` нь цэвэр дарж бичилт, мөргөлдөхгүй. Цөмд алдаа олбол өөр дээрээ засахгүй — upstream руу PR.

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
cross-build + vet + test + SDK-ийн хил (модуль `internal/*` импортолбол
унадаг) — push бүрийн өмнө заавал.
