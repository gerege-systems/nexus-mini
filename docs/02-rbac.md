# RBAC — Gerege Nexus-ийн суурь + засварууд

Gerege Nexus-ийн role бүтцийг судалж, Odoo ERP-тэй харьцуулсны эцэст
(2026-08-20) гаргасан шийдвэр: **суурь бүтцийг нь авч, илэрсэн бүх дутагдлыг
засна.**

## Юуг хэвээр нь авах вэ

- Tenant тусгаарлалт **DB давхаргад RLS-ээр** (Odoo шиг апп давхаргад биш)
- Tenant бүр өөрийн role-уудтай (`admin` / `manager` / `user` default + custom)
- Permission = flat string код, модуль өөрөө зарладаг, CRUD-д уягдаагүй
  (`devices.register` гэх мэт үйлдэл илэрхийлж чадна)
- Зөвхөн нэмэгдэх (additive) — deny дүрэм байхгүй
- Эрхийн шийдвэр нэг JOIN + богино хугацааны кэш
- Odoo-гийн domain engine, field-level эрх, AND/OR семантик — **авахгүй**
  (үнэ цэн нь төвөгтэй байдлаа дийлдэггүй)

## Засварууд (Gerege-ийн дутагдал → шийдэл)

### 1. Suffix-ийн дүрэм → тунхаглал

Gerege: `.read`-ээр төгссөн permission суулгах үед автоматаар бүх role-д
очдог — community модулийн нэрлэлт дээр аюулгүй байдал тогтдог байсан.

nexus-mini: модуль permission бүрдээ **default нь аль role-д очихоо өөрөө
зарлана** (Odoo-гийн XML security датаны Go хувилбар):

```go
nexus.PermissionDefinition{
    Code:         "devices.read",
    Name:         "Төхөөрөмж харах",
    DefaultRoles: []string{"manager", "user"}, // admin үргэлж бүгдийг авна
}
```

`DefaultRoles` хоосон бол зөвхөн admin — аюулгүй тал руугаа default.

### 2. Permission код чөлөөтэй → модулийн prefix албадана

Хоёр community модуль ижил код зарлаж мөргөлдөх/эрх өвлөх эрсдэлтэй байсан.
nexus-mini: `Register()` үед permission код `<moduleShortID>.`-ээр эхлэхгүй
бол бинари **асахгүй** (boot-time алдаа, runtime сюрприз биш).

### 3. Мөрийн түвшин байхгүй → `scope: all | own`

Odoo-гийн хамгийн их хэрэглэгддэг ганц record rule нь "Own Documents Only".
Бүрэн domain engine биш, яг энэ нэг шилжүүлэгчийг л авна:

- `role_permissions`-д permission бүрийн grant `scope` (`all` эсвэл `own`)
  багана тээнэ
- Модуль хүснэгтдээ `created_by` конвенц баримтална
- SDK-гийн `RequirePermission` шийдсэн scope-оо context-д хийнэ; модуль
  `nexus.Scope(ctx)` уншаад `own` бол `created_by = <user>` шүүлт нэмнэ
- Модуль permission-даа `OwnScope: true` зарласан үед л энэ сонголт UI-д
  харагдана

### 4. Role удамшилгүй → жижиг implied гинж

Gerege-д суулгагч manager, user хоёрт тус тусад нь өгдөг, шатлал байхгүй.
nexus-mini: role-д `implies` (нэг эцэг) — default нь
`admin ⊃ manager ⊃ user`. Үр дүнтэй эрх = өөрийн + implied гинжний нэгдэл
(recursive CTE, гүн ≤ 5). Custom role ч аль нэгийг нь өвлөж болно.

### 5. Модуль өөрөө middleware тавьдаг → платформ хаалгаа өөрөө барина

Gerege-д модульд root router өгдөг тул auth-аа мартсан route нээлттэй үлддэг
байсан. nexus-mini: модулийн route-ууд **урьдчилан хамгаалагдсан** дэд
router дээр суудаг — платформ `/api/apps/<shortid>/*`-д tenant auth +
"апп суусан эсэх" gate-ийг АЛЬ ХЭДИЙН тавьсан байна. Модуль зөвхөн нарийн
permission middleware-ээ л нэмнэ. Мартах юм үлдэхгүй.

### 6. Платформын админ

