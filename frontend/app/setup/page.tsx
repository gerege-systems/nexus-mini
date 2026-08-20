"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";

// Эхний ажиллуулалт: платформын админ үүсгэх wizard. Setup аль хэдийн
// хийгдсэн бол login руу явуулна.
export default function SetupPage() {
  const [form, setForm] = useState({ name: "", email: "", password: "" });
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [ready, setReady] = useState(false);
  const router = useRouter();

  useEffect(() => {
    void api.get<{ done: boolean }>("/api/setup").then((r) => {
      if (r.done) router.replace("/login");
      else setReady(true);
    });
  }, [router]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/api/setup", form);
      router.replace("/dashboard");
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : "Алдаа гарлаа");
      setBusy(false);
    }
  };

  if (!ready) return null;

  return (
    <div className="auth-wrap">
      <div className="auth-card">
        <div className="brand-row">
          <span className="brand-square">N</span>
          <div>
            <h1>Тавтай морил 👋</h1>
            <p className="sub">nexus-mini анх удаа асаж байна — платформын админаа үүсгэе</p>
          </div>
        </div>
        {err && <div className="alert alert--danger">{err}</div>}
        <form onSubmit={submit}>
          <div className="field">
            <label>Нэр</label>
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
          <button className="btn" style={{ width: "100%", justifyContent: "center" }} disabled={busy}>
            Эхлүүлэх
          </button>
        </form>
        <div className="foot">
          Энэ хэрэглэгч бүх байгууллага, каталогийг удирдах эрхтэй болно
        </div>
      </div>
    </div>
  );
}
