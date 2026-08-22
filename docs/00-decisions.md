# Шийдвэрүүд

2026-08-20-нд хэрэглэгчтэй тохирсон, өөрчлөгдвөл энд шинэчилнэ.

## Юу вэ

**nexus-mini** — [open-gerege-nexus](https://github.com/gerege-systems/open-gerege-nexus)-аас
санаа авсан, жижигрүүлж ойлгомжтой болгосон нээлттэй эхийн платформын цөм.
Хэн ч татаж аваад ажиллуулж, дээр нь чөлөөтэй модуль хөгжүүлж, app store-д
нийтэлж чадахуйц байхад бүх зүйл чиглэнэ.

- Репо: `gerege-systems/nexus-mini`, Apache 2.0, нээлттэй эх
- Stack: Go (chi + pgx, **GORM ашиглахгүй**), PostgreSQL 16 + RLS, Next.js (App Router), pnpm
- Баримт монголоор, код/identifier англиар

## Цөмд юу багтах вэ (хэрэглэгчийн тодорхойлолт)

1. **Tenant** — байгууллага бүр тусгаарлагдсан (RLS)
2. **RBAC** — модулиуд permission зарладаг, tenant дотор role-д оноодог
3. **Audit** — append-only, hash chain (DB дотор тооцно)
4. **OIDC provider + SSO client (federation)** — өөрөө токен гаргана, мөн өөр
   nexus-mini инстансын клиент болж чадна
5. **Resilience** — circuit breaker, load shedder, retry, singleflight
6. **App store** — модуль нийтлэх + татаж суулгах шийдэл

## Модулийн загвар: compile-time + CLI

Модуль нь Go package — бинарид компиллогдоно. `pkg/nexus` дахь `Module`
interface нь цорын ганц гэрээ. "Татаж авах" гэдэг нь:

```
nexus add devices   # registry-ээс манифест татна → go get → main.go маркер + UI хуулбар → rebuild
```

Runtime plugin/WASM хийхгүй — ойлгомжтой байдал нь mini-гийн гол чанар.

## Store топологи: төв registry + локал fallback

- Төв registry = **статик, гарын үсэгтэй** `index.json` (Ed25519), Git репо
  `gerege-systems/nexus-registry` raw URL-аар — сервер код байхгүй
  (2026-08-22-нд «runestone дээр сервер» гэснээс ингэж хялбарчилсан)
- Инстанс бүр `REGISTRY_URL`-ээс (default gerege-systems/nexus-registry (GitHub raw), түлхүүр цөмд) татна;
  офлайн бол кэш → `catalog/index.json` файл
- Community хөгжүүлэгч манифестаа (`make manifest`) registry репод PR-аар нийтэлнэ

## Хэрэглэгчийн 3 шаардлага (2026-08-20)

1. **Эхний ажиллуулалт: бүх тохиргоо env файлд** — DB холболтууд +
   `ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD`-оо `nexus-mini.env`-д бичээд
   `make migrate` (миграц + админ байхгүй бол env-ээс үүсгэнэ) →
   `make serve`. CLI = migrate + serve хоёрхон комманд; тусдаа
   setup/admin комманд байхгүй. **Бүх команд зөвхөн Makefile-аар**
   (2026-08-21) — бинарийг шууд дуудахгүй; deploy.sh ч `make build` /
   `make migrate ENV_FILE=…` ашиглана. Гагцхүү systemd unit (ExecStart)
   болон Docker image (make байхгүй) бинарийг шууд дуудна. Вэб талд setup wizard БАЙХГҮЙ, landing
   төлөв шалгадаггүй. Түүх: вэб wizard → CLI setup → бүгдийг env-д
   (хэрэглэгчийн шаардлагаар алхам алхмаар хялбарчилсан, 2026-08-20).
2. **Админ панель** — платформын админд хэрэгтэй бүх тохиргоо нэг дор
3. **Landing page** — цөм ба app store яаж ажилладаг, модуль яаж хөгжүүлэх нь
   бүгд тодорхой; тэндээсээ өөрийгөө + tenant-аа бүртгүүлээд орно; шинэ tenant
   эхлээд app store-оо хараад модуль суулгахаас эхэлдэг. "Юу болоод байгаа нь
   мэдэгддэггүй, аль хэдийн ажиллаад байдаг ойлгомжгүй юм" байж БОЛОХГҮЙ.

## Frontend: portal ба админ тусдаа

`frontend/` = landing + tenant portal; `admin/` = платформын админ —
**тусдаа код, тусдаа домэйн** (хэрэглэгчийн шаардлага, 2026-08-20).
Админ апп teal accent-тэй, зөвхөн platform_admin нэвтэрнэ. UI chrome нь
open-gerege-nexus-ийн дизайныг жишиг болгоно (өөрөө зохиохгүй — сургамж #10).

## Жишээ модуль

**Төхөөрөмжийн бүртгэл** (`devices`) — SDK-г баталдаг анхны модуль.

## Үе шат

| Үе | Агуулга |
|---|---|
| 1 | Цөм: миграц + CLI (migrate/serve) + session auth + tenant + RBAC + audit + module SDK + devices модуль + app store (локал каталог) + landing/portal/админ панель |
| 2 ✅ | Гарын үсэгтэй статик registry + `nexus` CLI (init/add/upgrade/remove/list) — 2026-08-22 |
| 3 | OIDC provider + SSO client federation |
| 4 | Resilience давхарга, чанаржуулалт |

Үе бүр дуусаад **ажиллаж баталгаажсаны дараа** л дараагийнх эхэлнэ
(өмнөх сургамж #11 — бүгдийг зэрэг хийхгүй).

## Хойшлуулсан зүйлс

- **ESLint**: eslint-config-next-ийн шүтэлцээ (typescript-eslint) нь одоогийн
  TypeScript 7-г дэмжихгүй байгаа тул тохиргоог түр хойшлуулав (2026-08-20).
  Дэмжлэг гармагц нэмнэ; түүнийг хүртэл `tsc --noEmit` (next build) л хамгаална.
- **DB backup стратеги**: cron-оор өдөр тутмын pg_dump — production
  ашиглалтын өмнө заавал (одоогоор demo instance тул алга).

## Deploy (runestone)

| Хэсэг | Хаяг | Порт |
|---|---|---|
| Portal + API (/api/* → Go) | nexus.runestonetechnologies.com | 3020 / 8084 |
| Платформын админ | nexus-admin.runestonetechnologies.com | 3021 |
| Registry | raw.githubusercontent.com/gerege-systems/nexus-registry (статик) | — |

Хуучин nexus-ийн deploy 2026-08-20-нд бүрэн устгагдсан, портууд чөлөөтэй.

## OGN-ээс авсан 3 зүйл (2026-08-22)

Tenant/байгууллага/гишүүний загварыг open-gerege-nexus-тэй харьцуулсны дараа:

1. **Байгууллагын профайл** (цөм) — `tenant_profiles` (хуулийн нэр, регистр, ТТД, хаяг, утас, имэйл, вэб) + байгууллагын нэр засах. `GET/PUT /api/tenant/profile`, унших — гишүүн бүр, бичих — шинэ `core.settings.manage` (admin-д автоматаар; backfill-ээр байгаа tenant-уудад ч орсон). Portal `/settings`. `tenants` UPDATE нь апп role-д зөвхөн `name` багана.
2. **Түдгэлзүүлэх + зөвхөн-унших** (платформ админ) — `tenants.suspended_at/suspension_reason/read_only`; `RequireTenant` хүсэлт бүрд `tenant_state()` definer функцээр шалгана (30с кэш, bus-аар invalidate): suspended → 403, read_only + бичих → 503 + Retry-After. `PUT /api/admin/tenants/{id}/state`, админ UI «Төлөв», portal banner (`/api/me.tenant_state`). Logout/байгууллага солих нь RequireTenant-ийн гадна тул үргэлж ажиллана.
3. **`organisation` модуль** (цөм биш — store-оос суулгана) — `org_departments` мод (дээд нэгж, менежер, идэвхтэй, мөчлөг шалгалт) + `org_positions` (гишүүнчлэл бүрт хэлтэс, албан тушаал). `organisation.read/manage`. Portal `/organisation/departments`, `/organisation/people`. OGN ч үүнийг эцэст нь апп болгосон (`00055_organisation_rename`).

Аваагүй: `allowed_tenant_ids` (олон байгууллагаас зэрэг унших — RLS-ийг төвөгтэй болгоно, группын компанид л хэрэгтэй), толгой компанийн холбоос, 30 хоногийн хүлээлттэй устгал, квот — хэрэгцээ гарахад.

## OGN цөмтэй харьцуулсны дараах 5 жижиг цоорхой (2026-08-22)

1. **Дансны түр түгжээ** — 15 мин дотор 5 буруу → 15 мин (`auth_login_result`, `auth_lockout`; IP rate limit-ээс тусдаа; байхгүй имэйлийг тоолохгүй). 429 + Retry-After.
2. **Session idle timeout** — 90 мин хэрэглээгүй бол дуусна (`sessions.last_seen_at`, lookup 5 минутад нэгээс олон бичихгүй). Impersonated session 30 мин хэвээр.
3. **Security headers Go талд** (nosniff, DENY, Referrer-Policy, CSP `default-src 'none'`, no-store, HSTS production) — nginx-гүй дистрибуц/Docker-т ч хамгаалалттай; nginx snippet-тэй давхардах нь ижил утгатай тул хор байхгүй.
4. **Production guard** — `ENVIRONMENT=production` үед `PORTAL_URL` https биш бол асахгүй; env-д `ADMIN_PASSWORD` үлдсэн бол сануулна.
5. **Түдгэлзүүлэх → session шууд устгах** (`auth_sessions_revoke_tenant`, audit-д тоо).

Аваагүй: OGN-ийн settings registry (DB→env), Sec-Fetch-Site CSRF нотолгоо — хэрэгцээ гарахад.

## Үе 2 — registry + `nexus` CLI (2026-08-22)

- **Registry = статик, гарын үсэгтэй** (`gerege-systems/nexus-registry`: `manifests/*.json` → `index.json` + Ed25519 `.sig`, raw URL). Сервер код байхгүй; нийтлэх = PR. Private key maintainer-ийн `~/.secrets/`-д, нийтийн түлхүүр цөмд default. Өөрийн registry = репо хуулж `keygen`.
- **Манифест кодоос** (`make manifest`, `registry.FromModule` reflect-ээр go_module) — drift байхгүй. `Validate()` нь Register-ийн дүрмүүдийн build-гүй хувилбар.
- **Fetch**: ETag, кэш (offline fallback), хэмжээний хязгаар, `generated_at` replay хамгаалалт; татаж чадахгүй бол boot унагаахгүй, кэш → локал `catalog/index.json`.
- **`nexus` CLI** (`go run …/cmd/nexus@latest`): `init` (tag tarball-аас frontend + маркертай main.go), `add/upgrade/remove/list`. Модулийн UI-г Go module cache-ээс `frontend/modules/<нэр>/ui` руу **хуулна** (commit хийгдэнэ, дистрибуц доторх зам — sync скриптийн репо-дотор дүрэм хэвээр). `upgrade` permission өргөсвөл `-approve`.
- Зориуд үгүй: runtime plugin/WASM, registry-д нэвтрэлт, модулийн tarball хостлох (Go proxy хийнэ).

## OGN-ээс авсан сүүлийн 3 жижиг зүйл (2026-08-23)

1. **Sec-Fetch-Site CSRF** — браузерын өөрөө тавьдаг толгой `cross-site` бол бичих хүсэлт 403 (Origin шалгалтаас гадна; handover чөлөөтэй).
2. **Устгалын хүлээлт** — `tenants.deletion_scheduled_at`; `POST /api/admin/tenants/{id}/delete` = +30 хоног, тэр дороо suspend + session revoke; `…/delete/cancel` буцаана; цагийн sweep (`SweepDeletions`, admin pool) өнгөрсөн байгууллагыг cascade устгана (audit гинж FK-гүй тул үлдэнэ). Portal banner, админ UI «Төлөв» modal. OGN-ийн two-person rule-ийг аваагүй — нэг platform_admin role тул; олон оператортой болбол нэмнэ.
3. **Хувилбарын түүх** — `app_releases` (нийтлэгчийн хувилбар анх харагдсан цаг; компиллогдсон + registry), `installation_events` (tenant бүрийн install/enable/disable/upgrade, actor). Boot-ийн Sync суулгасан tenant-уудын хувилбарыг компиллогдсон руу өргөж `upgrade` үйл явдал (actor=систем) бичнэ — permission backfill-тай хамт. Portal store «Түүх» modal, `GET /api/store/apps/{id}/history`.
