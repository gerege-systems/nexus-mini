"use client";

import Link from "next/link";
import {
  Building2,
  KeyRound,
  Layers,
  ScrollText,
  ShieldCheck,
  Store,
  Waves,
} from "lucide-react";

// Landing — юу болохыг нь эхнээс нь тайлбарлана: цөм + app store + модуль
// хөгжүүлэлт. Анхны тохируулга нь вэбээр биш `nexus-mini setup` CLI-ээр
// хийгддэг тул энд ямар ч төлөв шалгадаггүй.
export default function Landing() {
  return (
    <div className="mkt">
      <header className="mkt-top">
        <span className="brand-square">N</span>
        <b>nexus-mini</b>
        <nav>
          <a href="#core">Цөм</a>
          <a href="#store">Апп дэлгүүр</a>
          <a href="#dev">Модуль хөгжүүлэх</a>
        </nav>
        <span className="spacer" />
        <a href="https://github.com/gerege-systems/nexus-mini" className="btn btn--ghost btn--sm">
          GitHub
        </a>
        <Link href="/login" className="btn btn--ghost btn--sm">Нэвтрэх</Link>
        <Link href="/signup" className="btn btn--sm">Бүртгүүлэх</Link>
      </header>

      <section className="mkt-hero">
        <h1>
          Татаад ажиллуулаад, <em>модулиа бичээд</em>,<br />store-д нийтэлдэг платформ
        </h1>
        <p>
          nexus-mini бол нээлттэй эхийн multi-tenant цөм: байгууллага, эрх, audit,
          нэвтрэлтийг платформ хариуцна — бизнесийн боломж бүр модуль болж
          апп дэлгүүрээр ирнэ. Go + PostgreSQL + Next.js.
        </p>
        <div className="cta">
          <Link href="/signup" className="btn">Байгууллагаа бүртгүүлэх</Link>
          <a href="#dev" className="btn btn--ghost">Модуль хөгжүүлэх</a>
        </div>
      </section>

      <section className="mkt-sect" id="core">
        <div className="mkt-num">01 — ЦӨМ</div>
        <h2>Платформ юу хариуцдаг вэ</h2>
        <p className="lead">
          Модуль бүр дахин бичдэг байсан зүйлс нэг л удаа, цөмд:
        </p>
        <div className="mkt-grid3">
          <div className="card mkt-feature">
            <span className="ic"><Building2 size={19} /></span>
            <h3>Tenant тусгаарлалт</h3>
            <p>Байгууллага бүрийн өгөгдөл PostgreSQL Row-Level Security-ээр DB давхаргад тусгаарлагдана — кодын алдаа ч хана даван харагдуулахгүй.</p>
          </div>
          <div className="card mkt-feature">
            <span className="ic"><KeyRound size={19} /></span>
            <h3>RBAC</h3>
            <p>Модуль permission-оо тунхаглаад л болоо: суулгахад role-уудад автоматаар оноогдоно. «Зөвхөн өөрийн бүртгэл» scope, role-ийн өвлөлт дэмжинэ.</p>
          </div>
          <div className="card mkt-feature">
            <span className="ic"><ScrollText size={19} /></span>
            <h3>Audit гинж</h3>
            <p>Бүх чухал үйлдэл append-only, hash chain-тэй бүртгэлд ордог — гар хүрвэл гинж тасарч илэрнэ. Нэг товчоор шалгана.</p>
          </div>
          <div className="card mkt-feature">
            <span className="ic"><ShieldCheck size={19} /></span>
            <h3>Нэвтрэлт ба SSO</h3>
            <p>Session auth өнөөдөр; OIDC provider + өөр nexus-mini-тэй federation дараагийн үед ирнэ.</p>
          </div>
          <div className="card mkt-feature">
            <span className="ic"><Waves size={19} /></span>
            <h3>Resilience</h3>
            <p>Circuit breaker, load shedding, retry — гадаад системтэй холбогддог модулиудад бэлэн хэрэгсэл (үе 4).</p>
          </div>
          <div className="card mkt-feature">
            <span className="ic"><Layers size={19} /></span>
            <h3>Нэг бинари</h3>
            <p>Модулиуд Go кодоор нэг бинарид компиллогдоно — микросервисийн төвөгггүй, сүлжээний нэмэлт дуудлагагүй.</p>
          </div>
        </div>
      </section>

      <section className="mkt-sect" id="store">
        <div className="mkt-num">02 — АПП ДЭЛГҮҮР</div>
        <h2>Модуль хэрхэн ирдэг вэ</h2>
        <p className="lead">
          Байгууллага бүр өөрт хэрэгтэйгээ л суулгана — суусан апп нь эрх,
          цэсээ өөрөө авчирна.
        </p>
        <div className="mkt-steps">
          <div className="card mkt-feature">
            <span className="mkt-num">1</span>
            <h3>Дэлгүүрээс сонгоно</h3>
            <p>Каталогоос аппаа сонгоод «Суулгах» — хамаарлуудыг нь платформ өөрөө цэгцэлнэ.</p>
          </div>
          <div className="card mkt-feature">
            <span className="mkt-num">2</span>
            <h3>Эрх автоматаар</h3>
            <p>Аппын permission-ууд role-уудад тунхагласан ёсоороо оноогдоно; админ дараа нь чөлөөтэй өөрчилнө.</p>
          </div>
          <div className="card mkt-feature">
            <span className="mkt-num">3</span>
            <h3>Цэс гарч ирнэ</h3>
            <p>Эрхтэй хэрэглэгчид л аппын цэсийг харна. Унтраавал бүх зүйл нь эргэж алга болно.</p>
          </div>
        </div>
      </section>

      <section className="mkt-sect" id="dev">
        <div className="mkt-num">03 — МОДУЛЬ ХӨГЖҮҮЛЭХ</div>
        <h2>Долоон метод л хэрэгжүүлнэ</h2>
        <p className="lead">
          Модуль бол <code>pkg/nexus.Module</code> interface-тэй Go package.
          Tenant тусгаарлалт, нэвтрэлт, суулгалт, RBAC, audit — платформ хийнэ;
          та бизнес логикоо л бичнэ.
        </p>
        <div className="mkt-code">
          <div><span className="k">func</span> (m *Module) Permissions() []nexus.PermissionDefinition {"{"}</div>
          <div>&nbsp;&nbsp;<span className="k">return</span> []nexus.PermissionDefinition{"{{"}</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;Code: <span className="s">&quot;devices.manage&quot;</span>, OwnScope: <span className="k">true</span>,</div>
          <div>&nbsp;&nbsp;&nbsp;&nbsp;DefaultRoles: []<span className="k">string</span>{"{"}<span className="s">&quot;manager&quot;</span>, <span className="s">&quot;user:own&quot;</span>{"}"}, <span className="c">// оноолт нь тунхаглал</span></div>
          <div>&nbsp;&nbsp;{"}}"}</div>
          <div>{"}"}</div>
          <div>&nbsp;</div>
          <div><span className="k">func</span> (m *Module) RegisterRoutes(r chi.Router, deps nexus.Deps) {"{"}</div>
          <div>&nbsp;&nbsp;<span className="c">// r нь аль хэдийн хамгаалагдсан: auth + суулгалтын gate дээр сууна</span></div>
          <div>&nbsp;&nbsp;r.With(nexus.RequirePermission(deps.Perms, <span className="s">&quot;devices.manage&quot;</span>)).Post(<span className="s">&quot;/&quot;</span>, h.create)</div>
          <div>{"}"}</div>
        </div>
        <p style={{ color: "var(--text-2)", marginTop: "1rem" }}>
          Жишээ модуль{" "}
          <a href="https://github.com/gerege-systems/nexus-mini/tree/main/backend/internal/apps/devices"
            style={{ color: "var(--accent)", fontWeight: 600 }}>
            internal/apps/devices
          </a>{" "}
          ← эндээс хуулж эхэл. Бүрэн гарын авлага:{" "}
          <a href="https://github.com/gerege-systems/nexus-mini/blob/main/docs/03-module-guide.md"
            style={{ color: "var(--accent)", fontWeight: 600 }}>
            docs/03-module-guide.md
          </a>
        </p>
      </section>

      <section className="mkt-sect">
        <div className="card card__pad" style={{ display: "flex", alignItems: "center", gap: "1rem", flexWrap: "wrap" }}>
          <span className="stat__icon"><Store size={19} /></span>
          <div style={{ flex: 1, minWidth: 240 }}>
            <b>Өөрөө ажиллуулж үзэх үү?</b>
            <div style={{ color: "var(--text-2)", fontSize: "0.9rem" }}>
              <code>git clone</code> → <code>nexus-mini setup</code> — CLI нь DB, миграц,
              админыг тань тохируулна. Эсвэл <code>docker compose up</code>.
            </div>
          </div>
          <Link href="/signup" className="btn">Эсвэл эндээ бүртгүүлэх</Link>
        </div>
      </section>

      <footer className="mkt-foot">
        <span className="brand-square" style={{ width: "1.6rem", height: "1.6rem", fontSize: "0.8rem" }}>N</span>
        <span>nexus-mini · Apache 2.0 · <a href="https://github.com/gerege-systems/nexus-mini" style={{ color: "var(--accent)" }}>gerege-systems/nexus-mini</a></span>
      </footer>
    </div>
  );
}
