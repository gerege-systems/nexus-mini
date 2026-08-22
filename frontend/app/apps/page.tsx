"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Package } from "lucide-react";
import { api } from "@/lib/api";
import { MktHeader, MktFooter } from "@/components/mkt";
import { useT } from "@/lib/i18n";

type CatalogApp = {
  id: string;
  short_id: string;
  name: string;
  version: string;
  description: string;
  publisher: string;
  compiled: boolean;
};

// Нийтийн апп дэлгүүрийн хуудас — нэвтрэлт шаардахгүй каталог.
export default function PublicAppsPage() {
  const { t } = useT();
  const [apps, setApps] = useState<CatalogApp[] | null>(null);
  useEffect(() => {
    void api.get<{ apps: CatalogApp[] }>("/api/catalog").then((r) => setApps(r.apps));
  }, []);

  return (
    <div className="mkt">
      <MktHeader />

      <section className="mkt-hero" style={{ paddingBottom: "1.5rem" }}>
        <h1>{t("Апп дэлгүүр")}</h1>
        <p>
          {t("Байгууллага бүр өөрт хэрэгтэй модулиа л суулгана. Суусан апп нь permission-уудаа role-уудад тунхагласан ёсоор оноож, цэсээ эрхтэй хүнд л харуулна; унтраавал бүгд эргэж алга болно.")}
        </p>
      </section>

      <section className="mkt-sect" style={{ paddingTop: 0 }}>
        <div className="mkt-steps" style={{ marginBottom: "2rem" }}>
          <div className="card mkt-feature">
            <span className="mkt-num">1</span>
            <h3>{t("Дэлгүүрээс сонгоно")}</h3>
            <p>{t("Каталогоос аппаа сонгоод «Суулгах» — хамаарлуудыг нь платформ өөрөө цэгцэлнэ.")}</p>
          </div>
          <div className="card mkt-feature">
            <span className="mkt-num">2</span>
            <h3>{t("Эрх автоматаар")}</h3>
            <p>{t("Аппын permission-ууд role-уудад тунхагласан ёсоороо оноогдоно; админ дараа нь чөлөөтэй өөрчилнө.")}</p>
          </div>
          <div className="card mkt-feature">
            <span className="mkt-num">3</span>
            <h3>{t("Цэс гарч ирнэ")}</h3>
            <p>{t("Эрхтэй хэрэглэгчид л аппын цэсийг харна. Rail дээр аппын icon нэмэгдэж, өөрийн цэстэйгээ ирнэ.")}</p>
          </div>
        </div>

        <h2 style={{ fontSize: "1.3rem", margin: "0 0 1rem" }}>{t("Одоо байгаа аппууд")}</h2>
        <div className="app-grid">
          {apps?.map((a) => (
            <div key={a.id} className="card app-card">
              <div className="app-card__head">
                <span className="app-card__icon"><Package size={22} strokeWidth={1.7} /></span>
                <div>
                  <b>{a.name}</b>
                  <div style={{ color: "var(--text-3)", fontSize: "0.8rem" }}>{a.publisher || "—"}</div>
                </div>
              </div>
              <p>{a.description || t("Тайлбар оруулаагүй.")}</p>
              <div className="app-card__foot">
                {a.compiled
                  ? <span className="badge badge--ok">{t("Бэлэн")}</span>
                  : <span className="badge badge--muted" title={`nexus add ${a.short_id}`}>{t("Registry-д бүртгэлтэй")} · <code>nexus add {a.short_id}</code></span>}
                <span className="v">v{a.version}</span>
              </div>
            </div>
          ))}
          {apps && apps.length === 0 && (
            <p style={{ color: "var(--text-2)" }}>{t("Каталог хоосон байна.")}</p>
          )}
        </div>

        <div className="card card__pad" style={{ marginTop: "2rem", display: "flex", alignItems: "center", gap: "1rem", flexWrap: "wrap" }}>
          <div style={{ flex: 1, minWidth: 240 }}>
            <b>{t("Өөрийн модулиа энд гаргамаар байна уу?")}</b>
            <div style={{ color: "var(--text-2)", fontSize: "0.9rem" }}>
              {t("Гарын авлагыг дагаад модулиа бичээд каталогт PR илгээгээрэй.")}
            </div>
          </div>
          <Link href="/developers" className="btn">{t("Модуль хөгжүүлэх заавар")}</Link>
        </div>
      </section>

      <MktFooter />
    </div>
  );
}
