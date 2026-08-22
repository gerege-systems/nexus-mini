"use client";

import { useCallback, useEffect, useState } from "react";
import { Copy, KeyRound, Pencil, Plus, Trash2 } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

// OAuth2/OIDC клиентүүд — гадны систем энэ байгууллагын хэрэглэгчээр нэвтрэх.
type Client = { id: string; client_id: string; name: string; public: boolean; redirect_uris: string[]; post_logout_uris: string[]; scopes: string; created_at: string };
type Form = { id?: string; name: string; public: boolean; redirect_uris: string; post_logout_uris: string; scopes: string };
const empty: Form = { name: "", public: false, redirect_uris: "", post_logout_uris: "", scopes: "openid profile email" };
const ALL_SCOPES = ["openid", "profile", "email", "tenant", "roles", "offline_access"];

export default function SSOClientsPage() {
  const { t } = useT();
  const [clients, setClients] = useState<Client[] | null>(null);
  const [issuer, setIssuer] = useState("");
  const [form, setForm] = useState<Form | null>(null);
  const [secret, setSecret] = useState<{ client_id: string; client_secret: string } | null>(null);
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    const r = await api.get<{ clients: Client[]; issuer: string }>("/api/sso-clients");
    setClients(r.clients); setIssuer(r.issuer);
  }, []);
  useEffect(() => { void load(); }, [load]);

  const lines = (s: string) => s.split(/\n|,/).map((x) => x.trim()).filter(Boolean);
  const save = async () => {
    if (!form) return;
    setErr("");
    const body = { name: form.name, public: form.public, redirect_uris: lines(form.redirect_uris), post_logout_uris: lines(form.post_logout_uris), scopes: form.scopes };
    try {
      if (form.id) {
        await api.put(`/api/sso-clients/${form.id}`, body);
        toast(t("Хадгалагдлаа"));
      } else {
        const r = await api.post<{ client_id: string; client_secret: string }>("/api/sso-clients", body);
        if (r.client_secret) setSecret(r);
        toast(t("Клиент үүслээ"));
      }
      setForm(null); await load();
    } catch (e) { setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа")); }
  };
  const remove = async (c: Client) => {
    if (!confirm(`"${c.name}" ${t("клиентийг устгах уу? Олгосон бүх токен хүчингүй болно.")}`)) return;
    try { await api.del(`/api/sso-clients/${c.id}`); toast(t("Устгагдлаа")); await load(); }
    catch (e) { toast(e instanceof ApiError ? t(e.message) : t("Алдаа гарлаа")); }
  };
  const copy = (s: string) => { void navigator.clipboard?.writeText(s); toast(t("Хуулагдлаа")); };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("SSO клиентүүд")}</h1>
          <div className="sub">{t("Гадны систем энэ байгууллагын бүртгэлээр нэвтрэх (OpenID Connect)")}</div>
        </div>
        <div className="spacer" />
        <button className="btn" onClick={() => { setErr(""); setForm(empty); }}><Plus size={16} /> {t("Клиент нэмэх")}</button>
      </div>

      <div className="card card__pad" style={{ marginBottom: "1rem", fontSize: "0.9rem", color: "var(--text-2)" }}>
        <b>{t("Issuer")}:</b> <code>{issuer}</code>{" "}
        <button className="btn btn--ghost btn--sm" onClick={() => copy(issuer)} aria-label={t("Хуулах")}><Copy size={13} /></button>
        <div style={{ marginTop: "0.3rem" }}>{t("Discovery")}: <code>{issuer}/.well-known/openid-configuration</code> · PKCE S256 {t("заавал")} · RS256</div>
      </div>

      <div className="card">
        {clients === null ? <div className="empty">{t("Уншиж байна…")}</div> : clients.length === 0 ? (
          <div className="empty"><KeyRound size={36} strokeWidth={1.4} /><b>{t("Клиент байхгүй")}</b>{t("Гадны системээ бүртгээд client_id авна")}</div>
        ) : (
          <table className="table">
            <thead><tr><th>{t("Нэр")}</th><th>client_id</th><th>{t("Төрөл")}</th><th>Redirect</th><th>Scope</th><th></th></tr></thead>
            <tbody>
              {clients.map((c) => (
                <tr key={c.id}>
                  <td><b style={{ fontWeight: 600 }}>{c.name}</b></td>
                  <td><code>{c.client_id}</code> <button className="btn btn--ghost btn--sm" onClick={() => copy(c.client_id)} aria-label={t("Хуулах")}><Copy size={12} /></button></td>
                  <td>{c.public ? <span className="badge badge--muted">public · PKCE</span> : <span className="badge badge--accent">confidential</span>}</td>
                  <td style={{ fontSize: "0.82rem", color: "var(--text-2)" }}>{c.redirect_uris.map((u) => <div key={u}>{u}</div>)}</td>
                  <td style={{ fontSize: "0.82rem", color: "var(--text-2)" }}>{c.scopes}</td>
                  <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                    <button className="btn btn--ghost btn--sm" aria-label={`${c.name} ${t("засах")}`}
                      onClick={() => { setErr(""); setForm({ id: c.id, name: c.name, public: c.public, redirect_uris: c.redirect_uris.join("\n"), post_logout_uris: c.post_logout_uris.join("\n"), scopes: c.scopes }); }}>
                      <Pencil size={13} />
                    </button>{" "}
                    <button className="btn btn--ghost btn--sm" aria-label={`${c.name} ${t("устгах")}`} onClick={() => remove(c)}><Trash2 size={13} /></button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {secret && (
        <div className="modal-back" onClick={() => setSecret(null)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
            <h3>{t("Клиентийн нууц үг — нэг л удаа харагдана")}</h3>
            <div className="field"><label>client_id</label><input readOnly value={secret.client_id} onFocus={(e) => e.target.select()} /></div>
            <div className="field"><label>client_secret</label><input readOnly value={secret.client_secret} onFocus={(e) => e.target.select()} /></div>
            <div className="hint">{t("Хадгалаад хаа — дахин харуулахгүй, алдвал клиентээ шинээр үүсгэнэ.")}</div>
            <div className="modal__actions"><button className="btn" onClick={() => setSecret(null)}>{t("Хадгалсан")}</button></div>
          </div>
        </div>
      )}

      {form && (
        <div className="modal-back" onClick={() => setForm(null)} onKeyDown={(e) => e.key === "Escape" && setForm(null)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
            <h3>{form.id ? t("Клиент засах") : t("Клиент нэмэх")}</h3>
            {err && <div className="alert alert--danger">{t(err)}</div>}
            <div className="field"><label>{t("Нэр")}</label>
              <input value={form.name} autoFocus onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="Bold ERP" /></div>
            {!form.id && (
              <div className="field"><label style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                <input type="checkbox" checked={form.public} style={{ width: "auto" }} onChange={(e) => setForm({ ...form, public: e.target.checked })} />
                {t("Public клиент (SPA/mobile — secret-гүй, PKCE)")}
              </label></div>
            )}
            <div className="field"><label>{t("Redirect URI-ууд (мөр тус бүр)")}</label>
              <textarea rows={3} value={form.redirect_uris} onChange={(e) => setForm({ ...form, redirect_uris: e.target.value })} placeholder="https://erp.bold.mn/auth/callback" /></div>
            <div className="field"><label>{t("Logout-ын дараах URI-ууд")}</label>
              <textarea rows={2} value={form.post_logout_uris} onChange={(e) => setForm({ ...form, post_logout_uris: e.target.value })} /></div>
            <div className="field"><label>Scope</label>
              <span style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
                {ALL_SCOPES.map((s) => {
                  const on = form.scopes.split(" ").includes(s);
                  return <button key={s} type="button" className={`um__pill${on ? " is-on" : ""}`}
                    onClick={() => setForm({ ...form, scopes: (on ? form.scopes.split(" ").filter((x) => x !== s) : [...form.scopes.split(" ").filter(Boolean), s]).join(" ") })}>{s}</button>;
                })}
              </span></div>
            <div className="modal__actions">
              <button className="btn btn--ghost" onClick={() => setForm(null)}>{t("Болих")}</button>
              <button className="btn" onClick={save}>{t("Хадгалах")}</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
