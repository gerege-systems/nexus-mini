"use client";

import { useEffect, useState } from "react";
import { Building2, Package, ScrollText, Users } from "lucide-react";
import { api } from "@/lib/api";
import { useT } from "@/lib/i18n";

type Overview = { tenants: number; users: number; apps: number; installations: number };

export default function OverviewPage() {
  const { t } = useT();
  const [ov, setOv] = useState<Overview | null>(null);
  useEffect(() => {
    void api.get<Overview>("/api/admin/overview").then(setOv);
  }, []);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Тойм")}</h1>
          <div className="sub">{t("Платформын ерөнхий үзүүлэлтүүд")}</div>
        </div>
      </div>
      {ov && (
        <div className="stat-grid">
          <div className="card stat">
            <span className="stat__icon"><Building2 size={19} /></span>
            <span><b>{ov.tenants}</b><span>{t("Байгууллага")}</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><Users size={19} /></span>
            <span><b>{ov.users}</b><span>{t("Хэрэглэгч")}</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><Package size={19} /></span>
            <span><b>{ov.apps}</b><span>{t("Бэлэн апп")}</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><ScrollText size={19} /></span>
            <span><b>{ov.installations}</b><span>{t("Суулгалт")}</span></span>
          </div>
        </div>
      )}
    </>
  );
}
