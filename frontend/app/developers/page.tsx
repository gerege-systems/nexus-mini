"use client";

import Link from "next/link";
import { Button, Card, Icons } from '@gerege-systems/ui';
import { MktHeader, MktFooter } from "@/components/mkt";
import { useT } from "@/lib/i18n";

// Модуль хөгжүүлэх бүрэн гарын авлага — docs/03-module-guide.md-ийн вэб
// хувилбар. Агуулгыг өөрчилбөл хоёуланг нь синк байлга.
function Code({ children }: { children: React.ReactNode }) {
  return <div className="doc-code mb-5">{children}</div>;
}

export default function DevelopersPage() {
  const { t } = useT();
  return (
    <div className="flex min-h-dvh flex-col">
      <MktHeader />

      <main className="doc mx-auto w-full max-w-4xl flex-1 px-4 py-12 md:px-6 md:py-16">
      <section>
        <h1 className="text-foreground text-3xl font-semibold text-balance">{t("Хөгжүүлэгчийн гарын авлага")}</h1>
        <p>{t("Локал ажиллуулахаас эхлээд модуль, UI, өөрийн дистрибуц, registry, гадны системтэй OIDC-ээр холбох, цөмд хувь нэмэр оруулах хүртэл — нэг хуудсанд.")}</p>
        <nav className="doc-toc" aria-label={t("Агуулга")}>
          {[["#start", "0. Эхлэх"], ["#module", "1–5. Модуль (backend)"], ["#ui", "6. UI"], ["#register", "7–8. Бүртгэх · Registry"], ["#dist", "Өөрийн дистрибуц"], ["#oidc", "Гадны систем холбох (OIDC/SSO)"], ["#security", "Аюулгүй байдлын дүрэм"], ["#contrib", "Цөмд хувь нэмэр"], ["#test", "Тест · Deploy"]].map(([h, l]) => (
            <a key={h} href={h}>{t(l)}</a>
          ))}
        </nav>
        <p>
          {t("Модуль бол")} <code>pkg/nexus.Module</code> {t("interface-ийг хэрэгжүүлсэн Go package.")}{" "}
          {t("Tenant тусгаарлалт, нэвтрэлт, суулгалт, RBAC оноолт, audit — платформ хийнэ; та бизнес логикоо л бичнэ. Хамгийн сайн заавар бол ажиллаж байгаа жишээ —")}{" "}
          <a href="https://github.com/gerege-systems/nexus-mini/tree/main/backend/apps/devices">apps/devices</a> ({t("нэг resource")}),{" "}
          <a href="https://github.com/gerege-systems/nexus-mini/tree/main/backend/apps/organisation">apps/organisation</a> ({t("олон resource, олон хуудас")}).
        </p>
      </section>

      <section>
        <h2 id="start"><Icons.Zap className="size-5 shrink-0" aria-hidden /> {t("0. Эхлэх — локал ажиллуулах")}</h2>
        <Code>
          <div><span className="c">$</span> git clone https://github.com/gerege-systems/nexus-mini &amp;&amp; cd nexus-mini</div>
          <div><span className="c">$</span> psql -v owner_pw=… -v app_pw=… -v admin_pw=… -v auth_pw=… -f deploy/01-roles.sql&nbsp;&nbsp;<span className="c">{t("# нэг удаа, superuser")}</span></div>
          <div><span className="c">$</span> cp .env.example backend/nexus-mini.env&nbsp;&nbsp;<span className="c">{t("# 4 DB URL + ADMIN_EMAIL/NAME/PASSWORD + PORTAL_URL")}</span></div>
          <div><span className="c">$</span> make migrate &amp;&amp; make serve&nbsp;&nbsp;<span className="c">{t("# API :8084, анхны платформ админ env-ээс")}</span></div>
          <div><span className="c">$</span> cd frontend &amp;&amp; pnpm install &amp;&amp; cd .. &amp;&amp; make web&nbsp;&nbsp;<span className="c">{t("# portal :3020;  make admin — :3021")}</span></div>
          <div><span className="c">$</span> docker compose up -d&nbsp;&nbsp;<span className="c">{t("# эсвэл бүгд нэг коммандаар")}</span></div>
        </Code>
        <p>{t("Бүх команд Makefile-аар (make help). Push-ийн өмнө make check — linux build + vet + test + SDK-ийн хил. Хөгжүүлэлтийн төрлүүд:")}</p>
        <ul>
          <li><b>{t("Модуль")}</b> — {t("бизнес функц (энэ репогийн apps/ эсвэл өөрийн репо); доорх §1–8")}</li>
          <li><b>{t("Дистрибуц")}</b> — {t("өөрийн компанид nexus-mini ажиллуулах, модулиуд сонгох (nexus CLI)")}</li>
          <li><b>{t("Интеграц")}</b> — {t("гадны систем nexus-mini-ээр нэвтрэх / nexus-mini Google, SSO-оор нэвтрэх (OIDC)")}</li>
          <li><b>{t("Цөм")}</b> — {t("backend/internal, frontend, admin — PR-аар (доод хэсэг)")}</li>
        </ul>

        <h2 id="module">{t("Хэн юу хариуцдаг вэ")}</h2>
        <div className="border-border my-6 overflow-x-auto rounded-lg border">
          <table>
            <thead><tr><th>{t("МОДУЛЬ")}</th><th>{t("ПЛАТФОРМ")}</th></tr></thead>
            <tbody>
              <tr><td>{t("Permission-оо тунхаглана")}</td><td>{t("Tenant тусгаарлалт (RLS)")}</td></tr>
              <tr><td>{t("Цэсээ зарлана")}</td><td>{t("Нэвтрэлт, session")}</td></tr>
              <tr><td>{t("Route-уудаа бүртгэнэ")}</td><td>{t("Суулгалт, хамаарлын шийдэл")}</td></tr>
              <tr><td>{t("Өөрийн хүснэгт, миграц")}</td><td>{t("RBAC default оноолт, шалгалт")}</td></tr>
              <tr><td>{t("Бизнес логик")}</td><td>{t("Audit гинж, app store")}</td></tr>
              <tr><td>{t("Хувилбар + манифест (make manifest)")}</td><td>{t("Registry, гарын үсэг, nexus add/upgrade")}</td></tr>
              <tr><td>—</td><td>{t("Түдгэлзүүлэлт / зөвхөн-унших, impersonation, lockout")}</td></tr>
            </tbody>
          </table>
        </div>

        <h2><Icons.Folder className="size-5 shrink-0" aria-hidden /> {t("Файлын бүтэц")}</h2>
        <p>{t("Жижиг модуль нэг файлаас эхэлж болно; өсөхөөрөө ингэж хуваана:")}</p>
        <Code>
          <div>backend/apps/{"<name>"}/</div>
          <div>&nbsp;&nbsp;module.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("модулийн ГЭРЭЭ: ID, permission, цэс, миграц, route↔permission холболт")}</span></div>
          <div>&nbsp;&nbsp;types.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("хүсэлт/хариултын struct + validation")}</span></div>
          <div>&nbsp;&nbsp;handlers.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("HTTP handler-ууд (нэг resource = нэг файл)")}</span></div>
          <div>&nbsp;&nbsp;migrations/&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("модулийн goose миграцууд")}</span></div>
          <div>&nbsp;&nbsp;ui/pages/&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("portal хуудсууд → build үед app/(portal)/<нэр>/ руу хуулагдана")}</span></div>
          <div>&nbsp;&nbsp;ui/i18n.ts&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("модулийн толь (en: {...}) — цөмийн толинд нэгдэнэ")}</span></div>
        </Code>
        <p>{t("Олон resource-тэй бол handler файлыг resource тус бүрээр салгана (organisation: departments.go, people.go; ui/pages/departments/, ui/pages/people/).")}</p>

        <h2><Icons.Zap className="size-5 shrink-0" aria-hidden /> {t("Алхамууд")}</h2>

        <h3 style={{ marginTop: "1.5rem" }}>{t("1. Package үүсгэх")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) ID() <span className="k">string</span>      {"{"} <span className="k">return</span> <span className="s">&quot;mn.yourorg.name&quot;</span> {"}"} <span className="c">{t("// reverse-DNS, глобал давтагдашгүй")}</span></div>
          <div><span className="k">func</span> (m *Module) ShortID() <span className="k">string</span> {"{"} <span className="k">return</span> <span className="s">&quot;name&quot;</span> {"}"}          <span className="c">{t("// permission prefix + URL зам")}</span></div>
          <div><span className="k">func</span> (m *Module) Name() <span className="k">string</span>    {"{"} <span className="k">return</span> <span className="s">&quot;{t("Хүний нэр")}&quot;</span> {"}"}</div>
          <div><span className="k">func</span> (m *Module) Version() <span className="k">string</span> {"{"} <span className="k">return</span> <span className="s">&quot;1.0.0&quot;</span> {"}"}</div>
          <div>&nbsp;</div>
          <div><span className="c">{t("// заавал биш — store/registry-ийн тайлбар кодоос (make manifest авдаг):")}</span></div>
          <div><span className="k">func</span> (m *Module) Description() <span className="k">string</span> {"{"} <span className="k">return</span> <span className="s">&quot;{t("Юу хийдэг вэ…")}&quot;</span> {"}"}</div>
          <div><span className="k">func</span> (m *Module) Publisher() <span className="k">string</span>   {"{"} <span className="k">return</span> <span className="s">&quot;your-org&quot;</span> {"}"}</div>
        </Code>
        <p>{t("Version нь semver; registry-д нийтлэх git tag нь v<Version>. Permission нэмсэн/өргөсгөсөн бол minor-оо өсгө — дистрибуц nexus upgrade хийхэд -approve асуух шалтгаан нь энэ.")}</p>

        <h3>{t("2. Permission тунхаглах")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) Permissions() []nexus.PermissionDefinition {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">return</span> []nexus.PermissionDefinition{"{"}</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;{"{"}Code: <span className="s">&quot;name.read&quot;</span>, DefaultRoles: []<span className="k">string</span>{"{"}<span className="s">&quot;manager&quot;</span>, <span className="s">&quot;user&quot;</span>{"}}"},</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;{"{"}Code: <span className="s">&quot;name.manage&quot;</span>, OwnScope: <span className="k">true</span>, DefaultRoles: []<span className="k">string</span>{"{"}<span className="s">&quot;manager&quot;</span>, <span className="s">&quot;user:own&quot;</span>{"}}"},</div>
          <div>&nbsp;&nbsp;{"}"}</div>
          <div>{"}"}</div>
        </Code>
        <p>{t("Дүрмүүд (зөрчвөл бинари асахгүй):")}</p>
        <ul>
          <li>{t("Код заавал")} <code>&lt;ShortID&gt;.</code>{t("-ээр эхэлнэ — өөр модулийн эрхийг булааж чадахгүй")}</li>
          <li><code>DefaultRoles</code> {t("нь суулгах үед хэн авахыг тунхагладаг:")} <code>admin</code> {t("үргэлж бүгдийг авна, жагсаалтад бичсэн нь нэмж авна,")} <code>&quot;user:own&quot;</code> {t("нь зөвхөн өөрийн мөрийн эрх")}</li>
          <li><code>DefaultRoles</code> {t("хоосон = зөвхөн admin (аюулгүй default)")}</li>
          <li><code>&quot;role:own&quot;</code> {t("бичихийн тулд permission")} <code>OwnScope: true</code> {t("байх ёстой (үгүй бол panic). Runtime-д ч own_scope=false permission-д хэн ч «own» өгч чадахгүй")}</li>
          <li>{t("Нөөцөлсөн ShortID:")} <code>core api admin platform store apps developers login signup dashboard members roles audit settings org</code></li>
          <li>{t("Шинэ хувилбарт permission нэмбэл цөм асахдаа суусан tenant бүрийн admin-д (+DefaultRoles) автоматаар оноодог (backfill); байгаа кодод хүрэхгүй")}</li>
        </ul>

        <h3>{t("3. Миграц")}</h3>
        <Code>
          <div><span className="c">//go:embed migrations/*.sql</span></div>
          <div><span className="k">var</span> migrations embed.FS</div>
          <div><span className="k">func</span> (m *Module) Migrations() fs.FS {"{"} <span className="k">return</span> migrations {"}"}</div>
        </Code>
        <ul>
          <li><code>tenant_id uuid NOT NULL</code> + {t("RLS policy")} (<code>app_tenant_id()</code>) — {t("жишээг devices-ээс хуул")}</li>
          <li><code>OwnScope</code> {t("ашиглах бол")} <code>created_by uuid</code> {t("багана заавал")}</li>
          <li>{t("Бүх string баганад урттай хязгаар (varchar(n)) — задгай text хориотой")}</li>
          <li>{t("Төгсгөлд нь")} <code>GRANT ... ON &lt;table&gt; TO nexus_app, nexus_admin</code> {t("(функцэд автомат GRANT байхгүй)")}</li>
          <li>{t("Өөр хүснэгт рүү FK (memberships, өөрийн мод) заавал same-tenant trigger-тэй — FK шалгалт RLS-ийг давдаг. Загвар:")} <code>apps/organisation/migrations/00002_same_tenant.sql</code></li>
          <li>{t("Апп role-д temp хүснэгт, users.password_hash, auth_* функцууд хаалттай — зориуд")}</li>
        </ul>
        <p>
          {t("Модуль бүр өөрийн goose хүснэгттэй")} (<code>goose_&lt;shortid&gt;</code>) {t("тул цөм болон бусад модультай мөргөлдөхгүй.")}
        </p>

        <h3>{t("4. Route-ууд")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) RegisterRoutes(r chi.Router, deps nexus.Deps) {"{"}</div>
          <div>&nbsp;&nbsp;h := &amp;handler{"{"}deps: deps{"}"}</div>
          <div>&nbsp;&nbsp;r.With(nexus.RequirePermission(deps.Perms, <span className="s">&quot;name.read&quot;</span>)).Get(<span className="s">&quot;/&quot;</span>, h.list)</div>
          <div>&nbsp;&nbsp;r.With(nexus.RequirePermission(deps.Perms, <span className="s">&quot;name.manage&quot;</span>)).Post(<span className="s">&quot;/&quot;</span>, h.create)</div>
          <div>{"}"}</div>
        </Code>
        <p>
          {t("Танд өгөгдөх")} <code>r</code> {t("нь аль хэдийн хамгаалагдсан:")} <code>/api/apps/&lt;ShortID&gt;/</code>{" "}
          {t("дор байрладаг, нэвтрээгүй хүн 401, апп суулгаагүй tenant 403 авчихсан байдаг. Handler дотор:")}
        </p>
        <ul>
          <li><code>nexus.TenantID(ctx)</code>, <code>nexus.UserID(ctx)</code> — {t("хүсэлтийн identity")}</li>
          <li><code>nexus.Scope(ctx)</code> — <code>ScopeOwn</code> {t("бол query-дээ")} <code>created_by</code> {t("шүүлт нэм")}</li>
          <li><code>deps.DB</code> — {t("RLS context автоматаар тохирдог холболт; SQL-даа")} <code>tenant_id = $1</code> {t("гэж бас бич")}</li>
          <li><code>deps.Audit.Record(ctx, ...)</code> — {t("чухал үйлдлээ audit гинжид бич")}</li>
          <li><code>nexus.JSON / Decode / Error / DBError</code> — {t("вэб туслахууд")}</li>
          <li><code>nexus.UUIDParam(w, r, &quot;id&quot;)</code> / <code>nexus.IsUUID</code> — {t("зам/биеийн id-г DB-д хүргэхээс өмнө (буруу бол 400); string талбарын уртыг valid()-даа шалга")}</li>
          <li>{t("Түдгэлзүүлсэн байгууллага → 403, зөвхөн-унших → бичих 503: платформ RequireTenant-д хийнэ, модуль мэдэх шаардлагагүй; impersonated session-ийн audit-д impersonated_by автоматаар хавсарна")}</li>
        </ul>

        <h3>{t("5. Цэс")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) Menus() []nexus.MenuDefinition {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">return</span> []nexus.MenuDefinition{"{{"}</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;ID: <span className="s">&quot;name.list&quot;</span>, Label: <span className="s">&quot;{t("Монгол нэр")}&quot;</span>, Labels: <span className="k">map</span>[<span className="k">string</span>]<span className="k">string</span>{"{"}<span className="s">&quot;en&quot;</span>: <span className="s">&quot;English&quot;</span>{"}"},</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;Path: <span className="s">&quot;/name&quot;</span>, Icon: <span className="s">&quot;device&quot;</span>, Order: 10,</div>
          <div>&nbsp;&nbsp;{"}}"}</div>
          <div>{"}"}</div>
        </Code>
        <p>{t("Path заавал /<ShortID> эсвэл /<ShortID>/… — өөр зам Register panic (portal-ийн middleware нийтийн замаас бусдыг хамгаалдаг, модуль тойрч чадахгүй). Icon нэр: components/icons.tsx.")}</p>

        <h3 id="ui">{t("6. UI хуудас (portal)")}</h3>
        <p>
          {t("UI нь модулийн хавтаст амьдарна: ui/pages/ доторх хуудсууд build үед app/(portal)/<нэр>/ руу хуулагдана, ui/i18n.ts толь цөмийнхтэй нэгдэнэ — цөмийн frontend файлд гар хүрэхгүй. Бэлэн загвар")}{" "}
          (<a href="https://github.com/gerege-systems/nexus-mini/blob/main/backend/apps/devices/ui/pages/page.tsx">apps/devices/ui/pages/page.tsx</a>,{" "}
          <a href="https://github.com/gerege-systems/nexus-mini/tree/main/backend/apps/organisation/ui">apps/organisation/ui</a>).
        </p>
        <Code>
          <div><span className="c">{t("// apps/name/ui/pages/page.tsx → /name (ui/pages/reports/page.tsx → /name/reports)")}</span></div>
          <div><span className="s">&quot;use client&quot;</span>;</div>
          <div>&nbsp;</div>
          <div><span className="k">export default function</span> NamePage() {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">const</span> {"{"} me {"}"} = useShell();&nbsp;&nbsp;<span className="c">{t("// хэрэглэгч + permissions")}</span></div>
          <div>&nbsp;&nbsp;<span className="k">const</span> {"{"} t {"}"} = useT();&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("// хэл (mn/en)")}</span></div>
          <div>&nbsp;&nbsp;<span className="k">const</span> manage = me.permissions[<span className="s">&quot;name.manage&quot;</span>]; <span className="c">{t("// undefined | \"all\" | \"own\"")}</span></div>
          <div>&nbsp;&nbsp;<span className="c">{t("// api.get(`/api/apps/name/`) — cookie автоматаар, 401 бол login руу")}</span></div>
          <div>{"}"}</div>
        </Code>
        <ul>
          <li>{t("Эрхээр UI-гаа нуу:")} <code>me.permissions[&quot;name.manage&quot;]</code> {t("байхгүй бол товчоо бүү харуул (энэ нь UX — жинхэнэ хамгаалалт серверт)")}</li>
          <li>{t("«Өөрийн» scope-той хэрэглэгчид засах/устгах товчийг")} <code>created_by === me.user.id</code> {t("үед л харуулна")}</li>
          <li>{t("Цэсний icon нэрээ")} <code>components/icons.tsx</code>-{t("ийн map-д нэм (lucide icon)")}</li>
          <li>{t("Бэлэн загварууд:")} <code>card / table / btn / field / badge / modal</code> — globals.css; {t("амжилтад")} <code>toast(...)</code>, {t("текстэд")} <code>t(...)</code></li>
          <li>{t("Толь:")} <code>ui/i18n.ts</code> — <code>{"{ en: { \"Төхөөрөмжүүд\": \"Devices\" } }"}</code> {t("(түлхүүр нь монгол текст)")}</li>
        </ul>

        <h3 id="register">{t("7. Бүртгэх ба асаах")}</h3>
        <Code>
          <div><span className="c">{t("// backend/apps/apps.go — бинарид орох модулиуд:")}</span></div>
          <div><span className="k">func</span> All() []nexus.Module {"{"} <span className="k">return</span> []nexus.Module{"{"} devices.New(), name.New() {"}"} {"}"}</div>
          <div>&nbsp;</div>
          <div><span className="c">{t("// frontend/modules.json — portal-д орох UI:")}</span></div>
          <div>{"{"} <span className="s">&quot;short_id&quot;</span>: <span className="s">&quot;name&quot;</span>, <span className="s">&quot;ui&quot;</span>: <span className="s">&quot;../backend/apps/name/ui&quot;</span> {"}"}</div>
          <div>&nbsp;</div>
          <div><span className="c">$</span> make migrate &amp;&amp; make serve&nbsp;&nbsp;<span className="c">{t("# модуль store-д гарч ирнэ")}</span></div>
        </Code>

        <h3>{t("8. Store-д нийтлэх")}</h3>
        <p>
          {t("Манифест кодоос үүснэ:")} <code>make manifest MOD=name &gt; manifests/name.json</code> — {t("дараа нь")}{" "}
          <a href="https://github.com/gerege-systems/nexus-registry">nexus-registry</a>{" "}
          {t("репод PR илгээнэ; maintainer index.json-ийг Ed25519-ээр гарын үсэглэнэ. Код registry-д хадгалагдахгүй — go_module зам + git tag хангалттай. Орсны дараа хэн ч")} <code>nexus add name</code> {t("гэж дистрибуцдаа нэмнэ. Өөрийн registry: репог хуулж, nexus-registry keygen → REGISTRY_URL + REGISTRY_KEYS.")}
        </p>

        <h2 id="dist" style={{ marginTop: "2rem" }}><Icons.Github className="size-5 shrink-0" aria-hidden /> {t("Өөрийн дистрибуц — цөмийг fork хийхгүй")}</h2>
        <p>
          {t("Өөрийн компанид nexus-mini ашиглаж, өөрийн модулиуд, өөрийн store, өөрийн харилцагчидтай (tenant) платформ ажиллуулж болно. Цөмийн репог хуулбарлаж засахгүй — хамаарал болгоно. Ингэж байж цөмийн шинэчлэлтийг merge-гүй, мөргөлдөөнгүй авна.")}
        </p>
        <Code>
          <div><span className="c">{t("// nexus CLI — дистрибуц үүсгэх, модуль нэмэх (цөмийг fork хийхгүй):")}</span></div>
          <div><span className="c">$</span> go run github.com/gerege-systems/nexus-mini/backend/cmd/nexus@latest init my-dist</div>
          <div><span className="c">$</span> cd my-dist &amp;&amp; go run github.com/gerege-systems/nexus-mini/backend/cmd/nexus@latest add organisation</div>
          <div><span className="c">$</span> make migrate &amp;&amp; make serve</div>
          <div>&nbsp;</div>
          <div><span className="c">{t("// backend/main.go (init үүсгэнэ; add маркер хооронд мөр нэмнэ):")}</span></div>
          <div><span className="k">func</span> main() {"{"} core.Main(modules()...) {"}"}</div>
          <div>&nbsp;</div>
          <div><span className="c">{t("// Цөмийг шинэчлэх — merge байхгүй, зөвхөн хувилбар:")}</span></div>
          <div><span className="c">$</span> go get github.com/gerege-systems/nexus-mini/backend@v1.5.0</div>
          <div><span className="c">$</span> git fetch upstream --tags &amp;&amp; git checkout backend/v1.5.0 -- frontend</div>
        </Code>
        <ul>
          <li><code>core.Main</code> {t("— migrate/serve/manifest коммандууд, env, миграц, анхны админ, permission sync, сервер: бүгд цөмд; та модулиудаа л өгнө")}</li>
          <li><code>nexus upgrade</code> {t("— модулийн шинэ хувилбарт permission нэмэгдсэн/өргөссөн бол зогсоож -approve шаардана; модуль чимээгүй эрх авахгүй")}</li>
          <li>{t("Frontend: цөмийн frontend-ийн хуулбар +")} <code>modules.json</code>. {t("Та цөмийн файлд гар хүрдэггүй (UI ui/-д, толь ui/i18n.ts-д) тул цөмийн frontend-ийг tag-аас хуулж дарахад мөргөлдөхгүй")}</li>
          <li>{t("SDK амлалт:")} <code>pkg/nexus</code> + <code>core.Main</code> {t("v1.x дотор эвдэхгүй (semver); internal/* чөлөөтэй өөрчлөгдөнө — модуль түүнээс импортолж чадахгүй")}</li>
          <li>{t("Цөмд алдаа олбол өөр дээрээ засахгүй — upstream руу PR. Харилцагч тань таны instance дээр tenant болно; та платформ админ")}</li>
        </ul>

        <h2 id="oidc" style={{ marginTop: "2rem" }}><Icons.Key className="size-5 shrink-0" aria-hidden /> {t("Гадны систем холбох — OIDC provider, SSO, federation")}</h2>
        <p>
          {t("nexus-mini нь OpenID Connect provider (таны систем энэ платформын бүртгэлээр нэвтэрнэ) ба relying party (энэ платформ Google/өөр issuer-ээр нэвтэрнэ) хоёулаа. Хоёр nexus-mini хоорондоо = federation.")}
        </p>
        <h3>{t("Таны систем nexus-mini-ээр нэвтрэх")}</h3>
        <ol style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li>{t("Portal → SSO клиентүүд (core.sso.manage) → Клиент нэмэх: нэр, redirect URI (https/localhost), scope. Confidential бол client_secret нэг л удаа харагдана; SPA/mobile бол Public (PKCE).")}</li>
          <li>{t("Discovery:")} <code>&lt;PORTAL_URL&gt;/api/oauth2/.well-known/openid-configuration</code> — {t("дурын OIDC номын сан (openid-client, oidc-client-ts, Spring, NextAuth…) үүгээр бүх endpoint-ийг олно.")}</li>
          <li>{t("Урсгал: authorization_code + PKCE S256 заавал. Хэрэглэгч portal-д нэвтэрч, клиентийн байгууллагын гишүүн бол consent → code → /token. Зөвшөөрөл санагдана.")}</li>
          <li>{t("Токен: access opaque (/introspect, /revoke), id_token RS256 (/jwks), offline_access → refresh (rotation, replay бол гэр бүлээр хүчингүй). Claims: sub, name, email, tenant (slug) + tenant_id, roles.")}</li>
          <li>{t("Сервер-сервер: client_credentials → tenant scope-той access token. Гарах: end_session?id_token_hint=…")}</li>
        </ol>
        <Code>
          <div><span className="c">{t("// 1. browser → authorize (PKCE):")}</span></div>
          <div>&lt;issuer&gt;/authorize?response_type=code&amp;client_id=…&amp;redirect_uri=…&amp;scope=openid%20profile%20email&amp;state=…&amp;nonce=…&amp;code_challenge=…&amp;code_challenge_method=S256</div>
          <div><span className="c">{t("// 2. callback-ийн code-оор токен:")}</span></div>
          <div><span className="c">$</span> curl -u &quot;$CLIENT_ID:$CLIENT_SECRET&quot; -X POST &lt;issuer&gt;/token -d grant_type=authorization_code -d code=… -d redirect_uri=… -d code_verifier=…</div>
          <div><span className="c">{t("// 3. шалгах / хүчингүй болгох:")}</span></div>
          <div><span className="c">$</span> curl -u … -X POST &lt;issuer&gt;/introspect -d token=…&nbsp;&nbsp;&nbsp;<span className="c">$</span> curl -u … -X POST &lt;issuer&gt;/revoke -d token=…</div>
        </Code>
        <h3>{t("nexus-mini Google / өөр OIDC-ээр нэвтрэх (env)")}</h3>
        <Code>
          <div>GOOGLE_CLIENT_ID=… GOOGLE_CLIENT_SECRET=…&nbsp;&nbsp;<span className="c">{t("# redirect: <PORTAL_URL>/api/auth/sso/google/callback")}</span></div>
          <div>SSO_ISSUER=https://nexus.bold.mn/api/oauth2 SSO_CLIENT_ID=… SSO_CLIENT_SECRET=… SSO_NAME=&quot;Bold SSO&quot;&nbsp;&nbsp;<span className="c">{t("# federation")}</span></div>
          <div>SSO_AUTO_SIGNUP=false&nbsp;&nbsp;<span className="c">{t("# true: танигдаагүй имэйлд данс үүсгэнэ (JIT)")}</span></div>
        </Code>
        <p>{t("Login хуудсанд товч гарна; PKCE + state + nonce, id_token-ийг issuer-ийн JWKS-ээр шалгана. Дэлгэрэнгүй:")} <a href="https://github.com/gerege-systems/nexus-mini/blob/main/docs/04-integrations.md">docs/04-integrations.md</a>.</p>

        <h2 id="security" style={{ marginTop: "2rem" }}><Icons.Lock className="size-5 shrink-0" aria-hidden /> {t("Аюулгүй байдлын дүрэм — модуль, цөм хоёуланд")}</h2>
        <ul>
          <li>{t("Эрх зөвхөн серверт: route бүр RequirePermission; UI-гийн нуулт нь UX. Мөрийн түвшин — RLS + query-дээ tenant_id, own scope бол created_by.")}</li>
          <li>{t("SQL үргэлж параметртэй, төрлийн cast-тай ($1::uuid). Бүх string багана varchar(n); FK бусад хүснэгт рүү → same-tenant trigger.")}</li>
          <li>{t("Алдааг клиентэд түүхийгээр нь буцаахгүй (DBError/Error): 23505 → 409, бусад → лог + ерөнхий 500. Буруу uuid → 400 (UUIDParam).")}</li>
          <li>{t("Нууц зүйл (токен, нууц үг, session) лог/audit-д бичихгүй. Client secret argon2-оор hash-лагдсан; OAuth токен sha256 hash-аар хадгалагдана.")}</li>
          <li>{t("Апп DB role-д: temp хүснэгт үгүй, auth_* функц үгүй, users.password_hash үгүй, tenants төлөв багана үгүй — модуль эдгээрт хүрч чадахгүй, хүрэх гэж бүү оролд.")}</li>
          <li>{t("Cookie-тэй бичих хүсэлт Origin + Sec-Fetch-Site шалгалттай; гадны домэйноос дуудах endpoint бол токен-аар танигддаг байх (OAuth2 загвар).")}</li>
          <li>{t("Render дотор window/localStorage/matchMedia уншихгүй (hydration) — useEffect-д. Шалгахдаа dark/light хоёуланг.")}</li>
        </ul>

        <h2 id="contrib" style={{ marginTop: "2rem" }}><Icons.Github className="size-5 shrink-0" aria-hidden /> {t("Цөмд хувь нэмэр оруулах")}</h2>
        <ul>
          <li>{t("Бүтэц:")} <code>backend/internal/core/*</code> {t("(цөмийн дотоод — чөлөөтэй өөрчлөгдөнө),")} <code>backend/pkg/nexus</code> + <code>backend/core.Main</code> {t("(SDK — v1.x-д эвдэхгүй, эвдэх бол major),")} <code>backend/pkg/registry</code> {t("(registry гэрээ),")} <code>frontend/</code>, <code>admin/</code>.</li>
          <li>{t("Миграц: backend/db/migrations/000NN_*.sql (goose, Up/Down хоёулаа); definer функц бүр SET search_path = pg_catalog, public, pg_temp + REVOKE FROM PUBLIC + тодорхой GRANT.")}</li>
          <li>{t("Тест: unit (go test) + env-гэйт integration (NEXUS_TEST_DATABASE_URL*); RLS/RBAC өөрчлөлтөд integration заавал. make check push бүрийн өмнө.")}</li>
          <li>{t("Баримт: docs/00-decisions.md-д шийдвэрээ, docs/03 ↔ энэ хуудас синк, CLAUDE.md-ийн invariant-ууд.")}</li>
          <li>{t("PR: жижиг, нэг зорилготой; commit-д «яагаад». Алдаа олбол fork-доо биш upstream-д.")}</li>
        </ul>

        <h2 id="test"><Icons.FileText className="size-5 shrink-0" aria-hidden /> {t("Тест · Deploy")}</h2>
        <p>
          {t("SQL parse/encode бүх логикт unit тест бич.")} <code>make check</code> {t("нь linux cross-build + vet + test + SDK-ийн хилийн шалгалт (модуль internal/* импортолбол унадаг) — push бүрийн өмнө заавал.")}{" "}
          {t("Deploy: сервер дээр git pull → deploy/deploy.sh (make migrate ENV_FILE=… + атом бинари/Next солилт + systemd restart); unit/nginx өөрчлөлт гараар. Docker: docker-compose.yml. Production: ENVIRONMENT=production, PORTAL_URL https, nexus_auth role, ADMIN_* env-ээс анхны админ үүссэний дараа устга.")}
        </p>

        <Card className="mt-8 flex flex-wrap items-center gap-4">
          <span className="bg-background-muted text-foreground-muted grid size-10 shrink-0 place-items-center rounded-md"><Icons.Github className="size-5" aria-hidden /></span>
          <div className="min-w-[15rem] flex-1">
            <p className="text-foreground font-medium">{t("Бэлэн үү?")}</p>
            <p className="text-foreground-muted text-sm">
              {t("devices-ийг хуулж эхлээд, дуусаад каталогт PR илгээгээрэй — эсвэл өөрийн дистрибуц үүсгээрэй.")}
            </p>
          </div>
          <Button variant="secondary" asChild>
            <a href="https://github.com/gerege-systems/nexus-mini/blob/main/docs/03-module-guide.md">{t("Markdown хувилбар")}</a>
          </Button>
          <Button asChild>
            <Link href="/apps">{t("Апп дэлгүүр үзэх")}</Link>
          </Button>
        </Card>
      </section>
      </main>

      <MktFooter />
    </div>
  );
}
