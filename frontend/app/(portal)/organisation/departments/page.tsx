"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Building2, Pencil, Plus, Trash2 } from "lucide-react";
import { api, ApiError } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

type Dept = {
  id: string; code: string; name: string; parent_id: string | null;
  manager_membership_id: string | null; manager_name: string; active: boolean; people: number;
};
type Person = { membership_id: string; name: string };
type Form = { id?: string; code: string; name: string; parent_id: string; manager_membership_id: string; active: boolean };
const empty: Form = { code: "", name: "", parent_id: "", manager_membership_id: "", active: true };

// Хавтгай жагсаалтыг мод болгож, гүнтэй нь дарааллуулна.
function flatten(list: Dept[]): { d: Dept; depth: number }[] {
  const byParent = new Map<string | null, Dept[]>();
  for (const d of list) {
    const k = d.parent_id && list.some((x) => x.id === d.parent_id) ? d.parent_id : null;
    byParent.set(k, [...(byParent.get(k) ?? []), d]);
  }
  const out: { d: Dept; depth: number }[] = [];
  const walk = (parent: string | null, depth: number) => {
    for (const d of byParent.get(parent) ?? []) { out.push({ d, depth }); if (depth < 32) walk(d.id, depth + 1); }
  };
  walk(null, 0);
  return out;
}

export default function DepartmentsPage() {
  const { t } = useT();
  const { me } = useShell();
  const manage = !!me.permissions["organisation.manage"];
  const [depts, setDepts] = useState<Dept[] | null>(null);
  const [people, setPeople] = useState<Person[]>([]);
  const [form, setForm] = useState<Form | null>(null);
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    const [d, p] = await Promise.all([
      api.get<{ departments: Dept[] }>("/api/apps/organisation/departments"),
      api.get<{ people: Person[] }>("/api/apps/organisation/people").catch(() => ({ people: [] })),
    ]);
    setDepts(d.departments); setPeople(p.people);
  }, []);
  useEffect(() => { void load(); }, [load]);
  const tree = useMemo(() => (depts ? flatten(depts) : []), [depts]);

  const save = async () => {
    if (!form) return;
    setErr("");
    const body = { code: form.code, name: form.name, parent_id: form.parent_id || null,
      manager_membership_id: form.manager_membership_id || null, active: form.active };
    try {
      if (form.id) await api.put(`/api/apps/organisation/departments/${form.id}`, body);
      else await api.post("/api/apps/organisation/departments", body);
      setForm(null); toast(t("Хадгалагдлаа")); await load();
    } catch (e) { setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа")); }
  };
  const remove = async (d: Dept) => {
    if (!confirm(`"${d.name}" ${t("хэлтсийг устгах уу? Харьяа нэгжүүд дээд түвшингүй болно.")}`)) return;
    await api.del(`/api/apps/organisation/departments/${d.id}`);
    toast(t("Устгагдлаа")); await load();
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Хэлтэс, нэгж")}</h1>
          <div className="sub">{t("Байгууллагын бүтцийн мод")}</div>
        </div>
        <div className="spacer" />
        {manage && <button className="btn" onClick={() => { setErr(""); setForm(empty); }}><Plus size={16} /> {t("Нэгж нэмэх")}</button>}
      </div>

      <div className="card">
        {depts && depts.length === 0 ? (
          <div className="empty">
            <Building2 size={36} strokeWidth={1.4} />
            <b>{t("Нэгж байхгүй")}</b>
            {t("Эхний хэлтэс/нэгжээ үүсгээрэй")}
          </div>
        ) : (
          <table className="table">
            <thead><tr><th>{t("Нэгж")}</th><th>{t("Код")}</th><th>{t("Менежер")}</th><th>{t("Ажилтан")}</th><th></th></tr></thead>
            <tbody>
              {tree.map(({ d, depth }) => (
                <tr key={d.id} style={{ opacity: d.active ? 1 : 0.55 }}>
                  <td style={{ paddingLeft: `${1.25 + depth * 1.25}rem` }}>
                    <b style={{ fontWeight: 600 }}>{d.name}</b>
                    {!d.active && <span className="badge badge--muted" style={{ marginLeft: "0.5rem" }}>{t("идэвхгүй")}</span>}
                  </td>
                  <td><code>{d.code}</code></td>
                  <td style={{ color: "var(--text-2)" }}>{d.manager_name || "—"}</td>
                  <td>{d.people}</td>
                  <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                    {manage && (
                      <>
                        <button className="btn btn--ghost btn--sm" aria-label={`${d.name} ${t("засах")}`}
                          onClick={() => { setErr(""); setForm({ id: d.id, code: d.code, name: d.name, parent_id: d.parent_id ?? "", manager_membership_id: d.manager_membership_id ?? "", active: d.active }); }}>
                          <Pencil size={13} />
                        </button>{" "}
                        <button className="btn btn--ghost btn--sm" aria-label={`${d.name} ${t("устгах")}`} onClick={() => remove(d)}>
                          <Trash2 size={13} />
                        </button>
                      </>
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
            <h3>{form.id ? t("Нэгж засах") : t("Нэгж нэмэх")}</h3>
            {err && <div className="alert alert--danger">{t(err)}</div>}
            <div className="field"><label>{t("Нэр")}</label>
              <input value={form.name} autoFocus onChange={(e) => setForm({ ...form, name: e.target.value })} /></div>
            <div className="field"><label>{t("Код")}</label>
              <input value={form.code} placeholder="hr, it, sales…" onChange={(e) => setForm({ ...form, code: e.target.value })} /></div>
            <div className="field"><label>{t("Дээд нэгж")}</label>
              <select value={form.parent_id} onChange={(e) => setForm({ ...form, parent_id: e.target.value })}>
                <option value="">{t("— байхгүй (дээд түвшин) —")}</option>
                {tree.filter((x) => x.d.id !== form.id).map(({ d, depth }) => (
                  <option key={d.id} value={d.id}>{" ".repeat(depth * 3)}{d.name}</option>
                ))}
              </select></div>
            <div className="field"><label>{t("Менежер")}</label>
              <select value={form.manager_membership_id} onChange={(e) => setForm({ ...form, manager_membership_id: e.target.value })}>
                <option value="">—</option>
                {people.map((p) => <option key={p.membership_id} value={p.membership_id}>{p.name}</option>)}
              </select></div>
            {form.id && (
              <div className="field"><label style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
                <input type="checkbox" checked={form.active} onChange={(e) => setForm({ ...form, active: e.target.checked })} style={{ width: "auto" }} />
                {t("Идэвхтэй")}
              </label></div>
            )}
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
