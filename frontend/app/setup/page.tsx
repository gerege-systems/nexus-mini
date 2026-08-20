"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { slugify } from "@/lib/slug";

// Эхний ажиллуулалт, 2 алхам: ① платформын админаа үүсгэнэ ② өөрийн
// байгууллагаа үүсгээд шууд ажиллаж эхэлнэ. Setup хийгдсэн бол login руу.
export default function SetupPage() {
  const [step, setStep] = useState<1 | 2>(1);
  const [admin, setAdmin] = useState({ name: "", email: "", password: "" });
  const [org, setOrg] = useState({ name: "", slug: "" });
  const [slugTouched, setSlugTouched] = useState(false);
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

  const submitAdmin = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      await api.post("/api/setup", admin);
      setBusy(false);
      setStep(2);
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : "Алдаа гарлаа");
      setBusy(false);
    }
  };

  const submitOrg = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      const r = await api.post<{ tenant_id: string }>("/api/tenants", org);
      await api.post("/api/session/tenant", { tenant_id: r.tenant_id });
      router.replace("/store");
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
            <h1>{step === 1 ? "Тавтай морил 👋" : "Байгууллагаа үүсгэе"}</h1>
            <p className="sub">
              {step === 1
                ? "nexus-mini анх удаа асаж байна — админ эрхээ бүртгүүлье (1/2)"
                : "Танай ажлын талбар болох байгууллага (2/2)"}
            </p>
          </div>
        </div>
        {err && <div className="alert alert--danger">{err}</div>}

        {step === 1 ? (
          <form onSubmit={submitAdmin}>
            <div className="field">
              <label>Нэр</label>
              <input value={admin.name} autoFocus required
                onChange={(e) => setAdmin({ ...admin, name: e.target.value })} />
            </div>
            <div className="field">
              <label>Имэйл</label>
              <input type="email" value={admin.email} required
                onChange={(e) => setAdmin({ ...admin, email: e.target.value })} />
            </div>
            <div className="field">
              <label>Нууц үг (8+)</label>
              <input type="password" value={admin.password} required minLength={8}
                onChange={(e) => setAdmin({ ...admin, password: e.target.value })} />
            </div>
            <button className="btn" style={{ width: "100%", justifyContent: "center" }} disabled={busy}>
              Үргэлжлүүлэх
            </button>
            <div className="foot">
              Энэ хэрэглэгч платформын бүх тохиргоог удирдах эрхтэй болно
            </div>
          </form>
        ) : (
          <form onSubmit={submitOrg}>
            <div className="field">
              <label>Байгууллагын нэр</label>
              <input value={org.name} autoFocus required
                onChange={(e) =>
                  setOrg({
                    name: e.target.value,
                    slug: slugTouched ? org.slug : slugify(e.target.value),
                  })
                } />
            </div>
            <div className="field">
              <label>Богино нэр (slug)</label>
              <input value={org.slug} required
                onChange={(e) => {
                  setSlugTouched(true);
                  setOrg({ ...org, slug: slugify(e.target.value) });
                }} />
              <div className="hint">Жижиг латин үсэг, тоо, зураас</div>
            </div>
            <button className="btn" style={{ width: "100%", justifyContent: "center" }} disabled={busy}>
              Эхлүүлэх
            </button>
            <div className="foot">
              Дараа нь app store-оос хэрэгтэй модулиудаа суулгана
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
