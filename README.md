# nexus-mini

**Татаад ажиллуулаад, дээр нь модулиа бичээд, store-д нийтэлдэг платформын цөм.**

nexus-mini нь [Gerege Nexus](https://github.com/gerege-systems/open-gerege-nexus)-аас
санаа авсан, жижигрүүлж ойлгомжтой болгосон нээлттэй эхийн multi-tenant
платформ юм. Цөм нь tenant, RBAC, audit, OIDC/SSO federation, resilience-ийг
өгнө; бизнесийн боломж бүр **модуль** болж app store-оор ирнэ.

- **Go** (chi + pgx) · **PostgreSQL 16** (Row-Level Security) · **Next.js**
- Apache 2.0 · Баримт монголоор

## Эхлүүлэх

```bash
git clone https://github.com/gerege-systems/nexus-mini.git
cd nexus-mini
docker compose up -d     # PostgreSQL + миграц + API + вэб
```

Дараа нь хөтчөөр `http://localhost:3020` — эхний удаад `/setup` wizard
админ хэрэглэгчээ үүсгүүлээд шууд эхэлнэ. Өөр юу ч тохируулах шаардлагагүй.

Docker-гүй бол:

```bash
make dev    # локал Postgres дээр DB/role үүсгээд, миграц + API + вэб асаана
```

## Бүтэц

```
backend/
  cmd/api/           HTTP API
  cmd/migrate/       goose миграц
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
