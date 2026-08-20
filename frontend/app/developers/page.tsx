"use client";

import Link from "next/link";
import { BookOpen, FolderTree, GitPullRequest, Rocket } from "lucide-react";
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
          <div>backend/apps/{"<нэр>"}/</div>
          <div>&nbsp;&nbsp;module.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">модулийн ГЭРЭЭ: ID, permission, цэс, миграц, route↔permission холболт</span></div>
          <div>&nbsp;&nbsp;types.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">хүсэлт/хариултын struct + validation</span></div>
          <div>&nbsp;&nbsp;handlers.go&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">HTTP handler-ууд (нэг resource = нэг файл)</span></div>
          <div>&nbsp;&nbsp;migrations/&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;<span className="c">модулийн goose миграцууд</span></div>
        </Code>

        <h2><Rocket size={19} style={{ verticalAlign: "-3px" }} /> {t("Алхамууд")}</h2>

        <h3 style={{ marginTop: "1.5rem" }}>{t("1. Package үүсгэх")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) ID() <span className="k">string</span>      {"{"} <span className="k">return</span> <span className="s">&quot;mn.танай.&lt;нэр&gt;&quot;</span> {"}"} <span className="c">// reverse-DNS, глобал давтагдашгүй</span></div>
          <div><span className="k">func</span> (m *Module) ShortID() <span className="k">string</span> {"{"} <span className="k">return</span> <span className="s">&quot;&lt;нэр&gt;&quot;</span> {"}"}          <span className="c">// permission prefix + URL зам</span></div>
          <div><span className="k">func</span> (m *Module) Name() <span className="k">string</span>    {"{"} <span className="k">return</span> <span className="s">&quot;Хүний нэр&quot;</span> {"}"}</div>
          <div><span className="k">func</span> (m *Module) Version() <span className="k">string</span> {"{"} <span className="k">return</span> <span className="s">&quot;1.0.0&quot;</span> {"}"}</div>
        </Code>

        <h3>{t("2. Permission тунхаглах")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) Permissions() []nexus.PermissionDefinition {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">return</span> []nexus.PermissionDefinition{"{"}</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;{"{"}Code: <span className="s">&quot;&lt;нэр&gt;.read&quot;</span>, Name: <span className="s">&quot;...&quot;</span>, DefaultRoles: []<span className="k">string</span>{"{"}<span className="s">&quot;manager&quot;</span>, <span className="s">&quot;user&quot;</span>{"}}"},</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;{"{"}Code: <span className="s">&quot;&lt;нэр&gt;.manage&quot;</span>, Name: <span className="s">&quot;...&quot;</span>, OwnScope: <span className="k">true</span>,</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;DefaultRoles: []<span className="k">string</span>{"{"}<span className="s">&quot;manager&quot;</span>, <span className="s">&quot;user:own&quot;</span>{"}}"},</div>
          <div>&nbsp;&nbsp;{"}"}</div>
          <div>{"}"}</div>
        </Code>
        <p style={{ color: "var(--text-2)" }}>{t("Дүрмүүд (зөрчвөл бинари асахгүй):")}</p>
        <ul style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li>Код заавал <code>&lt;ShortID&gt;.</code>-ээр эхэлнэ — өөр модулийн эрхийг булааж чадахгүй</li>
          <li><code>DefaultRoles</code> нь суулгах үед хэн авахыг <b>тунхагладаг</b>: <code>admin</code> үргэлж бүгдийг авна, жагсаалтад бичсэн нь нэмж авна, <code>&quot;user:own&quot;</code> нь зөвхөн өөрийн мөрийн эрх</li>
          <li><code>DefaultRoles</code> хоосон = зөвхөн admin (аюулгүй default)</li>
          <li><code>core</code>, <code>api</code>, <code>admin</code> зэрэг нэрс нөөцлөгдсөн</li>
        </ul>

        <h3>{t("3. Миграц")}</h3>
        <Code>
          <div><span className="c">//go:embed migrations/*.sql</span></div>
          <div><span className="k">var</span> migrations embed.FS</div>
          <div><span className="k">func</span> (m *Module) Migrations() fs.FS {"{"} <span className="k">return</span> migrations {"}"}</div>
        </Code>
        <ul style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li><code>tenant_id uuid NOT NULL</code> + RLS policy (<code>app_tenant_id()</code>) — жишээг devices-ээс хуул</li>
          <li><code>OwnScope</code> ашиглах бол <code>created_by uuid</code> багана заавал</li>
          <li>Бүх string баганад урттай хязгаар (varchar(n)) — задгай text хориотой</li>
          <li>Төгсгөлд нь <code>GRANT ... TO nexus_app, nexus_admin</code></li>
        </ul>
        <p style={{ color: "var(--text-2)" }}>
          Модуль бүр өөрийн goose хүснэгттэй (<code>goose_&lt;shortid&gt;</code>) тул цөм болон бусад модультай мөргөлдөхгүй.
        </p>

        <h3>{t("4. Route-ууд")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) RegisterRoutes(r chi.Router, deps nexus.Deps) {"{"}</div>
          <div>&nbsp;&nbsp;h := &amp;handler{"{"}deps: deps{"}"}</div>
          <div>&nbsp;&nbsp;r.With(nexus.RequirePermission(deps.Perms, <span className="s">&quot;&lt;нэр&gt;.read&quot;</span>)).Get(<span className="s">&quot;/&quot;</span>, h.list)</div>
          <div>&nbsp;&nbsp;r.With(nexus.RequirePermission(deps.Perms, <span className="s">&quot;&lt;нэр&gt;.manage&quot;</span>)).Post(<span className="s">&quot;/&quot;</span>, h.create)</div>
          <div>{"}"}</div>
        </Code>
        <p style={{ color: "var(--text-2)" }}>
          Танд өгөгдөх <code>r</code> нь <b>аль хэдийн хамгаалагдсан</b>: <code>/api/apps/&lt;ShortID&gt;/</code> дор
          байрладаг, нэвтрээгүй хүн 401, апп суулгаагүй tenant 403 авчихсан байдаг. Handler дотор:
        </p>
        <ul style={{ color: "var(--text-2)", lineHeight: 1.8 }}>
          <li><code>nexus.TenantID(ctx)</code>, <code>nexus.UserID(ctx)</code> — хүсэлтийн identity</li>
          <li><code>nexus.Scope(ctx)</code> — <code>ScopeOwn</code> бол query-дээ <code>created_by</code> шүүлт нэм</li>
          <li><code>deps.DB</code> — RLS context автоматаар тохирдог холболт; SQL-даа <code>tenant_id = $1</code> гэж бас бич</li>
          <li><code>deps.Audit.Record(ctx, ...)</code> — чухал үйлдлээ audit гинжид бич</li>
          <li><code>nexus.JSON / Decode / Error / DBError</code> — вэб туслахууд</li>
        </ul>

        <h3>{t("5. Цэс")}</h3>
        <Code>
          <div><span className="k">func</span> (m *Module) Menus() []nexus.MenuDefinition {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">return</span> []nexus.MenuDefinition{"{{"}</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;ID: <span className="s">&quot;&lt;нэр&gt;.list&quot;</span>, Label: <span className="s">&quot;Монгол нэр&quot;</span>, Labels: <span className="k">map</span>[<span className="k">string</span>]<span className="k">string</span>{"{"}<span className="s">&quot;en&quot;</span>: <span className="s">&quot;English&quot;</span>{"}"},</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;Path: <span className="s">&quot;/&lt;нэр&gt;&quot;</span>, Icon: <span className="s">&quot;device&quot;</span>, Order: 10,</div>
          <div>&nbsp;&nbsp;{"}}"}</div>
          <div>{"}"}</div>
        </Code>

        <h3>{t("6. Бүртгэх ба асаах")}</h3>
        <Code>
          <div><span className="c">// backend/apps/apps.go — нэг мөр:</span></div>
          <div>nexus.Register(&lt;нэр&gt;.New())</div>
          <div>&nbsp;</div>
          <div><span className="c">$</span> make migrate &amp;&amp; make api  <span className="c"># модуль store-д гарч ирнэ</span></div>
        </Code>

        <h3>{t("7. Store-д нийтлэх")}</h3>
        <p style={{ color: "var(--text-2)" }}>
          <code>catalog/apps.json</code>-д бүртгэлээ нэмээд PR илгээнэ. Үе 2-т төв
          registry + <code>nexus-mini add</code> CLI ирэхэд <code>go_module</code> замаар
          тань шууд татдаг болно.
        </p>

        <h2 style={{ marginTop: "2rem" }}><BookOpen size={19} style={{ verticalAlign: "-3px" }} /> {t("Тест")}</h2>
        <p style={{ color: "var(--text-2)" }}>
          SQL parse/encode бүх логикт unit тест бич. <code>make check</code> нь linux
          cross-build + vet + test + SDK-ийн хилийн шалгалт (модуль <code>internal/*</code>
          импортолбол унадаг) — push бүрийн өмнө заавал.
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
