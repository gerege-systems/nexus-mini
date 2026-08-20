"use client";

import { useCallback, useEffect, useState } from "react";
import { MonitorSmartphone, Pencil, Plus, Search, Trash2 } from "lucide-react";
import { api, ApiError, type Device } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";

const statusMn: Record<Device["status"], { label: string; cls: string }> = {
  active: { label: "Ашиглагдаж байгаа", cls: "badge--ok" },
  repair: { label: "Засварт", cls: "badge--warn" },
  lost: { label: "Алдагдсан", cls: "badge--danger" },
  retired: { label: "Хассан", cls: "badge--muted" },
};

type FormState = {
  id?: string;
  name: string;
  kind: string;
  serial: string;
  status: Device["status"];
  note: string;
};

const empty: FormState = { name: "", kind: "", serial: "", status: "active", note: "" };

export default function DevicesPage() {
  const { me } = useShell();
  const [devices, setDevices] = useState<Device[] | null>(null);
  const [q, setQ] = useState("");
  const [form, setForm] = useState<FormState | null>(null);
  const [err, setErr] = useState("");
  const manage = me.permissions["devices.manage"]; // undefined | "all" | "own"

  const load = useCallback(async (query: string) => {
    const r = await api.get<{ devices: Device[] }>(
      `/api/apps/devices/?q=${encodeURIComponent(query)}`
    );
    setDevices(r.devices);
  }, []);
  useEffect(() => {
    const t = setTimeout(() => void load(q), q ? 250 : 0);
    return () => clearTimeout(t);
  }, [q, load]);

  const canEdit = (d: Device) =>
    manage === "all" || (manage === "own" && d.created_by === me.user.id);

  const save = async () => {
    if (!form) return;
    setErr("");
    const body = {
      name: form.name, kind: form.kind, serial: form.serial,
      status: form.status, note: form.note,
    };
    try {
      if (form.id) await api.put(`/api/apps/devices/${form.id}`, body);
      else await api.post("/api/apps/devices/", body);
      setForm(null);
      toast(form.id ? "Хадгалагдлаа" : "Бүртгэгдлээ");
      await load(q);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : "Алдаа гарлаа");
    }
  };

  const remove = async (d: Device) => {
    if (!confirm(`"${d.name}" төхөөрөмжийг устгах уу?`)) return;
    await api.del(`/api/apps/devices/${d.id}`);
    toast("Устгагдлаа");
    await load(q);
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Төхөөрөмжүүд</h1>
          <div className="sub">Байгууллагын төхөөрөмжийн бүртгэл</div>
        </div>
        <div className="spacer" />
        <div style={{ position: "relative" }}>
          <Search size={15} style={{ position: "absolute", left: "0.6rem", top: "0.62rem", color: "var(--text-3)" }} />
          <input placeholder="Хайх…" value={q} onChange={(e) => setQ(e.target.value)}
            style={{ padding: "0.45rem 0.7rem 0.45rem 2rem", border: "1px solid var(--border)", borderRadius: 8, background: "var(--surface)", color: "var(--text)" }} />
        </div>
        {manage && (
          <button className="btn" onClick={() => { setErr(""); setForm(empty); }}>
            <Plus size={16} /> Бүртгэх
          </button>
        )}
      </div>

      <div className="card">
        {devices && devices.length === 0 ? (
          <div className="empty">
            <MonitorSmartphone size={36} strokeWidth={1.4} />
            <b>Бүртгэл хоосон</b>
            Эхний төхөөрөмжөө бүртгээрэй
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr>
                <th>Нэр</th><th>Төрөл</th><th>Сериал</th><th>Статус</th>
                <th>Бүртгэсэн</th><th></th>
              </tr>
            </thead>
            <tbody>
              {devices?.map((d) => (
                <tr key={d.id}>
                  <td><b style={{ fontWeight: 600 }}>{d.name}</b>
                    {d.note && <div style={{ color: "var(--text-3)", fontSize: "0.82rem" }}>{d.note}</div>}
                  </td>
                  <td>{d.kind || "—"}</td>
                  <td style={{ fontFamily: "ui-monospace, monospace", fontSize: "0.84rem" }}>{d.serial}</td>
                  <td><span className={`badge ${statusMn[d.status].cls}`}>{statusMn[d.status].label}</span></td>
                  <td style={{ color: "var(--text-2)" }}>{d.owner_name}</td>
                  <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                    {canEdit(d) && (
                      <>
                        <button className="btn btn--ghost btn--sm"
                          onClick={() => { setErr(""); setForm({ id: d.id, name: d.name, kind: d.kind, serial: d.serial, status: d.status, note: d.note }); }}>
                          <Pencil size={13} />
                        </button>{" "}
                        <button className="btn btn--ghost btn--sm" onClick={() => remove(d)}>
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
        <div className="modal-back" onClick={() => setForm(null)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>{form.id ? "Төхөөрөмж засах" : "Төхөөрөмж бүртгэх"}</h3>
            {err && <div className="alert alert--danger">{err}</div>}
            <div className="field">
              <label>Нэр</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })}
                placeholder="Dell Latitude 5540" autoFocus />
            </div>
            <div className="field">
              <label>Төрөл</label>
              <input value={form.kind} onChange={(e) => setForm({ ...form, kind: e.target.value })}
                placeholder="laptop, printer…" />
            </div>
            <div className="field">
              <label>Сериал</label>
              <input value={form.serial} onChange={(e) => setForm({ ...form, serial: e.target.value })} />
            </div>
            <div className="field">
              <label>Статус</label>
              <select value={form.status}
                onChange={(e) => setForm({ ...form, status: e.target.value as Device["status"] })}>
                {Object.entries(statusMn).map(([k, v]) => (
                  <option key={k} value={k}>{v.label}</option>
                ))}
              </select>
            </div>
            <div className="field">
              <label>Тэмдэглэл</label>
              <input value={form.note} onChange={(e) => setForm({ ...form, note: e.target.value })} />
            </div>
            <div className="modal__actions">
              <button className="btn btn--ghost" onClick={() => setForm(null)}>Болих</button>
              <button className="btn" onClick={save}>Хадгалах</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
