"use client";

import Link from "next/link";
import { BookOpen, FolderTree, GitPullRequest, LayoutTemplate, Rocket } from "lucide-react";
import { MktHeader, MktFooter } from "@/components/mkt";
import { useT } from "@/lib/i18n";

// Модуль хөгжүүлэх бүрэн гарын авлага — docs/03-module-guide.md-ийн вэб
// хувилбар. Агуулгыг өөрчилбөл хоёуланг нь синк байлга.
function Code({ children }: { children: React.ReactNode }) {
  return <div className="mkt-code" style={{ marginBottom: "1.2rem" }}>{children}</div>;
}

export default function DevelopersPage() {
  const { t } = useT();
  return (
    <div className="mkt">
      <MktHeader />

      <section className="mkt-hero" style={{ paddingBottom: "1.5rem" }}>
        <h1>{t("Модуль хөгжүүлэх")}</h1>
        <p>
          {t("Модуль бол")} <code>pkg/nexus.Module</code> {t("interface-ийг хэрэгжүүлсэн Go package.")}{" "}
          {t("Tenant тусгаарлалт, нэвтрэлт, суулгалт, RBAC оноолт, audit — платформ хийнэ; та бизнес логикоо л бичнэ. Хамгийн сайн заавар бол ажиллаж байгаа жишээ —")}{" "}
          <a href="https://github.com/gerege-systems/nexus-mini/tree/main/backend/apps/devices"
            style={{ color: "var(--accent)", fontWeight: 600 }}>backend/apps/devices</a>.
        </p>
      </section>

      <section className="mkt-sect" style={{ paddingTop: 0 }}>
        <h2>{t("Хэн юу хариуцдаг вэ")}</h2>
        <div className="card" style={{ margin: "1rem 0 2rem", overflowX: "auto" }}>
          <table className="table">
            <thead><tr><th>{t("МОДУЛЬ")}</th><th>{t("ПЛАТФОРМ")}</th></tr></thead>
            <tbody>
              <tr><td>{t("Permission-оо тунхаглана")}</td><td>{t("Tenant тусгаарлалт (RLS)")}</td></tr>
              <tr><td>{t("Цэсээ зарлана")}</td><td>{t("Нэвтрэлт, session")}</td></tr>
              <tr><td>{t("Route-уудаа бүртгэнэ")}</td><td>{t("Суулгалт, хамаарлын шийдэл")}</td></tr>
              <tr><td>{t("Өөрийн хүснэгт, миграц")}</td><td>{t("RBAC default оноолт, шалгалт")}</td></tr>
              <tr><td>{t("Бизнес логик")}</td><td>{t("Audit гинж, app store")}</td></tr>
            </tbody>
          </table>
        </div>

        <h2><FolderTree size={19} style={{ verticalAlign: "-3px" }} /> {t("Файлын бүтэц")}</h2>
        <p className="lead">{t("Жижиг модуль нэг файлаас эхэлж болно; өсөхөөрөө ингэж хуваана:")}</p>
        <Code>
          <div>backend/apps/{"<name>"}/</div>
          <div>&nbsp;&nbsp;module.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("модулийн ГЭРЭЭ: ID, permission, цэс, миграц, route↔permission холболт")}</span></div>
          <div>&nbsp;&nbsp;types.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("хүсэлт/хариултын struct + validation")}</span></div>
          <div>&nbsp;&nbsp;handlers.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("HTTP handler-ууд (нэг resource = нэг файл)")}</span></div>
          <div>&nbsp;&nbsp;migrations/&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("модулийн goose миграцууд")}</span></div>
        </Code>

        <h2><Rocket size={19} style={{ verticalAlign: "-3px" }} /> {t("Алхамууд")}</h2>

        <h3 style={{ marginTop: "1.5rem" }}>{t("1. Package үүсгэх")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) ID() <span className="k">string</span>      {"{"} <span className="k">return</span> <span className="s">&quot;mn.yourorg.name&quot;</span> {"}"} <span className="c">{t("// reverse-DNS, глобал давтагдашгүй")}</span></div>
          <div><span className="k">func</span> (m *Module) ShortID() <span className="k">string</span> {"{"} <span className="k">return</span> <span className="s">&quot;name&quot;</span> {"}"}          <span className="c">{t("// permission prefix + URL зам")}</span></div>
          <div><span className="k">func</span> (m *Module) Name() <span className="k">string</span>    {"{"} <span className="k">return</span> <span className="s">&quot;{t("Хүний нэр")}&quot;</span> {"}"}</div>
          <div><span className="k">func</span> (m *Module) Version() <span className="k">string</span> {"{"} <span className="k">return</span> <span className="s">&quot;1.0.0&quot;</span> {"}"}</div>
        </Code>

        <h3>{t("2. Permission тунхаглах")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) Permissions() []nexus.PermissionDefinition {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">return</span> []nexus.PermissionDefinition{"{"}</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;{"{"}Code: <span className="s">&quot;name.read&quot;</span>, DefaultRoles: []<span className="k">string</span>{"{"}<span className="s">&quot;manager&quot;</span>, <span className="s">&quot;user&quot;</span>{"}}"},</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;{"{"}Code: <span className="s">&quot;name.manage&quot;</span>, OwnScope: <span className="k">true</span>, DefaultRoles: []<span className="k">string</span>{"{"}<span className="s">&quot;manager&quot;</span>, <span className="s">&quot;user:own&quot;</span>{"}}"},</div>
          <div>&nbsp;&nbsp;{"}"}</div>
          <div>{"}"}</div>
        </Code>
        <p style={{ color: "var(--text-2)" }}>{t("Дүрмүүд (зөрчвөл бинари асахгүй):")}</p>
        <ul style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li>{t("Код заавал")} <code>&lt;ShortID&gt;.</code>{t("-ээр эхэлнэ — өөр модулийн эрхийг булааж чадахгүй")}</li>
          <li><code>DefaultRoles</code> {t("нь суулгах үед хэн авахыг тунхагладаг:")} <code>admin</code> {t("үргэлж бүгдийг авна, жагсаалтад бичсэн нь нэмж авна,")} <code>&quot;user:own&quot;</code> {t("нь зөвхөн өөрийн мөрийн эрх")}</li>
          <li><code>DefaultRoles</code> {t("хоосон = зөвхөн admin (аюулгүй default)")}</li>
          <li><code>core</code>, <code>api</code>, <code>admin</code> {t("зэрэг нэрс нөөцлөгдсөн")}</li>
        </ul>

        <h3>{t("3. Миграц")}</h3>
        <Code>
          <div><span className="c">//go:embed migrations/*.sql</span></div>
          <div><span className="k">var</span> migrations embed.FS</div>
          <div><span className="k">func</span> (m *Module) Migrations() fs.FS {"{"} <span className="k">return</span> migrations {"}"}</div>
        </Code>
        <ul style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li><code>tenant_id uuid NOT NULL</code> + {t("RLS policy")} (<code>app_tenant_id()</code>) — {t("жишээг devices-ээс хуул")}</li>
          <li><code>OwnScope</code> {t("ашиглах бол")} <code>created_by uuid</code> {t("багана заавал")}</li>
          <li>{t("Бүх string баганад урттай хязгаар (varchar(n)) — задгай text хориотой")}</li>
          <li>{t("Төгсгөлд нь")} <code>GRANT ... TO nexus_app, nexus_admin</code></li>
        </ul>
        <p style={{ color: "var(--text-2)" }}>
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
        <p style={{ color: "var(--text-2)" }}>
          {t("Танд өгөгдөх")} <code>r</code> {t("нь аль хэдийн хамгаалагдсан:")} <code>/api/apps/&lt;ShortID&gt;/</code>{" "}
          {t("дор байрладаг, нэвтрээгүй хүн 401, апп суулгаагүй tenant 403 авчихсан байдаг. Handler дотор:")}
        </p>
        <ul style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li><code>nexus.TenantID(ctx)</code>, <code>nexus.UserID(ctx)</code> — {t("хүсэлтийн identity")}</li>
          <li><code>nexus.Scope(ctx)</code> — <code>ScopeOwn</code> {t("бол query-дээ")} <code>created_by</code> {t("шүүлт нэм")}</li>
          <li><code>deps.DB</code> — {t("RLS context автоматаар тохирдог холболт; SQL-даа")} <code>tenant_id = $1</code> {t("гэж бас бич")}</li>
          <li><code>deps.Audit.Record(ctx, ...)</code> — {t("чухал үйлдлээ audit гинжид бич")}</li>
          <li><code>nexus.JSON / Decode / Error / DBError</code> — {t("вэб туслахууд")}</li>
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

        <h3>{t("6. UI хуудас (portal)")}</h3>
        <p style={{ color: "var(--text-2)" }}>
          {t("Цэсэндээ зарласан Path-тайгаа ижил замд Next.js хуудас үүсгэнэ — devices-ийн хуудас")}{" "}
          (<a href="https://github.com/gerege-systems/nexus-mini/blob/main/frontend/app/(portal)/devices/page.tsx"
            style={{ color: "var(--accent)", fontWeight: 600 }}>frontend/app/(portal)/devices/page.tsx</a>){" "}
          {t("бэлэн загвар нь.")}
        </p>
        <Code>
          <div><span className="c">{t("// frontend/app/(portal)/name/page.tsx — Path: \"/name\"-тэй ижил")}</span></div>
          <div><span className="s">&quot;use client&quot;</span>;</div>
          <div>&nbsp;</div>
          <div><span className="k">export default function</span> NamePage() {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">const</span> {"{"} me {"}"} = useShell();&nbsp;&nbsp;<span className="c">{t("// хэрэглэгч + permissions")}</span></div>
          <div>&nbsp;&nbsp;<span className="k">const</span> {"{"} t {"}"} = useT();&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">{t("// хэл (mn/en)")}</span></div>
          <div>&nbsp;&nbsp;<span className="k">const</span> manage = me.permissions[<span className="s">&quot;name.manage&quot;</span>]; <span className="c">{t("// undefined | \"all\" | \"own\"")}</span></div>
          <div>&nbsp;&nbsp;<span className="c">{t("// api.get(`/api/apps/name/`) — cookie автоматаар, 401 бол login руу")}</span></div>
          <div>{"}"}</div>
        </Code>
        <ul style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li>{t("Эрхээр UI-гаа нуу:")} <code>me.permissions[&quot;name.manage&quot;]</code> {t("байхгүй бол товчоо бүү харуул (энэ нь UX — жинхэнэ хамгаалалт серверт)")}</li>
          <li>{t("«Өөрийн» scope-той хэрэглэгчид засах/устгах товчийг")} <code>created_by === me.user.id</code> {t("үед л харуулна")}</li>
          <li>{t("Цэсний icon нэрээ")} <code>components/icons.tsx</code>-{t("ийн map-д нэм (lucide icon)")}</li>
          <li>{t("Бэлэн загварууд:")} <code>card / table / btn / field / badge / modal</code> — globals.css; {t("амжилтад")} <code>toast(...)</code>, {t("текстэд")} <code>t(...)</code></li>
        </ul>

        <h3>{t("7. Бүртгэх ба асаах")}</h3>
        <Code>
          <div><span className="c">{t("// backend/apps/apps.go — нэг мөр:")}</span></div>
          <div>nexus.Register(name.New())</div>
          <div>&nbsp;</div>
          <div><span className="c">$</span> make migrate &amp;&amp; make serve&nbsp;&nbsp;<span className="c">{t("# модуль store-д гарч ирнэ")}</span></div>
        </Code>

        <h3>{t("8. Store-д нийтлэх")}</h3>
        <p style={{ color: "var(--text-2)" }}>
          <code>catalog/apps.json</code>{t("-д бүртгэлээ нэмээд PR илгээнэ. Үе 2-т төв registry +")}{" "}
          <code>nexus-mini add</code> {t("CLI ирэхэд go_module замаар тань шууд татдаг болно.")}
        </p>

        <h2 style={{ marginTop: "2rem" }}><BookOpen size={19} style={{ verticalAlign: "-3px" }} /> {t("Тест")}</h2>
        <p style={{ color: "var(--text-2)" }}>
          {t("SQL parse/encode бүх логикт unit тест бич.")} <code>make check</code> {t("нь linux cross-build + vet + test + SDK-ийн хилийн шалгалт (модуль internal/* импортолбол унадаг) — push бүрийн өмнө заавал.")}
        </p>

        <div className="card card__pad" style={{ marginTop: "2rem", display: "flex", alignItems: "center", gap: "1rem", flexWrap: "wrap" }}>
          <span className="stat__icon"><GitPullRequest size={19} /></span>
          <div style={{ flex: 1, minWidth: 240 }}>
            <b>{t("Бэлэн үү?")}</b>
            <div style={{ color: "var(--text-2)", fontSize: "0.9rem" }}>
              {t("devices-ийг хуулж эхлээд, дуусаад каталогт PR илгээгээрэй.")}
            </div>
          </div>
          <a href="https://github.com/gerege-systems/nexus-mini/blob/main/docs/03-module-guide.md" className="btn btn--ghost">{t("Markdown хувилбар")}</a>
          <Link href="/apps" className="btn">{t("Апп дэлгүүр үзэх")}</Link>
        </div>
      </section>

      <MktFooter />
    </div>
  );
}
