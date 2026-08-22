"use client";

import { useCallback, useEffect, useState } from "react";
import { Download, History, Package, Power } from "lucide-react";
import { api, ApiError, type StoreApp } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

export default function StorePage() {
  const { me, refresh } = useShell();
  const { t } = useT();
  const [apps, setApps] = useState<StoreApp[] | null>(null);
  const [err, setErr] = useState("");
  type Hist = { releases: { version: string; seen_at: string }[]; events: { action: string; from_version: string; to_version: string; user_name: string; at: string }[] };
  const [hist, setHist] = useState<{ app: StoreApp; data: Hist | null } | null>(null);
  const openHistory = async (app: StoreApp) => {
    setHist({ app, data: null });
    try { setHist({ app, data: await api.get<Hist>(`/api/store/apps/${app.id}/history`) }); }
    catch { setHist(null); }
  };
  const actionLabel: Record<string, string> = { install: "Суулгасан", enable: "Асаасан", disable: "Унтраасан", upgrade: "Шинэчлэгдсэн" };
  const [busy, setBusy] = useState("");
  const canManage = !!me.permissions["core.apps.manage"];

  const load = useCallback(async () => {
    const r = await api.get<{ apps: StoreApp[] }>("/api/store/apps");
    setApps(r.apps);
  }, []);
  useEffect(() => { void load(); }, [load]);

  const act = async (app: StoreApp, action: "install" | "enable" | "disable") => {
    setErr("");
    setBusy(app.id);
    try {
      await api.post(`/api/store/apps/${app.id}/${action}`);
      toast(action === "install" ? `${app.name} ${t("суулгагдлаа")}` : action === "enable" ? t("Асаалаа") : t("Унтраалаа"));
      await load();
      refresh(); // цэс шинэчлэгдэнэ
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    } finally {
      setBusy("");
    }
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Апп дэлгүүр")}</h1>
          <div className="sub">{t("Байгууллагадаа хэрэгтэй модулиудыг суулгана")}</div>
        </div>
      </div>
      {err && <div className="alert alert--danger">{t(err)}</div>}
      {apps && (
        <div className="app-grid">
          {apps.map((a) => (
            <div key={a.id} className="card app-card">
              <div className="app-card__head">
                <span className="app-card__icon"><Package size={22} strokeWidth={1.7} /></span>
                <div>
                  <b>{a.name}</b>
                  <div style={{ color: "var(--text-3)", fontSize: "0.8rem" }}>{a.publisher}</div>
                </div>
              </div>
              <p>{a.description}</p>
              <div className="app-card__foot">
                {a.status === "enabled" ? (
                  <>
                    <span className="badge badge--ok">{t("Суусан")}</span>
                    {canManage && (
                      <button className="btn btn--ghost btn--sm" disabled={busy === a.id}
                        onClick={() => act(a, "disable")}>
                        <Power size={14} /> {t("Унтраах")}
                      </button>
                    )}
                  </>
                ) : a.status === "disabled" ? (
                  <>
                    <span className="badge badge--warn">{t("Унтраасан")}</span>
                    {canManage && (
                      <button className="btn btn--sm" disabled={busy === a.id}
                        onClick={() => act(a, "enable")}>
                        <Power size={14} /> {t("Асаах")}
                      </button>
                    )}
                  </>
                ) : a.compiled ? (
                  canManage ? (
                    <button className="btn btn--sm" disabled={busy === a.id}
                      onClick={() => act(a, "install")}>
                      <Download size={14} /> {t("Суулгах")}
                    </button>
                  ) : (
                    <span className="badge badge--muted">{t("Суулгаагүй")}</span>
                  )
                ) : (
                  <span className="badge badge--muted" title={`nexus add ${a.short_id}`}>
                    {t("Бинарид ороогүй")} · <code>nexus add {a.short_id}</code>
                  </span>
                )}
                <button className="btn btn--ghost btn--sm" onClick={() => openHistory(a)} title={t("Хувилбарын түүх")} aria-label={`${a.name} ${t("Хувилбарын түүх")}`}>
                  <History size={14} />
                </button>
                <span className="v">v{a.version}</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {hist && (
        <div className="modal-back" onClick={() => setHist(null)} onKeyDown={(e) => e.key === "Escape" && setHist(null)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()} style={{ maxWidth: 560 }}>
            <h3>{hist.app.name} — {t("Хувилбарын түүх")}</h3>
            {!hist.data ? (
              <div style={{ color: "var(--text-3)" }}>{t("Уншиж байна…")}</div>
            ) : (
              <>
                <b style={{ display: "block", margin: "0.5rem 0" }}>{t("Нийтлэгчийн хувилбарууд")}</b>
                {hist.data.releases.length === 0 ? <div style={{ color: "var(--text-3)" }}>—</div> : (
                  <ul style={{ margin: "0 0 1rem", paddingLeft: "1.2rem", color: "var(--text-2)" }}>
                    {hist.data.releases.map((r) => (
                      <li key={r.version}><code>v{r.version}</code> · {new Date(r.seen_at).toLocaleDateString("mn-MN")}{r.version === hist.app.installed_version && <span className="badge badge--ok" style={{ marginLeft: "0.5rem" }}>{t("суусан")}</span>}</li>
                    ))}
                  </ul>
                )}
                <b style={{ display: "block", margin: "0.5rem 0" }}>{t("Энэ байгууллагад")}</b>
                {hist.data.events.length === 0 ? <div style={{ color: "var(--text-3)" }}>{t("Үйл явдал байхгүй")}</div> : (
                  <table className="table">
                    <thead><tr><th>{t("Үйлдэл")}</th><th>{t("Хувилбар")}</th><th>{t("Хэн")}</th><th>{t("Хэзээ")}</th></tr></thead>
                    <tbody>
                      {hist.data.events.map((e, i) => (
                        <tr key={i}>
                          <td>{t(actionLabel[e.action] ?? e.action)}</td>
                          <td><code>{e.from_version ? `${e.from_version} → ` : ""}{e.to_version || "—"}</code></td>
                          <td style={{ color: "var(--text-2)" }}>{e.user_name || t("систем")}</td>
                          <td style={{ color: "var(--text-2)", whiteSpace: "nowrap" }}>{new Date(e.at).toLocaleString("mn-MN")}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                )}
              </>
            )}
            <div className="modal__actions"><button className="btn btn--ghost" onClick={() => setHist(null)}>{t("Хаах")}</button></div>
          </div>
        </div>
      )}
    </>
  );
}
