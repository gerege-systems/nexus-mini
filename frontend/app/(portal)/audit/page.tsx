"use client";

import { useCallback, useEffect, useState } from "react";
import { ShieldCheck, ShieldX } from "lucide-react";
import { api, type AuditEntry } from "@/lib/api";

export default function AuditPage() {
  const [entries, setEntries] = useState<AuditEntry[] | null>(null);
  const [verify, setVerify] = useState<{ intact: boolean; broken_at: number | null } | null>(null);

  const load = useCallback(async () => {
    const r = await api.get<{ entries: AuditEntry[] }>("/api/audit?limit=100");
    setEntries(r.entries);
  }, []);
  useEffect(() => { void load(); }, [load]);

  const runVerify = async () => {
    setVerify(await api.get("/api/audit/verify"));
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Audit лог</h1>
          <div className="sub">Append-only, hash гинжтэй үйлдлийн бүртгэл</div>
        </div>
        <div className="spacer" />
        <button className="btn btn--ghost" onClick={runVerify}>
          <ShieldCheck size={16} /> Гинж шалгах
        </button>
      </div>

      {verify && (
        <div className={`alert ${verify.intact ? "alert--ok" : "alert--danger"}`}
          style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
          {verify.intact ? <ShieldCheck size={17} /> : <ShieldX size={17} />}
          {verify.intact
            ? "Гинж бүрэн — бүртгэлд гар хүрээгүй"
            : `Гинж #${verify.broken_at} дээр тасарсан!`}
        </div>
      )}

      <div className="card">
        <table className="table">
          <thead>
            <tr><th>#</th><th>Үйлдэл</th><th>Объект</th><th>Хэн</th><th>Хэзээ</th><th>Hash</th></tr>
          </thead>
          <tbody>
            {entries?.map((e) => (
              <tr key={e.id}>
                <td style={{ color: "var(--text-3)" }}>{e.id}</td>
                <td><span className="badge badge--accent">{e.action}</span></td>
                <td>{e.object || "—"}</td>
                <td>{e.user_name || "систем"}</td>
                <td style={{ color: "var(--text-2)", whiteSpace: "nowrap" }}>
                  {new Date(e.occurred_at).toLocaleString("mn-MN")}
                </td>
                <td style={{ fontFamily: "ui-monospace, monospace", fontSize: "0.76rem", color: "var(--text-3)" }}>
                  {e.hash.slice(0, 12)}…
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </>
  );
}
