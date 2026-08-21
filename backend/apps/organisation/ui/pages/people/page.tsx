"use client";

import { useCallback, useEffect, useState } from "react";
import { Pencil, Users } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

type Person = {
  membership_id: string; user_id: string; name: string;
  department_id: string | null; department_name: string; job_title: string;
};
type Dept = { id: string; name: string; active: boolean };
type Form = { membership_id: string; name: string; department_id: string; job_title: string };

export default function PeoplePage() {
  const { t } = useT();
  const { me } = useShell();
  const manage = !!me.permissions["organisation.manage"];
  const [people, setPeople] = useState<Person[] | null>(null);
  const [depts, setDepts] = useState<Dept[]>([]);
  const [form, setForm] = useState<Form | null>(null);
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    const [p, d] = await Promise.all([
      api.get<{ people: Person[] }>("/api/apps/organisation/people"),
      api.get<{ departments: Dept[] }>("/api/apps/organisation/departments"),
    ]);
    setPeople(p.people); setDepts(d.departments.filter((x) => x.active));
  }, []);
  useEffect(() => { void load(); }, [load]);

  const save = async () => {
    if (!form) return;
    setErr("");
    try {
      await api.put(`/api/apps/organisation/people/${form.membership_id}`,
        { department_id: form.department_id || null, job_title: form.job_title });
      setForm(null); toast(t("Хадгалагдлаа")); await load();
    } catch (e) { setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа")); }
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Ажилтнууд")}</h1>
          <div className="sub">{t("Гишүүн бүрийн хэлтэс, албан тушаал")}</div>
        </div>
      </div>
      <div className="card">
        {people === null ? (
          <div className="empty">{t("Уншиж байна…")}</div>
        ) : people.length === 0 ? (
          <div className="empty"><Users size={36} strokeWidth={1.4} /><b>{t("Гишүүн байхгүй")}</b></div>
        ) : (
          <table className="table">
            <thead><tr><th>{t("Нэр")}</th><th>{t("Хэлтэс")}</th><th>{t("Албан тушаал")}</th><th></th></tr></thead>
            <tbody>
              {people?.map((p) => (
                <tr key={p.membership_id}>
                  <td><b style={{ fontWeight: 600 }}>{p.name}</b></td>
                  <td>{p.department_name || "—"}</td>
                  <td>{p.job_title || "—"}</td>
                  <td style={{ textAlign: "right" }}>
                    {manage && (
                      <button className="btn btn--ghost btn--sm" aria-label={`${p.name} ${t("засах")}`}
                        onClick={() => { setErr(""); setForm({ membership_id: p.membership_id, name: p.name, department_id: p.department_id ?? "", job_title: p.job_title }); }}>
                        <Pencil size={13} />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {form && (
        <div className="modal-back" onClick={() => setForm(null)} onKeyDown={(e) => e.key === "Escape" && setForm(null)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
            <h3>{form.name}</h3>
            {err && <div className="alert alert--danger">{t(err)}</div>}
            <div className="field"><label>{t("Хэлтэс")}</label>
              <select value={form.department_id} onChange={(e) => setForm({ ...form, department_id: e.target.value })}>
                <option value="">—</option>
                {depts.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
              </select></div>
            <div className="field"><label>{t("Албан тушаал")}</label>
              <input value={form.job_title} maxLength={120} autoFocus onChange={(e) => setForm({ ...form, job_title: e.target.value })} /></div>
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
