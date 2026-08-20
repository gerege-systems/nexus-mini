"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { slugify } from "@/lib/slug";
import { useT } from "@/lib/i18n";

export default function SignupPage() {
  const [form, setForm] = useState({
    name: "", email: "", password: "", tenant_name: "", tenant_slug: "",
  });
  const [slugTouched, setSlugTouched] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const router = useRouter();
  const { t } = useT();

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/api/signup", form);
      // Шинэ байгууллага — app store-оос эхэлнэ.
      router.replace("/store");
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
            <h1>{t("Байгууллагаа бүртгүүлэх")}</h1>
            <p className="sub">{t("Бүртгүүлмэгц app store-оос модулиа сонгоно")}</p>
          </div>
        </div>
        {err && <div className="alert alert--danger">{t(err)}</div>}
        <form onSubmit={submit}>
          <div className="field">
            <label>{t("Таны нэр")}</label>
            <input value={form.name} autoFocus required
              onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("Имэйл")}</label>
            <input type="email" value={form.email} required
              onChange={(e) => setForm({ ...form, email: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("Нууц үг (8+)")}</label>
            <input type="password" value={form.password} required minLength={8}
              onChange={(e) => setForm({ ...form, password: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("Байгууллагын нэр")}</label>
            <input value={form.tenant_name} required
              onChange={(e) =>
                setForm({
                  ...form,
                  tenant_name: e.target.value,
                  tenant_slug: slugTouched ? form.tenant_slug : slugify(e.target.value),
                })
              } />
          </div>
          <div className="field">
            <label>{t("Богино нэр (slug)")}</label>
            <input value={form.tenant_slug} required
              onChange={(e) => {
                setSlugTouched(true);
                setForm({ ...form, tenant_slug: slugify(e.target.value) });
              }} />
            <div className="hint">{t("Жижиг латин үсэг, тоо, зураас")}</div>
          </div>
          <button className="btn" style={{ width: "100%", justifyContent: "center" }} disabled={busy}>
            {t("Бүртгүүлэх")}
          </button>
        </form>
        <div className="foot">
          {t("Бүртгэлтэй юу?")} <Link href="/login">{t("Нэвтрэх")}</Link>
        </div>
      </div>
    </div>
  );
}
