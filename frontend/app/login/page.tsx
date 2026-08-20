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

  // Аль хэдийн нэвтэрсэн хүнээс дахин нууц үг нэхэхгүй — session хүчинтэй
  // бол шууд портал руу.
  useEffect(() => {
    api.get("/api/me").then(() => router.replace("/dashboard")).catch(() => {});
  }, [router]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/api/login", { email, password });
      router.replace("/dashboard");
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : "Алдаа гарлаа");
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
        <div className="foot">
          {t("Бүртгэлгүй юу?")} <Link href="/signup">{t("Байгууллагаа бүртгүүлэх")}</Link>
        </div>
      </div>
    </div>
  );
}
