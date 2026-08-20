# nexus-mini

Нээлттэй эхийн multi-tenant платформын цөм (tenant, RBAC, audit, app store) + компайл-цагийн Go модулиуд. github.com/gerege-systems/nexus-mini (public, Apache 2.0). Баримт бичиг, UI, коммит мессеж — монголоор. Архитектурын шийдвэр: docs/00-decisions.md, RBAC: docs/02-rbac.md, модуль бичих: docs/03-module-guide.md.

## Стек ба бүтэц
- Go 1.25 (chi + pgx + goose + httprate) · PostgreSQL 16 RLS · Next.js 16 / React 19 / TS 7. ORM байхгүй, ESLint байхгүй (TS7-той зөрчилддөг, хойшлуулсан — `next build`-ийн tsc л шалгана).
- `backend/cmd/nexus-mini/` — CLI, зөвхөн `migrate` + `serve` (setup/admin комманд байхгүй, санал болгохгүй)
- `backend/db/migrations/` — цөмийн goose SQL; `backend/db/embed.go`-оор embed
- `backend/pkg/nexus/` — модулийн SDK (Module interface, Register, Scope, DB/web туслахууд)
- `backend/internal/core/{auth,rbac,audit,appstore,identity,db,config,handlers,...}` — цөм (handlers нь тусдаа subpackage — import cycle-ээс сэргийлэх)
- `backend/apps/devices/` — жишээ модуль (өөрийн migrations/ FS-тэй); модулиуд internal БИШ — гадны репо импортолж болно
- `backend/apps/apps.go` — `RegisterAll()`: бинарид орох модулиудын жагсаалт, нэг мөр = нэг модуль
- `frontend/` — landing + tenant portal (:3020) · `admin/` — платформын админ, ТУСДАА апп (:3021) · API :8084
- `catalog/apps.json` — локал app store каталог (registry-гүй үеийн fallback, `CATALOG_PATH`)
- `deploy/` — `01-roles.sql` (DB role-ууд), `deploy.sh`, 3 systemd unit, `nginx-nexus-mini.conf`
- Тохиргоо `backend/nexus-mini.env` (`.env.example`-оос; gitignore-д). DB role-ууд: `nexus_app` (RLS үйлчилнэ, апп), `nexus_admin` (`nexus_platform` гишүүн → бодлого платформ гэж таньдаг), `nexus_owner` (schema эзэн, зөвхөн миграц).

## Коммандууд
- `make migrate` — миграц + env-д ADMIN_* байгаа бөгөөд админ огт байхгүй бол анхны платформ админ үүсгэнэ
- `make api` / `make web` — API :8084 / portal dev :3020; админ: `cd admin && pnpm dev` (:3021)
- `make check` — `GOOS=linux GOARCH=amd64 go build` + `go vet` + `go test` + SDK-ийн хилийн шалгалт (`apps/` → `backend/internal/*` импорт байвал унана)
- `make push` — check амжилттай бол л `git push`
- Integration тестүүд (`db/rls_test.go`, `rbac/rbac_integration_test.go`) `NEXUS_TEST_DATABASE_URL` + `NEXUS_TEST_DATABASE_URL_OWNER` шаардана, байхгүй бол Skip
- `docker compose up -d` — PG + migrate + api + web + admin (ADMIN_* env-ээр дамжуулна)

