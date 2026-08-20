"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Row = { id: string; slug: string; name: string; created_at: string; members: number; apps: number };

export default function TenantsPage() {
  const [rows, setRows] = useState<Row[]>([]);
  useEffect(() => {
    void api.get<{ tenants: Row[] }>("/api/admin/tenants").then((r) => setRows(r.tenants));
  }, []);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Байгууллагууд</h1>
          <div className="sub">Платформ дээрх бүх байгууллага</div>
        </div>
      </div>
      <div className="card">
        <table className="table">
          <thead><tr><th>Нэр</th><th>Slug</th><th>Гишүүд</th><th>Аппууд</th><th>Үүссэн</th></tr></thead>
          <tbody>
            {rows.map((t) => (
              <tr key={t.id}>
                <td><b style={{ fontWeight: 600 }}>{t.name}</b></td>
                <td><code>{t.slug}</code></td>
                <td>{t.members}</td>
                <td>{t.apps}</td>
                <td style={{ color: "var(--text-2)" }}>{new Date(t.created_at).toLocaleDateString("mn-MN")}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
