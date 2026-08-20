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

### 6. Бүртгэх

`backend/apps/apps.go`-д нэг мөр:

```go
nexus.Register(<нэр>.New())
```

`make migrate && make api` — модуль чинь store-д гарч ирнэ.

### 7. Store-д нийтлэх

`catalog/apps.json`-д бүртгэлээ нэмээд PR илгээнэ (үе 2-т төв registry +
`nexus-mini add` CLI ирнэ — тэр үед go_module замаар тань шууд татна).

## Тест

SQL parse/encode бүх логикт unit тест бичнэ. `make check` нь linux
cross-build + vet + test — push бүрийн өмнө заавал.
