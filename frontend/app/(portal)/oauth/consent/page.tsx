"use client";

import { useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { useShell } from "@/components/shell";
import { useT } from "@/lib/i18n";

// OIDC зөвшөөрлийн хуудас: /api/oauth2/authorize энд шилжүүлнэ (query
// хэвээр). Зөвшөөрвөл сервер код гаргаж клиентийн redirect_uri руу буцаана.
type Info = { client_name: string; tenant_name: string; scopes: string[]; redirect_host: string };

const scopeLabel: Record<string, string> = {
  openid: "Таныг таних (ID)",
  profile: "Нэр",
  email: "Имэйл хаяг",
  tenant: "Байгууллагын мэдээлэл",
  roles: "Байгууллага дахь role-ууд",
  offline_access: "Таныг байхгүй үед ч хандах (refresh token)",
};

export default function ConsentPage() {
  const { t } = useT();
  const { me } = useShell();
  const [query, setQuery] = useState("");
  const [info, setInfo] = useState<Info | null>(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const q = window.location.search.replace(/^\?/, "");
    setQuery(q);
    api.get<Info>(`/api/oauth2/consent?${q}`).then(setInfo).catch((e) => setErr(e instanceof ApiError ? e.message : "Хүсэлт буруу"));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const decide = async (approve: boolean) => {
    setBusy(true);
    try {
      const r = await api.post<{ redirect: string }>("/api/oauth2/consent", { approve, query });
      window.location.assign(r.redirect);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
      setBusy(false);
    }
  };

  return (
    <div className="card card__pad" style={{ maxWidth: 520, margin: "2rem auto" }}>
      <div style={{ display: "flex", gap: "0.8rem", alignItems: "center", marginBottom: "1rem" }}>
        <span className="stat__icon"><ShieldCheck size={20} /></span>
        <h1 style={{ margin: 0, fontSize: "1.2rem" }}>{t("Хандалт зөвшөөрөх")}</h1>
      </div>
      {err && <div className="alert alert--danger">{t(err)}</div>}
      {!info ? (
        !err && <div style={{ color: "var(--text-3)" }}>{t("Уншиж байна…")}</div>
      ) : (
        <>
          <p style={{ color: "var(--text-2)" }}>
            <b>{info.client_name}</b> ({info.redirect_host}) {t("систем таны")} <b>{info.tenant_name}</b> {t("байгууллагын бүртгэлээр нэвтэрч, дараах мэдээллийг авахыг хүсэж байна:")}
          </p>
          <ul style={{ color: "var(--text-2)", lineHeight: 1.9 }}>
            {info.scopes.map((s) => <li key={s}>{t(scopeLabel[s] ?? s)} <code style={{ color: "var(--text-3)" }}>{s}</code></li>)}
          </ul>
          <p style={{ color: "var(--text-3)", fontSize: "0.85rem" }}>{t("Та")}: {me.user.name} · {me.user.email}</p>
          <div className="modal__actions">
            <button className="btn btn--ghost" disabled={busy} onClick={() => decide(false)}>{t("Татгалзах")}</button>
            <button className="btn" disabled={busy} onClick={() => decide(true)}>{t("Зөвшөөрөх")}</button>
          </div>
        </>
      )}
    </div>
  );
}
