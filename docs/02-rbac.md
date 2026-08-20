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
