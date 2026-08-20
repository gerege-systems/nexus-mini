"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Row = { id: string; short_id: string; name: string; version: string; compiled: boolean; publisher: string; installs: number };

export default function AppsPage() {
  const [rows, setRows] = useState<Row[]>([]);
  useEffect(() => {
    void api.get<{ apps: Row[] }>("/api/admin/apps").then((r) => setRows(r.apps));
  }, []);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Каталог</h1>
          <div className="sub">App store-ийн бүх апп, суулгалтын тоо</div>
        </div>
      </div>
      <div className="card">
        <table className="table">
          <thead><tr><th>Апп</th><th>ID</th><th>Хувилбар</th><th>Бинарид</th><th>Суулгалт</th></tr></thead>
          <tbody>
            {rows.map((a) => (
              <tr key={a.id}>
                <td><b style={{ fontWeight: 600 }}>{a.name}</b>
                  <div style={{ color: "var(--text-3)", fontSize: "0.8rem" }}>{a.publisher}</div></td>
                <td><code style={{ fontSize: "0.8rem" }}>{a.id}</code></td>
                <td>v{a.version}</td>
                <td>{a.compiled
                  ? <span className="badge badge--ok">Тийм</span>
                  : <span className="badge badge--muted">Үгүй</span>}</td>
                <td>{a.installs}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
