"use client";

import Link from "next/link";
import { useShell } from "@/components/shell";
import { Icon } from "@/components/icons";
import { ArrowRight, CheckCircle2, Circle } from "lucide-react";
import { useT } from "@/lib/i18n";

export default function Dashboard() {
  const { me, menu } = useShell();
  const { t } = useT();
  const tenant = me.tenants.find((t) => t.id === me.tenant_id);
  const hasApps = menu.length > 0;
  const canApps = !!me.permissions["core.apps.manage"];
  const canMembers = !!me.permissions["core.members.manage"];

  const steps = [
    {
      done: hasApps,
      title: t("Апп дэлгүүрээс модуль суулгах"),
      desc: t("Байгууллагад тань хэрэгтэй модулиудыг сонгож суулгана"),
      href: "/store",
      show: canApps,
    },
    {
      done: false,
      title: t("Гишүүдээ урих"),
      desc: t("Ажилтнуудаа нэмээд role оноогоорой"),
      href: "/members",
      show: canMembers,
    },
    {
      done: false,
      title: t("Эрхийн тохиргоо"),
      desc: t("Role бүрийн permission-ийг өөрийн бүтцэд тааруулна"),
      href: "/roles",
      show: !!me.permissions["core.roles.manage"],
    },
  ].filter((s) => s.show);

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{tenant?.name}</h1>
          <div className="sub">{t("Сайн байна уу,")} {me.user.name}</div>
        </div>
      </div>

      {!hasApps && steps.length > 0 && (
        <div className="card card__pad" style={{ marginBottom: "1rem" }}>
          <b style={{ display: "block", marginBottom: "0.8rem" }}>{t("Эхлэхэд туслах")}</b>
          {steps.map((s) => (
            <Link key={s.title} href={s.href}
              style={{ display: "flex", gap: "0.7rem", alignItems: "center", padding: "0.55rem 0" }}>
              {s.done ? (
                <CheckCircle2 size={19} style={{ color: "var(--ok)" }} />
              ) : (
                <Circle size={19} style={{ color: "var(--text-3)" }} />
              )}
              <span style={{ flex: 1 }}>
                <b style={{ fontWeight: 600 }}>{s.title}</b>
                <span style={{ display: "block", color: "var(--text-2)", fontSize: "0.85rem" }}>
                  {s.desc}
                </span>
              </span>
              <ArrowRight size={16} style={{ color: "var(--text-3)" }} />
            </Link>
          ))}
        </div>
      )}

      <div className="stat-grid">
        <div className="card stat">
          <span className="stat__icon"><Icon name="store" /></span>
          <span>
            <b>{menu.length}</b>
            <span>{t("Идэвхтэй апп")}</span>
          </span>
        </div>
        <div className="card stat">
          <span className="stat__icon"><Icon name="key" /></span>
          <span>
            <b>{Object.keys(me.permissions).length}</b>
            <span>{t("Таны эрх")}</span>
          </span>
        </div>
        <div className="card stat">
          <span className="stat__icon"><Icon name="building" /></span>
          <span>
            <b>{me.tenants.length}</b>
            <span>{t("Байгууллага")}</span>
          </span>
        </div>
      </div>

      {hasApps && (
        <div className="card" style={{ marginTop: "1rem" }}>
          <div className="card__pad" style={{ borderBottom: "1px solid var(--border)" }}>
            <b>{t("Суусан аппууд")}</b>
          </div>
          {menu.map((m) => (
            <Link key={m.app_id} href={m.items[0]?.path || "#"}
              style={{ display: "flex", gap: "0.8rem", alignItems: "center", padding: "0.8rem 1.25rem", borderBottom: "1px solid var(--border)" }}>
              <span className="stat__icon" style={{ width: "2.2rem", height: "2.2rem" }}>
                <Icon name={m.items[0]?.icon || "package"} size={17} />
              </span>
              <span style={{ flex: 1 }}>{m.name}</span>
              <ArrowRight size={16} style={{ color: "var(--text-3)" }} />
            </Link>
          ))}
        </div>
      )}
    </>
  );
}
