"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

type Row = { id: string; email: string; name: string; platform_admin: boolean; created_at: string; tenants: number };

export default function UsersPage() {
  const [rows, setRows] = useState<Row[]>([]);
  useEffect(() => {
    void api.get<{ users: Row[] }>("/api/admin/users").then((r) => setRows(r.users));
  }, []);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Хэрэглэгчид</h1>
          <div className="sub">Платформ дээрх бүх бүртгэлтэй хэрэглэгч</div>
        </div>
      </div>
      <div className="card">
        <table className="table">
          <thead><tr><th>Нэр</th><th>Имэйл</th><th>Байгууллага</th><th>Бүртгүүлсэн</th><th></th></tr></thead>
          <tbody>
            {rows.map((u) => (
              <tr key={u.id}>
                <td><b style={{ fontWeight: 600 }}>{u.name}</b></td>
                <td style={{ color: "var(--text-2)" }}>{u.email}</td>
                <td>{u.tenants}</td>
                <td style={{ color: "var(--text-2)" }}>{new Date(u.created_at).toLocaleDateString("mn-MN")}</td>
                <td>{u.platform_admin && <span className="badge badge--accent">платформ админ</span>}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