## Deploy
- runestone VPS (`ssh bay@46.250.254.85`), `/srv/nexus-mini`, deploy key alias `github-nexus-mini`.
- Домэйн: `nexus-mini.runestonetechnologies.com` (portal; nginx `/api/*` → :8084, бусад → :3020), `nexus-mini-admin.runestonetechnologies.com` (admin :3021, мөн `/api/*` → :8084). CORS байхгүй — same-origin rewrite.
- Сервер дээр: `bash deploy/deploy.sh` — pull → go build (атом mv, `.prev` үлдээнэ) → `migrate --env /home/bay/secrets/nexus-mini.env` → frontend/admin `pnpm build` (`.next.new` → атом солилт) → restart 3 unit → health curl.
- systemd: `nexus-mini-api`, `nexus-mini-web`, `nexus-mini-adminweb`. Unit нь `node_modules/.bin/next start` шууд дууддаг — pnpm/corepack дуудахгүй (ProtectHome).
- deploy.sh ХИЙДЭГГҮЙ зүйлс (гараар): unit файл өөрчлөгдвөл `sudo cp deploy/*.service /etc/systemd/system/ && sudo systemctl daemon-reload`; nginx conf өөрчлөгдвөл `sudo systemctl reload nginx`.
- Secrets: `/home/bay/secrets/nexus-mini.env` (600) — DATABASE_URL / _ADMIN / _OWNER, ADMIN_*, CATALOG_PATH, ENVIRONMENT=production. DB нэр `nexus_mini`.
- Go сервер дээр `/usr/local/go` (1.25); apt-ийн `/usr/bin/go` 1.22 — гараар build хийвэл PATH-ыг түрүүлж тавь.
- Прод DB цэвэр, demo дата байхгүй; тест мөр оруулсан бол устга.

## Инварантууд
- Модуль зөвхөн `pkg/nexus`-ээс хамаарна, `internal/*` хэзээ ч импортлохгүй (`make check` барина). Permission код модулийн prefix-тэй байхыг `Register` албаддаг; default оноолт `DefaultRoles` тунхаглал (`"user:own"` хэлбэр); scope `all|own` (`created_by` + `nexus.Scope(ctx)`); модулийн router платформ урьдчилан хамгаална (auth + install gate) — модуль өөрөө auth хийхгүй.
- Role implies гинж admin⊃manager⊃user (recursive CTE). Платформ админ = DB role (`pg_has_role`), GUC флаг биш.
- Нэвтрэлт/танилтын бүх хайлт (session, имэйл, audit prev_hash) RLS-ийн ӨМНӨ ажилладаг тул SECURITY DEFINER функцээр явна; SECURITY DEFINER функц бүр `SET search_path = pg_catalog, public`-тэй.
- Мутаци бүр `Audit(...)` бичнэ + RBAC-д нөлөөлбөл `rbac.Invalidate` дуудна (кэш). Audit hash гинж DB дотор, append-only trigger-тэй.
- Шинэ tenant үүсгэхдээ id-г урьдчилан гаргаж `app.tenant_id`-г ЭХЭЛЖ тохируулна — INSERT..RETURNING нь RLS SELECT бодлого шаарддаг (handlers/auth.go createTenant).
- Миграц зөвхөн `nexus_owner`-оор; апп `nexus_app`/`nexus_admin`-аар. Модулийн миграц модулийн өөрийн FS-д (`backend/apps/<x>/migrations/`), цөмийнхөд нэмэхгүй.
- Вэб талд setup wizard байхгүй, landing төлөв шалгадаггүй; анхны админ зөвхөн env + `migrate`-ээс. Энэ чиглэлээр "сайжруулалт" санал болгохгүй.
- UI chrome-ийг өөрөө зохиохгүй — open-gerege-nexus-ийн дизайн жишиг; админ teal accent.
- Frontend-ийн `.next` gitignore-д — серверт build заавал (deploy.sh хийнэ).

## Шалгах
- Push-ийн өмнө `make check` (linux build + vet + test + SDK хил). Integration тест ажиллуулах бол NEXUS_TEST_* env.
- Прод дээр `bash deploy/deploy.sh` өөрөө `is-active` + `/health`, `:3020/`, `:3021/login` curl хийдэг — гаралтыг нь унш. Next build/typecheck-ийг сервер дээр л баталгаажуулна.

## Нээлттэй ажил
- Үе 2: төв registry (`nexus-mini-registry.*` :8085) + гарын үсэгтэй каталог + `nexus-mini add` CLI
- Үе 3: OIDC provider + SSO client federation · Үе 4: resilience (breaker/loadshedder/retry/singleflight)
- Хойшлуулсан: ESLint (TS7 дэмжлэг гармагц), өдөр тутмын pg_dump backup (production ашиглалтын өмнө заавал) — 2026-08-20

@~/.claude/rules/go-pg.md
