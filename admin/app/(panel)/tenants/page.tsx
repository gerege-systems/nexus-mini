"use client";

import { useEffect, useState } from "react";
import { LogIn, Users } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { useT } from "@/lib/i18n";
import { toast } from "@/lib/toast";

type Row = { id: string; slug: string; name: string; created_at: string; members: number; apps: number };
type Member = { id: string; name: string; email: string; platform_admin: boolean; roles: string[] };

export default function TenantsPage() {
  const { t } = useT();
  const [rows, setRows] = useState<Row[]>([]);
  const [open, setOpen] = useState<Row | null>(null);
  const [members, setMembers] = useState<Member[] | null>(null);
  const [err, setErr] = useState("");
  useEffect(() => {
    void api.get<{ tenants: Row[] }>("/api/admin/tenants").then((r) => setRows(r.tenants));
  }, []);
  useEffect(() => {
    if (!open) { setMembers(null); setErr(""); return; }
    void api.get<{ members: Member[] }>(`/api/admin/tenants/${open.id}/members`)
      .then((r) => setMembers(r.members))
      .catch(() => setErr(t("Алдаа гарлаа")));
  }, [open, t]);

  // Нэг удаагийн handover URL-ийг шинэ таб-д нээнэ — portal тусдаа домэйн.
  const impersonate = async (m: Member) => {
    if (!open) return;
    if (!confirm(`${m.name} (${m.email}) — ${t("нэрийн өмнөөс нэвтрэх үү? Үйлдэл бүр audit-д таны нэрээр тэмдэглэгдэнэ.")}`)) return;
    try {
      const r = await api.post<{ url: string }>("/api/admin/impersonate", { tenant_id: open.id, user_id: m.id });
      window.open(r.url, "_blank", "noopener");
      toast(t("Handover холбоос нээгдлээ (60 секунд хүчинтэй)"));
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    }
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Байгууллагууд")}</h1>
          <div className="sub">{t("Платформ дээрх бүх байгууллага")}</div>
        </div>
      </div>
      <div className="card">
        <table className="table">
          <thead><tr><th>{t("Нэр")}</th><th>Slug</th><th>{t("Гишүүд")}</th><th>{t("Аппууд")}</th><th>{t("Үүссэн")}</th><th></th></tr></thead>
          <tbody>
            {rows.map((r) => (
              <tr key={r.id}>
                <td><b style={{ fontWeight: 600 }}>{r.name}</b></td>
                <td><code>{r.slug}</code></td>
                <td>{r.members}</td>
                <td>{r.apps}</td>
                <td style={{ color: "var(--text-2)" }}>{new Date(r.created_at).toLocaleDateString("mn-MN")}</td>
                <td style={{ textAlign: "right" }}>
                  <button className="btn btn--ghost btn--sm" onClick={() => setOpen(r)}>
                    <Users size={14} /> {t("Гишүүд")}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {open && (
        <div className="modal-back" onClick={() => setOpen(null)} onKeyDown={(e) => e.key === "Escape" && setOpen(null)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()} style={{ maxWidth: 640 }}>
            <h3>{open.name} — {t("Гишүүд")}</h3>
            {err && <div className="alert alert--danger">{t(err)}</div>}
            {members === null ? (
              <div style={{ color: "var(--text-3)" }}>{t("Уншиж байна…")}</div>
            ) : members.length === 0 ? (
              <div style={{ color: "var(--text-3)" }}>{t("Гишүүн байхгүй")}</div>
            ) : (
              <table className="table">
                <thead><tr><th>{t("Нэр")}</th><th>{t("Имэйл")}</th><th>Role</th><th></th></tr></thead>
                <tbody>
                  {members.map((m) => (
                    <tr key={m.id}>
                      <td>{m.name}</td>
                      <td style={{ color: "var(--text-2)" }}>{m.email}</td>
                      <td>{m.roles.join(", ") || "—"}</td>
                      <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                        {!m.platform_admin && (
                          <button className="btn btn--ghost btn--sm" onClick={() => impersonate(m)}
                            title={t("Энэ хэрэглэгчийн нэрийн өмнөөс portal-д нэвтрэх")}>
                            <LogIn size={14} /> {t("Нэрийн өмнөөс нэвтрэх")}
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
            <div className="modal__actions">
              <button className="btn btn--ghost" onClick={() => setOpen(null)}>{t("Хаах")}</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
