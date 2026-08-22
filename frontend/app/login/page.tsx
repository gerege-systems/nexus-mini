"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { useT } from "@/lib/i18n";

export default function LoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const router = useRouter();
  const { t } = useT();
  // ?next= — зөвхөн энэ сайтын харьцангуй зам (open redirect хаалттай).
  const [next, setNext] = useState("/dashboard");
  const [providers, setProviders] = useState<{ key: string; name: string }[]>([]);
  useEffect(() => {
    const sp = new URLSearchParams(window.location.search);
    const n = sp.get("next") || "";
    if (n.startsWith("/") && !n.startsWith("//")) setNext(n);
    const e = sp.get("error");
    if (e) setErr(e);
    api.get<{ providers: { key: string; name: string }[] }>("/api/auth/sso/providers").then((r) => setProviders(r.providers)).catch(() => {});
  }, []);

  // Аль хэдийн нэвтэрсэн хүнээс дахин нууц үг нэхэхгүй — session хүчинтэй
  // бол шууд портал руу.
  useEffect(() => {
    api.get("/api/me").then(() => { window.location.assign(next); }).catch(() => {});
  }, [router, next]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/api/login", { email, password });
      // /api/... (OIDC authorize) руу буцах бол бүтэн navigation.
      if (next.startsWith("/api/")) window.location.assign(next);
      else router.replace(next);
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : t("Алдаа гарлаа"));
      setBusy(false);
    }
  };

  return (
    <div className="auth-wrap">
      <div className="auth-card">
        <div className="brand-row">
          <span className="brand-square">N</span>
          <div>
            <h1>{t("Нэвтрэх")}</h1>
            <p className="sub">{t("nexus-mini ажлын талбар")}</p>
          </div>
        </div>
        {err && <div className="alert alert--danger">{t(err)}</div>}
        <form onSubmit={submit}>
          <div className="field">
            <label>{t("Имэйл")}</label>
            <input type="email" value={email} autoFocus required
              onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div className="field">
            <label>{t("Нууц үг")}</label>
            <input type="password" value={password} required
              onChange={(e) => setPassword(e.target.value)} />
          </div>
          <button className="btn" style={{ width: "100%", justifyContent: "center" }} disabled={busy}>
            {t("Нэвтрэх")}
          </button>
        </form>
        {providers.length > 0 && (
          <div style={{ marginTop: "1rem", display: "grid", gap: "0.5rem" }}>
            <div style={{ textAlign: "center", color: "var(--text-3)", fontSize: "0.8rem" }}>{t("эсвэл")}</div>
            {providers.map((p) => (
              <a key={p.key} className="btn btn--ghost" style={{ justifyContent: "center" }}
                href={`/api/auth/sso/${p.key}/start?next=${encodeURIComponent(next)}`}>
                {p.key === "google" ? t("Google-ээр нэвтрэх") : `${p.name} — ${t("SSO-оор нэвтрэх")}`}
              </a>
            ))}
          </div>
        )}
        <div className="foot">
          {t("Бүртгэлгүй юу?")} <Link href="/signup">{t("Байгууллагаа бүртгүүлэх")}</Link>
        </div>
      </div>
    </div>
  );
}
