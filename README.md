# nexus-mini

**Татаад ажиллуулаад, дээр нь модулиа бичээд, store-д нийтэлдэг платформын цөм.**

nexus-mini нь [Gerege Nexus](https://github.com/gerege-systems/open-gerege-nexus)-аас
санаа авсан, жижигрүүлж ойлгомжтой болгосон нээлттэй эхийн multi-tenant
платформ юм. Цөм нь tenant, RBAC, audit-ыг (OIDC/SSO federation — үе 3, resilience — үе 4)
өгнө; бизнесийн боломж бүр **модуль** болж app store-оор ирнэ.

- **Go** (chi + pgx) · **PostgreSQL 16** (Row-Level Security) · **Next.js**
- Apache 2.0 · Баримт монголоор

## Эхлүүлэх

Бүх тохиргоо нэг env файлд: DB холболтууд + анхны админыхаа
имэйл/нууц үг. `migrate` нь миграц хийгээд, админ байхгүй бол env-ээс үүсгэнэ.

```bash
git clone https://github.com/gerege-systems/nexus-mini.git
cd nexus-mini

# 1. DB role + сан (нэг удаа, superuser-ээр):
psql -v owner_pw='...' -v app_pw='...' -v admin_pw='...' -v auth_pw='...' -f deploy/01-roles.sql

# 2. Тохиргоо:
cp .env.example backend/nexus-mini.env   # DB URL + ADMIN_* бөглөнө

# 3. Миграц + анхны админ, дараа нь сервер (бүгд Makefile-аар):
make migrate
make serve                               # API :8084
make web                                 # portal :3020 (эхлээд cd frontend && pnpm install)
```

Бүх команд зөвхөн Makefile-аар: `make migrate` (миграц + env-ээс анхны админ),
`make serve`, `make web`, `make admin`, `make check`. Бинарийг шууд дуудахгүй.
Тохиргоо `backend/nexus-mini.env` файлд амьдарна (`ENV_FILE=/зам` гэж өөр файл
зааж болно); орчны хувьсагч файлаас дээгүүр.

### Docker Compose

```bash
ADMIN_EMAIL=tanii@mail.mn ADMIN_NAME="Таны нэр" ADMIN_PASSWORD='нууц-үг' \
  docker compose up -d
```

Postgres (role-ууд автомат), миграц + админ, API, вэб (:3020), админ панель (:3021) бүгд асаад
`http://localhost:3020` бэлэн болно.

## Бүтэц

```
backend/
  cmd/nexus-mini/    энэ репогийн дистрибуц: core.Main(apps.All()...) — 1 мөр
  core/              цөм САН хэлбэрээр (Main: migrate · serve) — дистрибуц импортолно
  db/migrations/     цөмийн SQL миграцууд
  pkg/nexus/         Модулийн SDK — модуль зөвхөн үүнээс хамаарна (semver)
  internal/core/     цөмийн дотоод: tenant, auth, rbac, audit, appstore, bus
  apps/              модулиуд (devices, organisation; apps.go All()-д нэг мөр)
    <нэр>/ui/        модулийн portal хуудас + толь → prebuild-д (sync-modules.mjs) portal руу хуулагдана
  pkg/registry/      app store registry-ийн нийтийн гэрээ (манифест, Ed25519 index)
  cmd/nexus/         дистрибуцийн CLI: init · add · upgrade · remove · list
  cmd/nexus-registry/ registry эзэмшигчийн хэрэгсэл: keygen · build · verify
frontend/            Next.js — landing + portal (modules.json: ямар UI орох)
admin/               Next.js — платформын админ (тусдаа апп, тусдаа домэйн)
catalog/             локал index fallback (registry хүрэхгүй үед)
docs/                шийдвэр, архитектур, модуль хөгжүүлэх гарын авлага
```

## Модуль хөгжүүлэх

Модуль бол `pkg/nexus.Module` interface-ийг хэрэгжүүлсэн Go package.
Permission-оо тунхаглаж, цэсээ зарлаад, урьдчилан хамгаалагдсан router дээрээ
route-уудаа бүртгэнэ — үлдсэнийг (tenant тусгаарлалт, auth, суулгалт, RBAC
оноолт, audit) платформ хийнэ. Дэлгэрэнгүй:
[docs/03-module-guide.md](docs/03-module-guide.md).

### Өөрийн компанид — fork хийхгүй, `nexus` CLI

```bash
go run github.com/gerege-systems/nexus-mini/backend/cmd/nexus@latest init my-dist   # цөм = хамаарал
cd my-dist && go run github.com/gerege-systems/nexus-mini/backend/cmd/nexus@latest add organisation
make migrate && make serve
```

`add` нь гарын үсэгтэй registry-ээс (default: gerege-systems/nexus-registry raw URL, эсвэл өөрийн) модулийн
манифестийг татаж, `go get` + `main.go` + portal UI-г автоматаар нэмнэ;
`upgrade` permission өргөссөн бол `-approve` шаардана. Цөмийн шинэчлэлт =
`go get …/backend@v1.x` — merge байхгүй. Өөрийн store, өөрийн харилцагчид
(tenant), өөрийн платформ админ. Дэлгэрэнгүй:
[docs/03-module-guide.md → «Өөрийн дистрибуц»](docs/03-module-guide.md#өөрийн-дистрибуц--цөмийг-fork-хийхгүй),
registry: [gerege-systems/nexus-registry](https://github.com/gerege-systems/nexus-registry).

## Баримт бичиг

| Файл | Агуулга |
|---|---|
| [docs/00-decisions.md](docs/00-decisions.md) | Архитектурын шийдвэрүүд |
| [docs/01-lessons.md](docs/01-lessons.md) | Өмнөх төслөөс авсан сургамж, мөрдөх дүрмүүд |
| [docs/02-rbac.md](docs/02-rbac.md) | RBAC — Gerege Nexus-ийн суурь + засварууд |
| [docs/03-module-guide.md](docs/03-module-guide.md) | Модуль хөгжүүлэх гарын авлага |

## License

Apache 2.0 — [LICENSE](LICENSE)
