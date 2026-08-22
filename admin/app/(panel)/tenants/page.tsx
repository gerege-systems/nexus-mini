"use client";

import { useEffect, useState } from "react";
import { LogIn, ShieldAlert, Users } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { useT } from "@/lib/i18n";
import { toast } from "@/lib/toast";

type Row = { id: string; slug: string; name: string; created_at: string; suspended: boolean; reason: string; read_only: boolean; deletion_at: string | null; members: number; apps: number };
type Member = { id: string; name: string; email: string; platform_admin: boolean; roles: string[] };

export default function TenantsPage() {
  const { t } = useT();
  const [rows, setRows] = useState<Row[]>([]);
  const [open, setOpen] = useState<Row | null>(null);
  const [members, setMembers] = useState<Member[] | null>(null);
  const [err, setErr] = useState("");
  const [state, setState] = useState<{ row: Row; suspended: boolean; reason: string; read_only: boolean } | null>(null);
  const loadRows = () => api.get<{ tenants: Row[] }>("/api/admin/tenants").then((r) => setRows(r.tenants));
  useEffect(() => { void loadRows(); }, []);
  // Escape — modal-ууд focus-гүй div тул document түвшинд сонсоно.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => { if (e.key === "Escape") { setState(null); setOpen(null); } };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, []);

  const setDeletion = async (row: Row, schedule: boolean) => {
    const msg = schedule
      ? `${row.name} — ${t("30 хоногийн дараа бүрмөсөн устгахаар товлох уу? Гишүүд тэр дороо хандах боломжгүй болно; хүртэл нь буцааж болно.")}`
      : `${row.name} — ${t("устгалыг цуцлах уу?")}`;
    if (!confirm(msg)) return;
    try {
      await api.post(`/api/admin/tenants/${row.id}/delete${schedule ? "" : "/cancel"}`);
      toast(schedule ? t("Устгалд товлогдлоо (30 хоног)") : t("Устгал цуцлагдлаа"));
      setState(null);
      await loadRows();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    }
  };

  const saveState = async () => {
    if (!state) return;
    try {
      await api.put(`/api/admin/tenants/${state.row.id}/state`,
        { suspended: state.suspended, reason: state.reason, read_only: state.read_only });
      toast(t("Төлөв хадгалагдлаа"));
      setState(null);
      await loadRows();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    }
  };
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
      const r = await api.post<{ url: string; token: string }>("/api/admin/impersonate", { tenant_id: open.id, user_id: m.id });
      // Token-ийг URL-д биш POST биед — access log/түүхэнд үлдэхгүй.
      const f = document.createElement("form");
      f.method = "POST"; f.action = r.url; f.target = "_blank"; f.rel = "noopener";
      const i = document.createElement("input");
      i.type = "hidden"; i.name = "token"; i.value = r.token;
      f.appendChild(i); document.body.appendChild(f); f.submit(); f.remove();
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
                <td>
                  <b style={{ fontWeight: 600 }}>{r.name}</b>
                  {r.suspended && <span className="badge badge--danger" style={{ marginLeft: "0.5rem" }}>{t("түдгэлзүүлсэн")}</span>}
                  {r.read_only && !r.suspended && <span className="badge badge--warn" style={{ marginLeft: "0.5rem" }}>{t("зөвхөн унших")}</span>}
                  {r.deletion_at && <span className="badge badge--danger" style={{ marginLeft: "0.5rem" }}>{t("устгал")}: {new Date(r.deletion_at).toLocaleDateString("mn-MN")}</span>}
                </td>
                <td><code>{r.slug}</code></td>
                <td>{r.members}</td>
                <td>{r.apps}</td>
                <td style={{ color: "var(--text-2)" }}>{new Date(r.created_at).toLocaleDateString("mn-MN")}</td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  <button className="btn btn--ghost btn--sm" onClick={() => setOpen(r)}>
                    <Users size={14} /> {t("Гишүүд")}
                  </button>{" "}
                  <button className="btn btn--ghost btn--sm"
                    onClick={() => { setErr(""); setState({ row: r, suspended: r.suspended, reason: r.reason, read_only: r.read_only }); }}>
                    <ShieldAlert size={14} /> {t("Төлөв")}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {state && (
        <div className="modal-back" onClick={() => setState(null)} onKeyDown={(e) => e.key === "Escape" && setState(null)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
            <h3>{state.row.name} — {t("Төлөв")}</h3>
            {err && <div className="alert alert--danger">{t(err)}</div>}
            <div className="field">
              <label style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                <input type="checkbox" checked={state.suspended} style={{ width: "auto" }}
                  onChange={(e) => setState({ ...state, suspended: e.target.checked })} />
                {t("Түдгэлзүүлэх — гишүүд өгөгдөлдөө хандаж чадахгүй")}
              </label>
            </div>
            {state.suspended && (
              <div className="field">
                <label>{t("Шалтгаан (гишүүдэд харагдана)")}</label>
                <input value={state.reason} maxLength={300} onChange={(e) => setState({ ...state, reason: e.target.value })} />
              </div>
            )}
            <div className="field">
              <label style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                <input type="checkbox" checked={state.read_only} style={{ width: "auto" }}
                  onChange={(e) => setState({ ...state, read_only: e.target.checked })} />
                {t("Зөвхөн унших — бичих хүсэлт 503 (засвар, төлбөр)")}
              </label>
            </div>
            <div className="field" style={{ borderTop: "1px solid var(--border)", paddingTop: "0.8rem" }}>
              <label>{t("Устгал (30 хоногийн хүлээлт)")}</label>
              {state.row.deletion_at ? (
                <div style={{ display: "flex", gap: "0.6rem", alignItems: "center" }}>
                  <span style={{ color: "var(--danger)" }}>{new Date(state.row.deletion_at).toLocaleString("mn-MN")}</span>
                  <button className="btn btn--ghost btn--sm" onClick={() => setDeletion(state.row, false)}>{t("Устгалыг цуцлах")}</button>
                </div>
              ) : (
                <button className="btn btn--ghost btn--sm" style={{ color: "var(--danger)" }} onClick={() => setDeletion(state.row, true)}>{t("Устгалд товлох")}</button>
              )}
            </div>
            <div className="modal__actions">
              <button className="btn btn--ghost" onClick={() => setState(null)}>{t("Болих")}</button>
              <button className="btn" onClick={saveState}>{t("Хадгалах")}</button>
            </div>
          </div>
        </div>
      )}

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