`users.platform_admin` багана хэвээр (Gerege-ийн `is_admin` шиг), гэхдээ
өмнөх nexus-ийн сургамжаар **GUC-ээр биш** — платформын админ хүсэлтүүд
тусдаа DB pool (`nexus_admin` role)-оор явж, RLS бодлого нь
`pg_has_role`-оор таньдаг. Апп pool өөрийгөө өргөмжлөх зам байхгүй.

## Хүснэгтүүд

```
users(id, email, password_hash, name, platform_admin, created_at)
tenants(id, slug, name, created_at)
memberships(id, tenant_id, user_id) UNIQUE(tenant_id, user_id)
roles(id, tenant_id, code, name, implies, active) UNIQUE(tenant_id, code)
permissions(code PK, module_id, name, description, own_scope, default_roles jsonb)
role_permissions(role_id, permission_code, scope 'all'|'own')
membership_roles(membership_id, role_id)
```

Эрхийн шийдвэр: membership → membership_roles → (roles implied гинж) →
role_permissions → `map[code]scope`, 30 секунд кэш.

### 7. Runtime хамгаалалт (2026-08-21, OGN-тэй дахин харьцуулсны дараа)

Тунхаглалын үеийн шалгалтууд (`Register` panic) runtime-д ч давхар байна:

- **Оноолт өөрийн эрхээс хэтрэхгүй — гурван хаалга бүгд.** (a) `PUT /api/roles/{id}/grants`: шинэ/өргөссөн оноолт бүрийг оноож буй хүн өөрөө эзэмшсэн байх ёстой (байсан оноолтыг хэвээр үлдээх/нарийсгахад шаардахгүй); (b) role оноох (`POST /api/members`, `PUT /api/members/{id}/roles`): role-ийн implies гинжтэйгээ олгодог бүх permission оноож буй хүнд багтах ёстой — `core.members.manage`-тэй хүн өөрийгөө `admin` болгож чадахгүй; (c) `POST /api/roles` `implies`: өвлөх role мөн (b) дүрмээр.
- **`own` зөвхөн `own_scope` permission-д.** Каталогт `own_scope=false` бол 400 — үгүй бол шүүдэггүй модульд «own» чимээгүй бүрэн эрх болно.
- **admin role-ийн оноолт автомат** — гараар засах API 400 (UI аль хэдийн түгжсэн байсан).
- **Шинэ permission backfill.** Boot-ийн `Sync` permission бүрийг **нэг tx-д** upsert + (анх удаа орж байвал, `RETURNING xmax = 0`) тухайн модулийг суулгасан tenant бүрт (core бол бүх tenant) default оноолт хийнэ — алдвал бүхэлдээ буцаж дараагийн асалтад дахин оролдоно (хагас төлөв үлдэхгүй). Байгаа кодод хүрэхгүй — гараар хассан оноолт сэргэхгүй.
- **`membership_roles` tenant триггер** (00007) — өөр tenant-ийн role оноохыг DB давхаргад хориглоно; grants CTE-ийн суурь гишүүн ч `r.tenant_id = $1` шалгана.
- **`own` унших талдаа ч үйлчилнэ** — `devices.read` `OwnScope: true`, `list` `ownFilter` хэрэглэнэ (жишээ модуль тул загвар болно).

- **Cross-process cache invalidation** — `internal/core/bus`: Postgres `LISTEN/NOTIFY` (`nexus_invalidate` суваг, `grants:<tenant>` / `gate:<tenant>`). Redis хэрэггүй — DB аль хэдийн бий. At-most-once тул 30с TTL хэвээр (bus нь хоцролтыг ~0 болгоно, зөв байдлын суурь биш).
- **Impersonation (платформ админ → tenant-ийн хэрэглэгч)** — админ панель тусдаа домэйн тул cookie шууд тавьж чадахгүй: `POST /api/admin/impersonate` → DB талын шалгалттай (админ мөн, бай platform_admin биш, гишүүнчлэл бий) нэг удаагийн 60с handover token → portal `GET /api/auth/handover?token=` → 30 мин-ын `sessions.impersonated_by` тэмдэгтэй session. Тухайн session-ий audit бүртгэл бүрд `impersonated_by` хавсарна, `platform.impersonate` үйлдэл tenant-ийн гинжид бичигдэнэ, профайл/нууц үг солих 403, portal-д сануулах banner. `handover_tokens` policy-гүй RLS — зөвхөн definer функцээр.

Нээлттэй: `RevokeOnUninstall` дуудах uninstall зам.
