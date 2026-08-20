"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { useT } from "@/lib/i18n";

type Row = { id: string; short_id: string; name: string; version: string; compiled: boolean; publisher: string; installs: number };

export default function AppsPage() {
  const { t } = useT();
  const [rows, setRows] = useState<Row[]>([]);
  useEffect(() => {
    void api.get<{ apps: Row[] }>("/api/admin/apps").then((r) => setRows(r.apps));
  }, []);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Каталог")}</h1>
          <div className="sub">{t("App store-ийн бүх апп, суулгалтын тоо")}</div>
        </div>
      </div>
      <div className="card">
        <table className="table">
          <thead><tr><th>{t("Апп")}</th><th>ID</th><th>{t("Хувилбар")}</th><th>{t("Бинарид")}</th><th>{t("Суулгалт")}</th></tr></thead>
          <tbody>
            {rows.map((a) => (
              <tr key={a.id}>
                <td><b style={{ fontWeight: 600 }}>{a.name}</b>
                  <div style={{ color: "var(--text-3)", fontSize: "0.8rem" }}>{a.publisher}</div></td>
                <td><code style={{ fontSize: "0.8rem" }}>{a.id}</code></td>
                <td>v{a.version}</td>
                <td>{a.compiled
                  ? <span className="badge badge--ok">{t("Тийм")}</span>
                  : <span className="badge badge--muted">{t("Үгүй")}</span>}</td>
                <td>{a.installs}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
