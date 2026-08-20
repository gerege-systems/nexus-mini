"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError, type Me } from "@/lib/api";
import { useT } from "@/lib/i18n";

// Зөвхөн платформын админ нэвтэрнэ — жирийн хэрэглэгчийг буцаана.
export default function AdminLoginPage() {
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const router = useRouter();
  const { t } = useT();

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/api/login", { email, password });
      const me = await api.get<Me>("/api/me");
      if (!me.user.platform_admin) {
        await api.post("/api/logout");
        setErr(t("Энэ систем зөвхөн платформын админд зориулагдсан"));
        setBusy(false);
        return;
      }
      router.replace("/");
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
            <h1>{t("Платформын админ")}</h1>
            <p className="sub">{t("nexus-mini удирдлагын систем")}</p>
          </div>
        </div>
        {err && <div className="alert alert--danger">{err}</div>}
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
      </div>
    </div>
  );
}
