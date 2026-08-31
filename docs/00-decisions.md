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
  (2026-08-22-нд «өөрийн сервер дээр registry сервис» гэснээс ингэж хялбарчилсан)
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
| 3 ✅ | OIDC provider (PKCE, RS256/JWKS, consent, refresh rotation) + SSO client (Google/OIDC/federation) — 2026-08-23 |
| 4 | Resilience давхарга, чанаржуулалт |

Үе бүр дуусаад **ажиллаж баталгаажсаны дараа** л дараагийнх эхэлнэ
(өмнөх сургамж #11 — бүгдийг зэрэг хийхгүй).

## Хойшлуулсан зүйлс

- **ESLint**: eslint-config-next-ийн шүтэлцээ (typescript-eslint) нь одоогийн
  TypeScript 7-г дэмжихгүй байгаа тул тохиргоог түр хойшлуулав (2026-08-20).
  Дэмжлэг гармагц нэмнэ; түүнийг хүртэл `tsc --noEmit` (next build) л хамгаална.
- **DB backup стратеги**: cron-оор өдөр тутмын pg_dump — production
  ашиглалтын өмнө заавал (одоогоор demo instance тул алга).

## Deploy

| Хэсэг | Хаяг | Порт |
|---|---|---|
| Portal + API (/api/* → Go) | `<portal-host>` | 3020 / 8084 |
| Платформын админ | `<admin-host>` | 3021 |
| Registry | raw.githubusercontent.com/gerege-systems/nexus-registry (статик) | — |

Бодит домэйн, зам, secrets нь **зөвхөн сервер дээр** — репод ерөнхий хэлбэр л
бичигдэнэ (`deploy/nginx-nexus-mini.conf` бол хуулж тохируулах дээж).

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

## Үе 3 — OIDC provider + SSO client (2026-08-23)

- **Provider** `/api/oauth2/*` (issuer = `PORTAL_URL/api/oauth2` — nginx/rewrite өөрчлөлтгүй). Зөвхөн authorization code + PKCE S256 (заавал), refresh rotation (replay → гэр бүлээр хүчингүй), client_credentials; opaque access + introspect/revoke; id_token RS256 (JWT-г өөрсдөө бичсэн, sign-only, alg confusion байхгүй); consent санагдана; end_session. Клиент = tenant-ийнх (`oauth_clients`, `core.sso.manage`); хэрэглэгч клиентийн байгууллагын гишүүн байх ёстой. Токен/код/түлхүүр/зөвшөөрлийн хүснэгт зөвхөн `nexus_auth`. CORS cookie-гүй endpoint-уудад; consent CSRF-тэй.
- **RP** `internal/core/ssoclient`: Google + ерөнхий issuer (env), discovery 1ц кэш, JWKS (kid эргэлтэд дахин татна), PKCE/state/nonce HMAC cookie, JIT `SSO_AUTO_SIGNUP` (default хаалттай). Federation = өөр nexus-mini issuer.
- Зориуд үгүй: implicit/hybrid, JWT access token, dynamic registration, MFA (дараа).
- Баталгаа: бүтэн урсгал curl (code/PKCE/refresh/replay/introspect/revoke/end_session/CORS) + federation (өөрийгөө issuer-ээр) + Playwright (SSO клиент UI, consent).

## Аудитын 4 засвар (2026-08-24)

Дөрвүүлээ "нэг ч удаа гүйцэд ажиллуулж үзээгүй" урсгалд байсан; тус бүрд DB-тэй
integration тест нэмж, `make check-db`-д оруулав (хуучин код дээр унадаг нь
батлагдсан).

1. **Signup 500** — 00010-ын дараа `auth_signup`-ийг `nexus_app` дуудаж чадахгүй
   болсон ч handler `h.DB` (апп pool)-оор дуудсаар байсан → `/signup` бүрэн үхсэн.
   **Шийдвэр**: хэрэглэгчийг `nexus_auth` pool-оор, байгууллагыг апп pool-ын
   гүйлгээнд үүсгэнэ. Хоёр DB role тул НЭГ гүйлгээ боломжгүй — байгууллага
   үүсэхгүй бол дөнгөж үүссэн хэрэглэгчийг `auth_delete_tenantless_user`
   (00015, гишүүнчлэлгүй бол л устгана) -аар буцаана. Процесс дундуур унавал
   tenant-гүй хэрэглэгч үлдэж болно — тэр нь бүтээгдэхүүний хувьд хүчинтэй
   төлөв (`/org/new`). Альтернатив (бүх tenant seed-ийг SQL definer-т зөөх)-ыг
   аваагүй: role seed/permission оноолтын эх сурвалж Go-д байх ёстой.
2. **Түдгэлзүүлсэн байгууллага → login давталт** — `/api/menu` 403 → shell
   `/login` руу, login `/api/me` (RequireUser) OK → `/dashboard` → дахин 403…
   Одоо shell 403-ыг таньж **хаагдсан дэлгэц** (шалтгаан, байгууллага солих,
   гарах) үзүүлнэ; 30 хоногийн устгалын мэдэгдэл ч эндээс харагдана.
3. **Каталогийн "татаж авах" апп boot унагаана** — `app_releases.app_id` нь
   `apps` руу FK-тай атал `recordRelease` нь `apps` upsert-ээс ӨМНӨ дуудагдаж
   байсан (23503 → Sync алдаа → serve зогсоно). Дараалал засав.
4. **`nexus add` цөмийг буулгана** — манифестын `go_module` нь пакежийн зам
   байсан тул `go get <пакеж>@v1.0.0` нь агуулагч модуль (цөм)-ийг тэр таг руу
   буулгаж байв. Одоо `go_module` = модулийн үндэс (build info-оос), `import` =
   пакежийн зам; цөмийн дотоод модульд `go get` огт хийхгүй (цөмтэйгээ ирдэг).
   `Validate` нь import ⊂ go_module-ийг шаардана.

## Тестийн бүрэн аудит (2026-08-24)

Багц бүрийг төрлөөр нь хамрав; `make check` (DB-гүй), `make check-db` (бүх
багц, DB заавал), `make check-web` (frontend статик аудит + build).

| Төрөл | Юу шалгадаг |
|---|---|
| Unit (DB-гүй) | middleware (clientIP spoof, CSRF 2 давхарга, security headers, CORS), OIDC JWT/secret/redirect/scope, SSO client id_token баталгаажуулалт (fake IdP: alg=none, aud, iss, nonce, exp, kid эргэлт, хуурамч гарын үсэг), registry (гарын үсэг, replay, ETag, офлайн кэш, хэмжээ, Validate хүснэгт), pkg/nexus web (Decode хязгаар, UUID, DBError зураглал), envfile, config (production guard), identity, nexus CLI (main.go маркер, permission diff), nexus-registry CLI (build/verify/tamper) |
| Integration (DB) | RLS тусгаарлалт, RBAC (implies, scope, эрх дээшлүүлэлтийн 6 зам), signup (хоёр DB role + нөхөн устгал), session/lockout/idle/handover, tenant төлөв (suspend/read-only/устгал), OIDC бүтэн урсгал (code→token→refresh rotation→replay→introspect/revoke), audit гинж + append-only, appstore (хамаарал, default оноолт, gate, enable/disable, каталогийн апп), bus (LISTEN/NOTIFY), TenantDB GUC/rollback, модулиуд (devices own-scope, organisation мод/мөчлөг/cross-tenant) |
| Frontend статик | i18n бүрэн байдал + давхардал, middleware matcher (12 хамгаалалттай / 11 нийтийн зам), hydration эрсдэл (render дотор browser API), build |

**Тест бичих явцад олдсон 3 бодит алдаа** (бүгд засагдсан):
1. `pkg/registry` — ETag-ийг гарын үсэг шалгахаас ӨМНӨ хадгалдаг тул нэг удаа
   буруу гарын үсэг ирсний дараа клиент 304 аваад хуучин кэшэндээ мөнхөд гацдаг.
2. `apps/organisation` — same-tenant триггерийн `23514` нь 500 болж хувирдаг
   (байхгүй/өөр tenant-ийн хэлтэс өгөхөд). Одоо 400.
3. Тестийн цэвэрлэлт: `defer pool.Close()` нь `t.Cleanup`-ээс өмнө ажилладаг
   тул цэвэрлэлт хаагдсан pool дээр чимээгүй унаж, DB-д тест дата үлддэг байв.

**Хамрах хүрээ (cross-package): 76.4%.** Тестгүй үлдсэн нь ЗӨВХӨН гурван
`func main()` (os.Args уншиж os.Exit дуудна — процессын оролт, туршилтын
утгагүй). Бусад бүх функц дор хаяж нэг тестээр дайрагдана:

- **Амьд сервер** (`cmdServe`): санамсаргүй порт дээр асааж, 9 маршрутын код,
  аюулгүй байдлын толгойнууд, CSRF, login-ы rate limit шалгаад SIGTERM-ээр
  graceful унтраана.
- **CLI**: `Main help`, `withEnv`, `cmdManifest`, `cmdMigrate` + анхны
  платформ админ (өргөмжлөх нь нууц үгэнд хүрэхгүй), `nexus init` — ЛОКАЛ
  tarball сервертэй офлайн, `add/upgrade(-approve)/remove/list` — локал
  registry, цөмийн хувилбар буулгахгүйг батална, `nexus-registry keygen`.
- **Build script** (`sync-modules.mjs`) — `node --test`: нөөцөлсөн short_id
  цөмийн хуудсыг устгахгүй, зам репогоос гарахгүй, symlink/сервер код
  хуулагдахгүй, давхардал, толийн alias.
- **Админ апп** — өөрийн статик аудиттай (i18n + hydration), `make check-web`
  одоо portal + admin хоёрын build-ийг ч ажиллуулна.

Тест бичих явцад олдсон 5 дахь алдаа: `upsertAdmin`-д ч нууц үгийн урт
байтаар тоологдож байсан (`core/migrate.go`) — тэмдэгтээр болов.

Тест бичих явцад олдсон 4 дэх алдаа: **нууц үгийн урт байтаар тоологддог**
байсан — кирилл 4 тэмдэгт = 8 байт болж "8+ тэмдэгт" дүрмийг тойрдог.
Signup, нууц үг солих, гишүүн нэмэх гурвуулаа `utf8.RuneCountInString` болов.

## Нууц үгийн дүрэм (2026-08-24)

`internal/core/password.Validate` — ганц эх сурвалж; signup, нууц үг солих,
гишүүн нэмэх (түр нууц үг), анхны админ (`ADMIN_PASSWORD`) дөрвүүлээ дуудна.

- 8–128 **тэмдэгт** (байт биш).
- Зөвхөн ASCII: латин үсэг `A-Z a-z`, тоо `0-9`, тусгай тэмдэгт (`!@#$%^&*`…).
  **Кирилл, зай, эмодзи, хяналтын тэмдэгт хориотой** — layout солигдоход
  хэрэглэгч өөрийн нууц үгээ дахин оруулж чаддаггүй, терминал/гар утасны гар,
  эх файл дамжих үед төөрдөг.
- Латин үсэг + тоо + тусгай тэмдэгт **гурвуулаа** байх ёстой.
- Алдааны мессеж юуг зөрчсөнийг шууд хэлнэ (клиентэд харуулж болно); UI-д
  гурван талбарт (signup, гишүүн нэмэх, админ профайл) hint бичигдсэн.

Байгаа нууц үгэнд хамаарахгүй — зөвхөн ШИНЭЭР тавихад шалгагдана.

## Тестийн төрлүүд — эцсийн байдал (2026-08-24)

| Төрөл | Комманд | Байдал |
|---|---|---|
| Unit (Go) | `make check` | ✅ бүх багц |
| Integration (DB) | `make check-db` | ✅ 24 багц, RLS/RBAC/OIDC/handler/модуль |
| Амьд сервер | `make check-db` дотор | ✅ асаах → route/CSRF/rate limit → SIGTERM |
| Fuzz | `make check-fuzz` | ✅ 6 зорилт (нууц үг, registry JSON+гарын үсэг, OIDC authz+JWT) |
| Race detector | `make check-race` | ✅ (bus, кэш, semaphore) |
| Эмзэг байдал | `make check-vuln` | ✅ govulncheck |
| Миграцын Down | `NEXUS_TEST_ALLOW_DOWN=1` | ✅ (өгөгдөл устгадаг тул opt-in) |
| Frontend статик | `make check-web` | ✅ i18n, middleware matcher, hydration, build (portal+admin) |
| Build script (Node) | `make check-web` | ✅ `node --test` 7 тест |
| **Browser E2E** | — | ❌ репод байхгүй (гараар Playwright-аар шалгасан) |
| **Ачааллын тест** | — | ❌ (хэрэгцээ гараагүй) |

**Fuzz-аар илэрсэн бодит алдаа**: `password.Verify` нь гэмтсэн hash-д
(`t=0`, `p=0`) `argon2.IDKey`-г шууд дуудаж **panic** хийдэг байсан — DB-д
хуурамч/эвдэрсэн мөр байхад нэвтрэлт бүр процессын горимыг унагаана.
Параметрийн шалгалт нэмсэн; regression corpus `testdata/fuzz/`-д хадгалагдав.

**govulncheck-ээр илэрсэн**: сервер Go 1.25.5-аар build хийж байсан —
`net/http`, `crypto/tls`, `net/url` зэрэгт 17 мэдэгдсэн CVE. Сервер дээр Go
**1.25.13** суулгаж дахин build хийсэн (`go version -m bin/nexus-mini` →
go1.25.13); одоо кодоос дуудагддаг эмзэг байдал алга. Локал toolchain-ыг ч
шинэчлэх нь зүйтэй (`brew upgrade go`).

## Хуудасны CSP, tab-ийн нэр, TLS шалгалт (2026-08-31)

1. **HTML хуудсанд CSP** — API хариу `default-src 'none'`-той байсан ч Next-ийн
   өгдөг баримтууд ямар ч `Content-Security-Policy` авдаггүй байв. Одоо хоёр
   аппын `middleware.ts` хүсэлт бүрт nonce үүсгээд бүрэн бодлого тавина:
   `script-src 'self' 'nonce-…' 'strict-dynamic'`, бусад нь `'self'`
   (`style-src`-т л `'unsafe-inline'` — React-ийн style атрибутыг nonce-оор
   хамгаалах боломжгүй). Next өөрийн inline flight script-үүддээ nonce-ыг
   ХҮСЭЛТИЙН CSP толгойгоос уншиж тавьдаг тул middleware түүнийг request
   header-т ч давхар бичнэ; layout нь `x-nonce`-оос уншиж theme script-д өгнө.
   **Үр дагавар**: root layout `headers()` уншдаг тул бүх хуудас dynamic —
   энэ апп-д static хуудас байхгүй тул зардал нь бага. RSC/prefetch хариуд CSP
   тавихгүй (баримт биш, nonce нь client-д кэшлэгдэж хуучирна).
2. **Хуудас бүр өөрийн нэртэй** — бүх page нь client component тул `metadata`-г
   зэрэгцээ `layout.tsx`-аас өгнө; root layout-д `title.template` (`%s · nexus-mini`,
   админд `%s · Платформын админ`). Модулийн хуудас ч мөн адил (03-module-guide #6).
3. **`Next-Action` толгойг nginx таслана** — апп server action ашигладаггүй
   атал сканнерууд хог `Next-Action`-той POST илгээж Next-ийн лог руу
   `Server Reference ID did not match…` алдаа бичүүлж байв.
4. **`deploy/check-tls.sh`** — nginx-ийн 443 дээр үйлчилдэг нэр бүрээр SNI
   тавьж, сертификатын SAN-д тэр нэр байгаа эсэхийг шалгана; `deploy.sh`
   төгсгөлд ажиллана. Шалтгаан: wildcard серт **нэг л шат** таардаг тул
   `admin.nexus.*` шиг хоёр шаттай нэр `*.<домэйн>` сертэд хамрагдахгүй —
   nginx/DNS/service бүгд хэвийн, лог цэвэр байхад браузер л TLS дээр унана
   (2026-08-31-нд яг ингэж админ домэйн чимээгүй унасан).
