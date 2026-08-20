"use client";

import { useCallback, useEffect, useState } from "react";
import { Plus, Trash2, Users } from "lucide-react";
import { api, ApiError, type Member, type Role } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";

export default function MembersPage() {
  const { me } = useShell();
  const [members, setMembers] = useState<Member[] | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState({ email: "", name: "", password: "", roles: ["user"] });
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    const [m, r] = await Promise.all([
      api.get<{ members: Member[] }>("/api/members"),
      me.permissions["core.roles.manage"]
        ? api.get<{ roles: Role[] }>("/api/roles")
        : Promise.resolve({ roles: [] as Role[] }),
    ]);
    setMembers(m.members);
    setRoles(r.roles);
  }, [me.permissions]);
  useEffect(() => { void load(); }, [load]);

  const roleCodes = roles.length > 0 ? roles.map((r) => r.code) : ["admin", "manager", "user"];

  const add = async () => {
    setErr("");
    try {
      await api.post("/api/members", form);
      setAdding(false);
      setForm({ email: "", name: "", password: "", roles: ["user"] });
      toast("Гишүүн нэмэгдлээ");
      await load();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : "Алдаа гарлаа");
    }
  };

  const setMemberRoles = async (m: Member, code: string, on: boolean) => {
    const next = on ? [...m.roles, code] : m.roles.filter((r) => r !== code);
    try {
      await api.put(`/api/members/${m.membership_id}/roles`, { roles: next });
      toast("Role шинэчлэгдлээ");
    } catch (e) {
      toast(e instanceof ApiError ? e.message : "Алдаа гарлаа", "err");
    }
    await load();
  };

  const remove = async (m: Member) => {
    if (!confirm(`${m.name}-г байгууллагаас хасах уу?`)) return;
    try {
      await api.del(`/api/members/${m.membership_id}`);
      toast("Гишүүн хасагдлаа");
    } catch (e) {
      toast(e instanceof ApiError ? e.message : "Алдаа гарлаа", "err");
    }
    await load();
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Гишүүд</h1>
          <div className="sub">Байгууллагын гишүүд ба role оноолт</div>
        </div>
        <div className="spacer" />
        <button className="btn" onClick={() => { setErr(""); setAdding(true); }}>
          <Plus size={16} /> Гишүүн нэмэх
        </button>
      </div>

      <div className="card">
        {members && members.length === 0 ? (
          <div className="empty">
            <Users size={36} strokeWidth={1.4} />
            <b>Гишүүн алга</b>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr><th>Нэр</th><th>Имэйл</th><th>Role</th><th></th></tr>
            </thead>
            <tbody>
              {members?.map((m) => (
                <tr key={m.membership_id}>
                  <td><b style={{ fontWeight: 600 }}>{m.name}</b></td>
                  <td style={{ color: "var(--text-2)" }}>{m.email}</td>
                  <td>
                    <span style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
                      {roleCodes.map((code) => {
                        const on = m.roles.includes(code);
                        const self = m.user_id === me.user.id;
                        return (
                          <button key={code}
                            className={`um__pill${on ? " is-on" : ""}`}
                            disabled={self}
                            title={self ? "Өөрийн role-г эндээс өөрчлөхгүй" : ""}
                            onClick={() => setMemberRoles(m, code, !on)}>
                            {code}
                          </button>
                        );
                      })}
                    </span>
                  </td>
                  <td style={{ textAlign: "right" }}>
                    {m.user_id !== me.user.id && (
                      <button className="btn btn--ghost btn--sm" onClick={() => remove(m)}>
                        <Trash2 size={13} />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {adding && (
        <div className="modal-back" onClick={() => setAdding(false)}>
          <div className="modal" onClick={(e) => e.stopPropagation()}>
            <h3>Гишүүн нэмэх</h3>
            {err && <div className="alert alert--danger">{err}</div>}
            <div className="field">
              <label>Имэйл</label>
              <input value={form.email} autoFocus
                onChange={(e) => setForm({ ...form, email: e.target.value })} />
              <div className="hint">Бүртгэлтэй имэйл бол шууд нэгдэнэ, нэр/нууц үг хэрэггүй</div>
            </div>
            <div className="field">
              <label>Нэр (шинэ хэрэглэгчид)</label>
              <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="field">
              <label>Түр нууц үг (шинэ хэрэглэгчид, 8+)</label>
              <input type="password" value={form.password}
                onChange={(e) => setForm({ ...form, password: e.target.value })} />
            </div>
            <div className="field">
              <label>Role</label>
              <span style={{ display: "flex", gap: "0.3rem", flexWrap: "wrap" }}>
                {roleCodes.map((code) => {
                  const on = form.roles.includes(code);
                  return (
                    <button key={code} className={`um__pill${on ? " is-on" : ""}`}
                      onClick={() =>
                        setForm({
                          ...form,
                          roles: on ? form.roles.filter((r) => r !== code) : [...form.roles, code],
                        })
                      }>
                      {code}
                    </button>
                  );
                })}
              </span>
            </div>
            <div className="modal__actions">
              <button className="btn btn--ghost" onClick={() => setAdding(false)}>Болих</button>
              <button className="btn" onClick={add}>Нэмэх</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
