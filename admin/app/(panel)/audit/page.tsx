"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Row = { id: number; tenant: string; user_name: string; action: string; object: string; occurred_at: string };

export default function AuditPage() {
  const [rows, setRows] = useState<Row[]>([]);
  useEffect(() => {
    void api.get<{ entries: Row[] }>("/api/admin/audit").then((r) => setRows(r.entries));
  }, []);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Audit</h1>
          <div className="sub">Бүх байгууллагын сүүлийн үйлдлүүд</div>
        </div>
      </div>
      <div className="card">
        <table className="table">
          <thead><tr><th>#</th><th>Байгууллага</th><th>Үйлдэл</th><th>Объект</th><th>Хэн</th><th>Хэзээ</th></tr></thead>
          <tbody>
            {rows.map((e) => (
              <tr key={e.id}>
                <td style={{ color: "var(--text-3)" }}>{e.id}</td>
                <td><code>{e.tenant}</code></td>
                <td><span className="badge badge--accent">{e.action}</span></td>
                <td>{e.object || "—"}</td>
                <td>{e.user_name || "систем"}</td>
                <td style={{ color: "var(--text-2)", whiteSpace: "nowrap" }}>
                  {new Date(e.occurred_at).toLocaleString("mn-MN")}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
