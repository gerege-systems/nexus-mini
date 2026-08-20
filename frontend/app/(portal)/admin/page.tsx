"use client";

import { useEffect, useState } from "react";
import { Building2, Package, ScrollText, Users } from "lucide-react";
import { api } from "@/lib/api";
import { useShell } from "@/components/shell";

type Overview = { tenants: number; users: number; apps: number; installations: number };
type TenantRow = { id: string; slug: string; name: string; created_at: string; members: number; apps: number };
type UserRow = { id: string; email: string; name: string; platform_admin: boolean; created_at: string; tenants: number };
type AppRow = { id: string; short_id: string; name: string; version: string; compiled: boolean; publisher: string; installs: number };
type AuditRow = { id: number; tenant: string; user_name: string; action: string; object: string; occurred_at: string };

const tabs = [
  { id: "tenants", label: "Байгууллагууд" },
  { id: "users", label: "Хэрэглэгчид" },
  { id: "apps", label: "Каталог" },
  { id: "audit", label: "Audit" },
] as const;

export default function AdminPage() {
  const { me } = useShell();
  const [ov, setOv] = useState<Overview | null>(null);
  const [tab, setTab] = useState<(typeof tabs)[number]["id"]>("tenants");
  const [tenants, setTenants] = useState<TenantRow[]>([]);
  const [users, setUsers] = useState<UserRow[]>([]);
  const [apps, setApps] = useState<AppRow[]>([]);
  const [audit, setAudit] = useState<AuditRow[]>([]);

  useEffect(() => {
    if (!me.user.platform_admin) return;
    void api.get<Overview>("/api/admin/overview").then(setOv);
  }, [me.user.platform_admin]);

  useEffect(() => {
    if (!me.user.platform_admin) return;
    if (tab === "tenants") void api.get<{ tenants: TenantRow[] }>("/api/admin/tenants").then((r) => setTenants(r.tenants));
    if (tab === "users") void api.get<{ users: UserRow[] }>("/api/admin/users").then((r) => setUsers(r.users));
    if (tab === "apps") void api.get<{ apps: AppRow[] }>("/api/admin/apps").then((r) => setApps(r.apps));
    if (tab === "audit") void api.get<{ entries: AuditRow[] }>("/api/admin/audit").then((r) => setAudit(r.entries));
  }, [tab, me.user.platform_admin]);

  if (!me.user.platform_admin) {
    return <div className="alert alert--danger">Энэ хуудас платформын админд л нээлттэй.</div>;
  }

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Платформын админ</h1>
          <div className="sub">Бүх байгууллага, хэрэглэгч, каталогийн тойм</div>
        </div>
      </div>

      {ov && (
        <div className="stat-grid" style={{ marginBottom: "1.25rem" }}>
          <div className="card stat">
            <span className="stat__icon"><Building2 size={19} /></span>
            <span><b>{ov.tenants}</b><span>Байгууллага</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><Users size={19} /></span>
            <span><b>{ov.users}</b><span>Хэрэглэгч</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><Package size={19} /></span>
            <span><b>{ov.apps}</b><span>Бэлэн апп</span></span>
          </div>
          <div className="card stat">
            <span className="stat__icon"><ScrollText size={19} /></span>
            <span><b>{ov.installations}</b><span>Суулгалт</span></span>
          </div>
        </div>
      )}

      <div style={{ display: "flex", gap: "0.4rem", marginBottom: "0.9rem" }}>
        {tabs.map((t) => (
          <button key={t.id} className={`um__pill${tab === t.id ? " is-on" : ""}`}
            style={{ padding: "0.35rem 0.9rem" }} onClick={() => setTab(t.id)}>
            {t.label}
          </button>
        ))}
      </div>

      <div className="card">
        {tab === "tenants" && (
          <table className="table">
            <thead><tr><th>Нэр</th><th>Slug</th><th>Гишүүд</th><th>Аппууд</th><th>Үүссэн</th></tr></thead>
            <tbody>
              {tenants.map((t) => (
                <tr key={t.id}>
                  <td><b style={{ fontWeight: 600 }}>{t.name}</b></td>
                  <td><code>{t.slug}</code></td>
                  <td>{t.members}</td>
                  <td>{t.apps}</td>
                  <td style={{ color: "var(--text-2)" }}>{new Date(t.created_at).toLocaleDateString("mn-MN")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {tab === "users" && (
          <table className="table">
            <thead><tr><th>Нэр</th><th>Имэйл</th><th>Байгууллага</th><th></th></tr></thead>
            <tbody>
              {users.map((u) => (
                <tr key={u.id}>
                  <td><b style={{ fontWeight: 600 }}>{u.name}</b></td>
                  <td style={{ color: "var(--text-2)" }}>{u.email}</td>
                  <td>{u.tenants}</td>
                  <td>{u.platform_admin && <span className="badge badge--accent">платформ админ</span>}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {tab === "apps" && (
          <table className="table">
            <thead><tr><th>Апп</th><th>ID</th><th>Хувилбар</th><th>Бинарид</th><th>Суулгалт</th></tr></thead>
            <tbody>
              {apps.map((a) => (
                <tr key={a.id}>
                  <td><b style={{ fontWeight: 600 }}>{a.name}</b>
                    <div style={{ color: "var(--text-3)", fontSize: "0.8rem" }}>{a.publisher}</div></td>
                  <td><code style={{ fontSize: "0.8rem" }}>{a.id}</code></td>
                  <td>v{a.version}</td>
                  <td>{a.compiled
                    ? <span className="badge badge--ok">Тийм</span>
                    : <span className="badge badge--muted">Үгүй</span>}</td>
                  <td>{a.installs}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {tab === "audit" && (
          <table className="table">
            <thead><tr><th>#</th><th>Байгууллага</th><th>Үйлдэл</th><th>Объект</th><th>Хэн</th><th>Хэзээ</th></tr></thead>
            <tbody>
              {audit.map((e) => (
                <tr key={e.id}>
                  <td style={{ color: "var(--text-3)" }}>{e.id}</td>
                  <td><code>{e.tenant}</code></td>
                  <td><span className="badge badge--accent">{e.action}</span></td>
                  <td>{e.object || "—"}</td>
                  <td>{e.user_name || "систем"}</td>
                  <td style={{ color: "var(--text-2)", whiteSpace: "nowrap" }}>
                    {new Date(e.occurred_at).toLocaleString("mn-MN")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </>
  );
}
