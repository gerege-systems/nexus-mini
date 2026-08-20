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
nexus-mini add io.gerege.devices   # registry-ээс метадата татна → go get → бүртгэнэ → rebuild
```

Runtime plugin/WASM хийхгүй — ойлгомжтой байдал нь mini-гийн гол чанар.

## Store топологи: төв registry + локал fallback

- Төв registry (бид runestone дээр ажиллуулна) — модулийн каталог,
  **гарын үсэгтэй JSON** тараана
- Инстанс бүр `REGISTRY_URL`-ээс каталог татна; тохируулаагүй/офлайн бол
  `catalog/apps.json` файлаараа ажиллана
- Community хөгжүүлэгч модулиа төв registry рүү нийтэлнэ

## Хэрэглэгчийн 3 шаардлага (2026-08-20)

1. **Эхний ажиллуулалт: бүх тохиргоо env файлд** — DB холболтууд +
   `ADMIN_EMAIL/ADMIN_NAME/ADMIN_PASSWORD`-оо `nexus-mini.env`-д бичээд
   `nexus-mini migrate` (миграц + админ байхгүй бол env-ээс үүсгэнэ) →
   `nexus-mini serve`. Вэб талд setup wizard БАЙХГҮЙ, landing төлөв
   шалгадаггүй. Түүх: эхлээд вэб wizard хийсэн → хэрэглэгч буруушааж CLI
   болгосон → интерактив `setup` коммандыг ч илүүц гэж үзээд env-д
   суурилсан энэ хэлбэрт тогтсон (2026-08-20).
2. **Админ панель** — платформын админд хэрэгтэй бүх тохиргоо нэг дор
3. **Landing page** — цөм ба app store яаж ажилладаг, модуль яаж хөгжүүлэх нь
   бүгд тодорхой; тэндээсээ өөрийгөө + tenant-аа бүртгүүлээд орно; шинэ tenant
   эхлээд app store-оо хараад модуль суулгахаас эхэлдэг. "Юу болоод байгаа нь
   мэдэгддэггүй, аль хэдийн ажиллаад байдаг ойлгомжгүй юм" байж БОЛОХГҮЙ.

## Frontend: нэг Next.js апп

Landing + tenant portal + админ панель (`/admin`) нэг аппд. Self-host хийхэд
нэг л процесс — mini гэдгийн утга. UI chrome нь open-gerege-nexus-ийн
дизайныг жишиг болгоно (өөрөө зохиохгүй — өмнөх сургамж #10).

## Жишээ модуль

**Төхөөрөмжийн бүртгэл** (`devices`) — SDK-г баталдаг анхны модуль.

## Үе шат

| Үе | Агуулга |
|---|---|
| 1 | Цөм: миграц + CLI (migrate/admin/serve) + session auth + tenant + RBAC + audit + module SDK + devices модуль + app store (локал каталог) + landing/portal/админ панель |
| 2 | Төв registry + гарын үсэгтэй каталог + `nexus-mini add` CLI |
| 3 | OIDC provider + SSO client federation |
| 4 | Resilience давхарга, чанаржуулалт |

Үе бүр дуусаад **ажиллаж баталгаажсаны дараа** л дараагийнх эхэлнэ
(өмнөх сургамж #11 — бүгдийг зэрэг хийхгүй).

## Deploy (runestone)

| Хэсэг | Хаяг | Порт |
|---|---|---|
| Web + API (нэг домэйн: /api/* → Go) | nexus-mini.runestonetechnologies.com | 3020 / 8084 |
| Registry (үе 2) | nexus-mini-registry.runestonetechnologies.com | 8085 |

Хуучин nexus-ийн deploy 2026-08-20-нд бүрэн устгагдсан, портууд чөлөөтэй.
