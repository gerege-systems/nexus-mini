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

# 3. Миграц + анхны админ, дараа нь сервер:
cd backend
go run ./cmd/nexus-mini migrate
go run ./cmd/nexus-mini serve            # API :8084

cd ../frontend && pnpm install && pnpm dev   # вэб :3020
```

CLI коммандууд: `migrate` (миграц + env-ээс анхны админ), `serve`.
Тохиргоо `nexus-mini.env` файлд амьдарна; орчны хувьсагч түүнээс дээгүүр.

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
  cmd/nexus-mini/    CLI: migrate · admin · serve
  db/migrations/     цөмийн SQL миграцууд
  pkg/nexus/         Модулийн SDK — модуль зөвхөн үүнээс хамаарна
  internal/platform/ цөм: tenant, auth, rbac, audit, appstore
  internal/apps/     жишээ модулиуд (devices)
frontend/            Next.js — landing + portal + админ панель нэг апп
catalog/             локал каталог (registry-гүй үеийн fallback)
docs/                шийдвэр, архитектур, модуль хөгжүүлэх гарын авлага
```

## Модуль хөгжүүлэх

Модуль бол `pkg/nexus.Module` interface-ийг хэрэгжүүлсэн Go package.
Permission-оо тунхаглаж, цэсээ зарлаад, урьдчилан хамгаалагдсан router дээрээ
route-уудаа бүртгэнэ — үлдсэнийг (tenant тусгаарлалт, auth, суулгалт, RBAC
оноолт, audit) платформ хийнэ. Дэлгэрэнгүй:
[docs/03-module-guide.md](docs/03-module-guide.md).

## Баримт бичиг

| Файл | Агуулга |
|---|---|
| [docs/00-decisions.md](docs/00-decisions.md) | Архитектурын шийдвэрүүд |
| [docs/01-lessons.md](docs/01-lessons.md) | Өмнөх төслөөс авсан сургамж, мөрдөх дүрмүүд |
| [docs/02-rbac.md](docs/02-rbac.md) | RBAC — Gerege Nexus-ийн суурь + засварууд |
| [docs/03-module-guide.md](docs/03-module-guide.md) | Модуль хөгжүүлэх гарын авлага |

## License

Apache 2.0 — [LICENSE](LICENSE)
