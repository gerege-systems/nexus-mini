"use client";

import Link from "next/link";
import {
  ArrowRight,
  Building2,
  Code2,
  KeyRound,
  Layers,
  ScrollText,
  ShieldCheck,
  Store,
  Waves,
} from "lucide-react";
import { MktHeader, MktFooter } from "@/components/mkt";

// Нүүр — ерөнхий танилцуулга. Апп дэлгүүр (/apps) ба модуль хөгжүүлэх
// (/developers) тус тусдаа хуудсуудтай.
export default function Landing() {
  return (
    <div className="mkt">
      <MktHeader />

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
          <Link href="/developers" className="btn btn--ghost">Модуль хөгжүүлэх</Link>
        </div>
      </section>

      <section className="mkt-sect" id="core">
        <div className="mkt-num">ЦӨМ</div>
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

      <section className="mkt-sect">
        <div className="mkt-grid3" style={{ gridTemplateColumns: "repeat(auto-fit, minmax(320px, 1fr))" }}>
          <Link href="/apps" className="card mkt-feature" style={{ display: "block" }}>
            <span className="ic"><Store size={19} /></span>
            <h3>Апп дэлгүүр <ArrowRight size={14} style={{ verticalAlign: "-2px" }} /></h3>
            <p>
              Байгууллага бүр өөрт хэрэгтэй модулиа сонгож суулгана — суусан апп
              эрх, цэсээ өөрөө авчирна. Одоо байгаа аппуудыг тайлбартай нь үзэх.
            </p>
          </Link>
          <Link href="/developers" className="card mkt-feature" style={{ display: "block" }}>
            <span className="ic"><Code2 size={19} /></span>
            <h3>Модуль хөгжүүлэх <ArrowRight size={14} style={{ verticalAlign: "-2px" }} /></h3>
            <p>
              Модуль бол долоон метод хэрэгжүүлсэн Go package. Файлын бүтэц,
              permission, миграц, route — бүрэн гарын авлага.
            </p>
          </Link>
        </div>
      </section>

      <section className="mkt-sect">
        <div className="card card__pad" style={{ display: "flex", alignItems: "center", gap: "1rem", flexWrap: "wrap" }}>
          <span className="stat__icon"><Store size={19} /></span>
          <div style={{ flex: 1, minWidth: 240 }}>
            <b>Өөрөө ажиллуулж үзэх үү?</b>
            <div style={{ color: "var(--text-2)", fontSize: "0.9rem" }}>
              <code>git clone</code> → env-ээ бөглөөд <code>nexus-mini migrate</code> →{" "}
              <code>serve</code>. Эсвэл <code>docker compose up</code>.
            </div>
          </div>
          <Link href="/signup" className="btn">Эсвэл эндээ бүртгүүлэх</Link>
        </div>
      </section>

      <MktFooter />
    </div>
  );
}
