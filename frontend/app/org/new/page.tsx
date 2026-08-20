"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { api, ApiError } from "@/lib/api";
import { slugify } from "@/lib/slug";

// Нэвтэрсэн ч байгууллагагүй (эсвэл шинээр нэмэх) хэрэглэгчид.
export default function NewOrgPage() {
  const [org, setOrg] = useState({ name: "", slug: "" });
  const [slugTouched, setSlugTouched] = useState(false);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const router = useRouter();

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    setBusy(true);
    try {
      const r = await api.post<{ tenant_id: string }>("/api/tenants", org);
      await api.post("/api/session/tenant", { tenant_id: r.tenant_id });
      router.replace("/store");
    } catch (ex) {
      if (ex instanceof ApiError && ex.status === 401) router.replace("/login");
      else setErr(ex instanceof ApiError ? ex.message : "Алдаа гарлаа");
      setBusy(false);
    }
  };

  return (
    <div className="auth-wrap">
      <div className="auth-card">
        <div className="brand-row">
          <span className="brand-square">N</span>
          <div>
            <h1>Байгууллага үүсгэх</h1>
            <p className="sub">Ажлын талбараа үүсгээд store-оос модулиа сонгоно</p>
          </div>
        </div>
        {err && <div className="alert alert--danger">{err}</div>}
        <form onSubmit={submit}>
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
            Үүсгэх
          </button>
        </form>
      </div>
    </div>
  );
}
