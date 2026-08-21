# nexus-mini

**Татаад ажиллуулаад, дээр нь модулиа бичээд, store-д нийтэлдэг платформын цөм.**

nexus-mini нь [Gerege Nexus](https://github.com/gerege-systems/open-gerege-nexus)-аас
санаа авсан, жижигрүүлж ойлгомжтой болгосон нээлттэй эхийн multi-tenant
платформ юм. Цөм нь tenant, RBAC, audit, OIDC/SSO federation, resilience-ийг
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
psql -v owner_pw='...' -v app_pw='...' -v admin_pw='...' -f deploy/01-roles.sql

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

Postgres (role-ууд автомат), миграц + админ, API, вэб бүгд асаад
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
    <нэр>/ui/        модулийн portal хуудас + толь → build үед portal руу хуулагдана
frontend/            Next.js — landing + portal (modules.json: ямар UI орох)
admin/               Next.js — платформын админ (тусдаа апп, тусдаа домэйн)
catalog/             локал каталог (registry-гүй үеийн fallback)
docs/                шийдвэр, архитектур, модуль хөгжүүлэх гарын авлага
```

## Модуль хөгжүүлэх

Модуль бол `pkg/nexus.Module` interface-ийг хэрэгжүүлсэн Go package.
Permission-оо тунхаглаж, цэсээ зарлаад, урьдчилан хамгаалагдсан router дээрээ
route-уудаа бүртгэнэ — үлдсэнийг (tenant тусгаарлалт, auth, суулгалт, RBAC
оноолт, audit) платформ хийнэ. Дэлгэрэнгүй:
[docs/03-module-guide.md](docs/03-module-guide.md).

### Өөрийн компанид — fork хийхгүй

Цөмийг хамаарал болгоод өөрийн дистрибуц үүсгэнэ: `go.mod`-д
`github.com/gerege-systems/nexus-mini/backend` + модулиуд, `main.go` нь
`core.Main(inventory.New())`. Цөмийн шинэчлэлт = `go get ...@v1.x` — merge
байхгүй. Өөрийн store, өөрийн харилцагчид (tenant), өөрийн платформ админ.
Дэлгэрэнгүй: [docs/03-module-guide.md → «Өөрийн дистрибуц»](docs/03-module-guide.md#өөрийн-дистрибуц--цөмийг-fork-хийхгүй).

## Баримт бичиг

| Файл | Агуулга |
|---|---|
| [docs/00-decisions.md](docs/00-decisions.md) | Архитектурын шийдвэрүүд |
| [docs/01-lessons.md](docs/01-lessons.md) | Өмнөх төслөөс авсан сургамж, мөрдөх дүрмүүд |
| [docs/02-rbac.md](docs/02-rbac.md) | RBAC — Gerege Nexus-ийн суурь + засварууд |
| [docs/03-module-guide.md](docs/03-module-guide.md) | Модуль хөгжүүлэх гарын авлага |

## License

Apache 2.0 — [LICENSE](LICENSE)
