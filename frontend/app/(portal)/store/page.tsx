"use client";

import { useCallback, useEffect, useState } from "react";
import { Download, Package, Power } from "lucide-react";
import { api, ApiError, type StoreApp } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

export default function StorePage() {
  const { me, refresh } = useShell();
  const { t } = useT();
  const [apps, setApps] = useState<StoreApp[] | null>(null);
  const [err, setErr] = useState("");
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
                  <span className="badge badge--muted" title={`nexus-mini add ${a.id}`}>
                    {t("Бинарид ороогүй")}
                  </span>
                )}
                <span className="v">v{a.version}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </>
  );
}
