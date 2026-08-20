"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";

const slugify = (s: string) =>
  s.toLowerCase().replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "").slice(0, 40);

export default function SignupPage() {
  const [form, setForm] = useState({
    name: "", email: "", password: "", tenant_name: "", tenant_slug: "",
  });
  const [slugTouched, setSlugTouched] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const router = useRouter();

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/api/signup", form);
      // Шинэ байгууллага — app store-оос эхэлнэ.
      router.replace("/store");
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
            <h1>Байгууллагаа бүртгүүлэх</h1>
            <p className="sub">Бүртгүүлмэгц app store-оос модулиа сонгоно</p>
          </div>
        </div>
        {err && <div className="alert alert--danger">{err}</div>}
        <form onSubmit={submit}>
          <div className="field">
            <label>Таны нэр</label>
            <input value={form.name} autoFocus required
              onChange={(e) => setForm({ ...form, name: e.target.value })} />
          </div>
          <div className="field">
            <label>Имэйл</label>
            <input type="email" value={form.email} required
              onChange={(e) => setForm({ ...form, email: e.target.value })} />
          </div>
          <div className="field">
            <label>Нууц үг (8+)</label>
            <input type="password" value={form.password} required minLength={8}
              onChange={(e) => setForm({ ...form, password: e.target.value })} />
          </div>
          <div className="field">
            <label>Байгууллагын нэр</label>
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
            <label>Богино нэр (slug)</label>
            <input value={form.tenant_slug} required
              onChange={(e) => {
                setSlugTouched(true);
                setForm({ ...form, tenant_slug: slugify(e.target.value) });
              }} />
            <div className="hint">Жижиг латин үсэг, тоо, зураас</div>
          </div>
          <button className="btn" style={{ width: "100%", justifyContent: "center" }} disabled={busy}>
            Бүртгүүлэх
          </button>
        </form>
        <div className="foot">
          Бүртгэлтэй юу? <Link href="/login">Нэвтрэх</Link>
        </div>
      </div>
    </div>
  );
}
