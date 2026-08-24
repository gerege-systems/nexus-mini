"use client";

import { useCallback, useEffect, useState } from "react";
import { Plus, Trash2, Users } from "lucide-react";
import { api, ApiError, type Member, type Role } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

export default function MembersPage() {
  const { me } = useShell();
  const { t } = useT();
  const [members, setMembers] = useState<Member[] | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState({ email: "", name: "", password: "", roles: ["user"] });
  const [err, setErr] = useState("");
  // Имэйлээр хайлт: null = хараахан хайгаагүй/хүчингүй имэйл
  const [lookup, setLookup] = useState<{ exists: boolean; name?: string; member?: boolean } | null>(null);
  const emailOk = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim());
  useEffect(() => {
    if (!adding || !emailOk) { setLookup(null); return; }
    const id = setTimeout(() => {
      api.get<{ exists: boolean; name?: string; member?: boolean }>(
        `/api/members/lookup?email=${encodeURIComponent(form.email.trim())}`
      ).then(setLookup).catch(() => setLookup(null));
    }, 350);
    return () => clearTimeout(id);
  }, [form.email, adding, emailOk]);

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
      toast(t("Гишүүн нэмэгдлээ"));
      await load();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    }
  };

  const setMemberRoles = async (m: Member, code: string, on: boolean) => {
    const next = on ? [...m.roles, code] : m.roles.filter((r) => r !== code);
    try {
      await api.put(`/api/members/${m.membership_id}/roles`, { roles: next });
      toast(t("Role шинэчлэгдлээ"));
    } catch (e) {
      toast(e instanceof ApiError ? t(e.message) : t("Алдаа гарлаа"), "err");
    }
    await load();
  };

  const remove = async (m: Member) => {
    if (!confirm(`${m.name} ${t("хасах уу?")}`)) return;
    try {
      await api.del(`/api/members/${m.membership_id}`);
      toast(t("Гишүүн хасагдлаа"));
    } catch (e) {
      toast(e instanceof ApiError ? t(e.message) : t("Алдаа гарлаа"), "err");
    }
    await load();
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Гишүүд")}</h1>
          <div className="sub">{t("Байгууллагын гишүүд ба role оноолт")}</div>
        </div>
        <div className="spacer" />
        <button className="btn" onClick={() => { setErr(""); setAdding(true); }}>
          <Plus size={16} /> {t("Гишүүн нэмэх")}
        </button>
      </div>

      <div className="card">
        {members && members.length === 0 ? (
          <div className="empty">
            <Users size={36} strokeWidth={1.4} />
            <b>{t("Гишүүн алга")}</b>
          </div>
        ) : (
          <table className="table">
            <thead>
              <tr><th>{t("Нэр")}</th><th>{t("Имэйл")}</th><th>Role</th><th></th></tr>
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
                            title={self ? t("Өөрийн role-г эндээс өөрчлөхгүй") : ""}
                            onClick={() => setMemberRoles(m, code, !on)}>
                            {code}
                          </button>
                        );
                      })}
                    </span>
                  </td>
                  <td style={{ textAlign: "right" }}>
                    {m.user_id !== me.user.id && (
                      <button className="btn btn--ghost btn--sm" aria-label={`${m.name} ${t("хасах уу?")}`}
                        onClick={() => remove(m)}>
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
        <div className="modal-back" onClick={() => setAdding(false)} onKeyDown={(e) => e.key === "Escape" && setAdding(false)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
            <h3>{t("Гишүүн нэмэх")}</h3>
            {err && <div className="alert alert--danger">{t(err)}</div>}
            <div className="field">
              <label>{t("Имэйл")}</label>
              <input type="email" value={form.email} autoFocus
                onChange={(e) => setForm({ ...form, email: e.target.value })} />
              <div className="hint">
                {!emailOk
                  ? t("Имэйлээр хайна: бүртгэлтэй бол нэр нь гарна, үгүй бол шинээр үүсгэнэ")
                  : lookup === null
                    ? t("Хайж байна…")
                    : lookup.exists
                      ? lookup.member ? t("Аль хэдийн энэ байгууллагын гишүүн") : t("Бүртгэлтэй хэрэглэгч — role өгөөд нэмнэ")
                      : t("Бүртгэлгүй — нэр, түр нууц үг өгч шинээр үүсгэнэ")}
              </div>
            </div>
            {lookup?.exists && (
              <div className="field">
                <label>{t("Нэр")}</label>
                <input value={lookup.name ?? ""} readOnly disabled />
              </div>
            )}
            {lookup && !lookup.exists && (
              <>
                <div className="field">
                  <label>{t("Нэр")}</label>
                  <input value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
                </div>
                <div className="field">
                  <label>{t("Түр нууц үг")}</label>
                  <input type="password" value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })} />
                  <div className="hint">{t("8+ тэмдэгт: латин үсэг, тоо, тусгай тэмдэгт (кирилл хориотой)")}</div>
                </div>
              </>
            )}
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
              <button className="btn btn--ghost" onClick={() => setAdding(false)}>{t("Болих")}</button>
              <button className="btn" onClick={add} disabled={!lookup || lookup.member}>{t("Нэмэх")}</button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
