"use client";

import { Fragment, useCallback, useEffect, useMemo, useState } from "react";
import { KeyRound, Plus } from "lucide-react";
import { api, ApiError, type Permission, type Role } from "@/lib/api";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

// Role × permission матриц: нүд бүр — / all / own гэсэн 3 төлөвт шилжинэ.
export default function RolesPage() {
  const { t } = useT();
  const [roles, setRoles] = useState<Role[]>([]);
  const [perms, setPerms] = useState<Permission[]>([]);
  const [err, setErr] = useState("");
  const [creating, setCreating] = useState(false);
  const [form, setForm] = useState({ code: "", name: "", implies: "" });

  const load = useCallback(async () => {
    const [r, p] = await Promise.all([
      api.get<{ roles: Role[] }>("/api/roles"),
      api.get<{ permissions: Permission[] }>("/api/permissions"),
    ]);
    setRoles(r.roles);
    setPerms(p.permissions);
  }, []);
  useEffect(() => { void load(); }, [load]);

  const groups = useMemo(() => {
    const g = new Map<string, Permission[]>();
    for (const p of perms) {
      // Модулийн ID урт тул permission кодын prefix-ээр нэрлэнэ.
      const key = p.module_id === "core" ? "core" : p.code.split(".")[0];
      if (!g.has(key)) g.set(key, []);
      g.get(key)!.push(p);
    }
    return [...g.entries()];
  }, [perms]);

  const cycle = async (role: Role, p: Permission) => {
    if (role.code === "admin") return; // admin үргэлж бүгдийг эзэмшинэ
    const cur = role.grants[p.code];
    let next: "all" | "own" | undefined;
    if (!cur) next = "all";
    else if (cur === "all" && p.own_scope) next = "own";
    else next = undefined;
    const grants = { ...role.grants };
    if (next) grants[p.code] = next;
    else delete grants[p.code];
    try {
      await api.put(`/api/roles/${role.id}/grants`, { grants });
      toast(t("Оноолт хадгалагдлаа"));
      await load();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    }
  };

  const create = async () => {
    setErr("");
    try {
      await api.post("/api/roles", {
        code: form.code,
        name: form.name,
        implies: form.implies,
      });
      setCreating(false);
      setForm({ code: "", name: "", implies: "" });
      toast(t("Role үүслээ"));
      await load();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    }
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Эрхийн тохиргоо")}</h1>
          <div className="sub">
            {t("Нүд дарж — → бүгд → өөрийн гэж эргэлдэнэ. Role нь implies-ээрээ доод role-ийн эрхийг өвлөнө.")}
          </div>
        </div>
        <div className="spacer" />
        <button className="btn" onClick={() => { setErr(""); setCreating(true); }}>
          <Plus size={16} /> {t("Role нэмэх")}
        </button>
      </div>
      {err && <div className="alert alert--danger">{t(err)}</div>}

      <div className="card" style={{ overflowX: "auto" }}>
        <table className="table" style={{ minWidth: 560 }}>
          <thead>
            <tr>
              <th>Permission</th>
              {roles.map((r) => (
                <th key={r.id} style={{ textAlign: "center", width: 170, minWidth: 150, borderLeft: "1px solid var(--border)" }}>
                  {r.name}
                  <div style={{ textTransform: "none", letterSpacing: 0, fontWeight: 400 }}>
                    <code style={{ fontSize: "0.75rem" }}>{r.code}</code>
                    {r.implies && (
                      <span style={{ color: "var(--text-3)" }}> ⊃ {r.implies}</span>
                    )}
                  </div>
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {groups.map(([group, list]) => (
              <Fragment key={group}>
                <tr>
                  <td colSpan={roles.length + 1}
                    style={{ background: "var(--bg)", fontWeight: 700, fontSize: "0.8rem", textTransform: "uppercase", letterSpacing: "0.04em", color: "var(--text-2)" }}>
                    {group === "core" ? t("Платформ") : group}
                  </td>
                </tr>
                {list.map((p) => (
                  <tr key={p.code}>
                    <td>
                      <b style={{ fontWeight: 600 }}>{p.name}</b>
                      <div style={{ color: "var(--text-3)", fontSize: "0.78rem" }}>
                        <code>{p.code}</code>
                      </div>
                    </td>
                    {roles.map((r) => {
                      const v = r.code === "admin" ? "all" : r.grants[p.code];
                      return (
                        <td key={r.id} style={{ textAlign: "center", borderLeft: "1px solid var(--border)" }}>
                          <button
                            className={`um__pill${v ? " is-on" : ""}`}
                            style={{ minWidth: "4.2rem", justifyContent: "center", opacity: r.code === "admin" ? 0.6 : 1 }}
                            disabled={r.code === "admin"}
                            title={r.code === "admin" ? t("Админ үргэлж бүх эрхтэй") : ""}
                            onClick={() => cycle(r, p)}>
                            {v === "all" ? t("Бүгд") : v === "own" ? t("Өөрийн") : "—"}
                          </button>
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </Fragment>
            ))}
          </tbody>
        </table>
      </div>

      {creating && (
        <div className="modal-back" onClick={() => setCreating(false)} onKeyDown={(e) => e.key === "Escape" && setCreating(false)}>
          <div className="modal" role="dialog" aria-modal="true" onClick={(e) => e.stopPropagation()}>
            <h3>{t("Role нэмэх")}</h3>
            <div className="field">
              <label>{t("Код")}</label>
              <input value={form.code} autoFocus placeholder="warehouse_staff"
                onChange={(e) => setForm({ ...form, code: e.target.value })} />
              <div className="hint">{t("Жижиг үсэг, тоо, _")}</div>
            </div>
            <div className="field">
              <label>{t("Нэр")}</label>
              <input value={form.name} placeholder={t("Агуулахын ажилтан")}
                onChange={(e) => setForm({ ...form, name: e.target.value })} />
            </div>
            <div className="field">
              <label>{t("Өвлөх role (сонголттой)")}</label>
              <select value={form.implies}
                onChange={(e) => setForm({ ...form, implies: e.target.value })}>
                <option value="">{t("— өвлөхгүй —")}</option>
                {roles.map((r) => (
                  <option key={r.id} value={r.code}>{r.name} ({r.code})</option>
                ))}
              </select>
            </div>
            <div className="modal__actions">
              <button className="btn btn--ghost" onClick={() => setCreating(false)}>{t("Болих")}</button>
              <button className="btn" onClick={create}>{t("Үүсгэх")}</button>
            </div>
          </div>
        </div>
      )}
      <p style={{ color: "var(--text-3)", fontSize: "0.82rem", marginTop: "0.8rem", display: "flex", gap: "0.4rem", alignItems: "center" }}>
        <KeyRound size={14} /> {t("«Өөрийн» = зөвхөн өөрийн үүсгэсэн бүртгэл дээр үйлдэл хийнэ (модуль нь дэмждэг бол)")}
      </p>
    </>
  );
}
