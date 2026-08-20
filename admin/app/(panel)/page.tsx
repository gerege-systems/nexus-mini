"use client";

import { useEffect, useState } from "react";
import { Building2, Package, ScrollText, Users } from "lucide-react";
import { api } from "@/lib/api";

type Overview = { tenants: number; users: number; apps: number; installations: number };

export default function OverviewPage() {
  const [ov, setOv] = useState<Overview | null>(null);
  useEffect(() => {
    void api.get<Overview>("/api/admin/overview").then(setOv);
  }, []);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Тойм</h1>
          <div className="sub">Платформын ерөнхий үзүүлэлтүүд</div>
        </div>
      </div>
      {ov && (
        <div className="stat-grid">
          <div className="card stat">
            <span className="stat__icon"><Building2 size={19} /></span>
            <span><b>{ov.tenants}</b><span>Байгууллага</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><Users size={19} /></span>
            <span><b>{ov.users}</b><span>Хэрэглэгч</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><Package size={19} /></span>
            <span><b>{ov.apps}</b><span>Бэлэн апп</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><ScrollText size={19} /></span>
            <span><b>{ov.installations}</b><span>Суулгалт</span></span>
          </div>
        </div>
      )}
    </>
  );
}
