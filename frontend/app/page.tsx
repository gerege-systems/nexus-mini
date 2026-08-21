"use client";

import Link from "next/link";
import { ArrowRight, Code2, Store } from "lucide-react";
import { MktHeader, MktFooter } from "@/components/mkt";
import { useT } from "@/lib/i18n";

// Нүүр — ерөнхий танилцуулга. Апп дэлгүүр (/apps) ба модуль хөгжүүлэх
// (/developers) тус тусдаа хуудсуудтай.
export default function Landing() {
  const { t } = useT();
  return (
    <div className="mkt">
      <MktHeader />

      <section className="hero2">
        <div className="hero2__left">
          <div className="hero2__badge"><span className="dot" /> v1.0 · Apache 2.0</div>
          <h1>{t("Үйл ажиллагааны нэгдсэн дижитал платформ")}</h1>
          <p>
            {t("Байгууллагын үйлчилгээ, үйл ажиллагаа, систем, өгөгдлийг нэг дор холбодог модульт платформ — цөм нь суурийг, апп дэлгүүр нь боломжуудыг өгнө.")}
          </p>
          <div className="cta">
            <Link href="/signup" className="btn">{t("Байгууллагаа бүртгүүлэх")}</Link>
            <Link href="/developers" className="btn btn--ghost hero2__mono-btn">$ docker compose up</Link>
          </div>
        </div>
        <div className="term" aria-hidden="true">
          <div className="term__bar">
            <span className="term__dot" style={{ background: "#f87171" }} />
            <span className="term__dot" style={{ background: "#fbbf24" }} />
            <span className="term__dot" style={{ background: "#34d399" }} />
            <span className="term__title">nexus-mini — zsh</span>
          </div>
          <div className="term__body">
            <div><span className="term__p">$</span> git clone gerege-systems/nexus-mini</div>
            <div><span className="term__p">$</span> nexus-mini migrate</div>
            <div className="term__out">{t("миграц: цөм ok · devices ok")}</div>
            <div className="term__out">✓ {t("платформын админ үүслээ")}</div>
            <div><span className="term__p">$</span> nexus-mini serve</div>
            <div className="term__out">nexus-mini API :8084 (1 {t("модуль")})</div>
            <div><span className="term__p">$</span> <span className="term__cursor">&nbsp;</span></div>
          </div>
          <div className="term__foot">
            <span>Go 1.25</span><span>PostgreSQL 16 · RLS</span><span>Next.js</span><span>{t("нэг бинари")}</span>
          </div>
        </div>
      </section>

      <section className="mkt-sect" id="core">
        <div className="mkt-grid3">
          <div className="card mkt-feature">
            <div className="mkt-mono-tag">tenant + RLS</div>
            <h3>{t("Тусгаарлалт DB давхаргад")}</h3>
            <p>{t("Байгууллага бүрийн өгөгдөл PostgreSQL Row-Level Security-ээр тусгаарлагдана — кодын алдаа ч хана даван харагдуулахгүй.")}</p>
          </div>
          <div className="card mkt-feature">
            <div className="mkt-mono-tag">rbac + audit</div>
            <h3>{t("Эрх тунхаглалаар, бүртгэл гинжээр")}</h3>
            <p>{t("Permission суулгах үед role-уудад автоматаар оноогдоно; бүх чухал үйлдэл hash chain-тэй audit бүртгэлд үлдэнэ.")}</p>
          </div>
          <div className="card mkt-feature">
            <div className="mkt-mono-tag">nexus.Module</div>
            <h3>{t("7 метод = таны модуль")}</h3>
            <p>{t("Go interface хэрэгжүүлээд каталогт PR илгээхэд л таны модуль store-д — нэг бинари, микросервисийн төвөггүй.")}</p>
          </div>
        </div>
      </section>

      <section className="mkt-sect">
        <div className="mkt-grid3" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))" }}>
          <Link href="/apps" className="card mkt-feature" style={{ display: "block" }}>
            <span className="ic"><Store size={19} /></span>
            <h3>{t("Апп дэлгүүр")} <ArrowRight size={14} style={{ verticalAlign: "-2px" }} /></h3>
            <p>
              {t("Байгууллага бүр өөрт хэрэгтэй модулиа сонгож суулгана — суусан апп эрх, цэсээ өөрөө авчирна. Одоо байгаа аппуудыг тайлбартай нь үзэх.")}
            </p>
          </Link>
          <Link href="/developers" className="card mkt-feature" style={{ display: "block" }}>
            <span className="ic"><Code2 size={19} /></span>
            <h3>{t("Модуль хөгжүүлэх")} <ArrowRight size={14} style={{ verticalAlign: "-2px" }} /></h3>
            <p>
              {t("Модуль бол долоон метод хэрэгжүүлсэн Go package. Файлын бүтэц, permission, миграц, route — бүрэн гарын авлага.")}
            </p>
          </Link>
        </div>
      </section>

      <section className="mkt-sect">
        <div className="card card__pad" style={{ display: "flex", alignItems: "center", gap: "1rem", flexWrap: "wrap" }}>
          <span className="stat__icon"><Store size={19} /></span>
          <div style={{ flex: 1, minWidth: 240 }}>
            <b>{t("Өөрөө ажиллуулж үзэх үү?")}</b>
            <div style={{ color: "var(--text-2)", fontSize: "0.9rem" }}>
              <code>git clone</code> → {t("env-ээ бөглөөд")} <code>nexus-mini migrate</code> →{" "}
              <code>serve</code>. {t("Эсвэл")} <code>docker compose up</code>.
            </div>
          </div>
          <Link href="/signup" className="btn">{t("Эсвэл эндээ бүртгүүлэх")}</Link>
        </div>
      </section>

      <MktFooter />
    </div>
  );
}
