"use client";

// Ажлын мужийн chrome — open-gerege-nexus-ийн дизайныг жишиг болгосон:
// харанхуй топбар (хоёр горимд адил), icon rail + цэсний panel.

import { createContext, useCallback, useContext, useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Boxes, ShieldCheck } from "lucide-react";
import { api, type Me, type MenuApp } from "@/lib/api";
import { Icon } from "./icons";
import { UserMenu } from "./usermenu";

type ShellData = { me: Me; menu: MenuApp[]; refresh: () => void };
const ShellCtx = createContext<ShellData | null>(null);
export const useShell = () => useContext(ShellCtx)!;

export function Shell({ children }: { children: React.ReactNode }) {
  const [data, setData] = useState<ShellData | null>(null);
  const router = useRouter();
  const pathname = usePathname();

  const load = useCallback(async () => {
    try {
      const me = await api.get<Me>("/api/me");
      if (!me.tenant_id) {
        // Байгууллагагүй/сонгоогүй: жагсаалтаас эхнийхийг идэвхжүүлнэ.
        if (me.tenants.length > 0) {
          await api.post("/api/session/tenant", { tenant_id: me.tenants[0].id });
          return load();
        }
        // Платформын админ байгууллагагүйгээр ч админ панель руу орж болно;
        // бусад нь байгууллагаа үүсгэнэ.
        if (!me.user.platform_admin) {
          router.replace("/org/new");
          return;
        }
        if (!window.location.pathname.startsWith("/admin")) {
          router.replace("/admin");
          return;
        }
        setData({ me, menu: [], refresh: () => void load() });
        return;
      }
      const menu = await api.get<{ apps: MenuApp[] }>("/api/menu");
      setData({ me, menu: menu.apps || [], refresh: () => void load() });
    } catch {
      router.replace("/login");
    }
  }, [router]);

  useEffect(() => {
    void load();
  }, [load]);

  if (!data) return null;
  const { me, menu } = data;
  const perms = me.permissions;
  const tenant = me.tenants.find((t) => t.id === me.tenant_id);
  const isOn = (p: string) => pathname === p || pathname.startsWith(p + "/");

  const adminItems = [
    { path: "/members", label: "Гишүүд", icon: "users", perm: "core.members.manage" },
    { path: "/roles", label: "Эрхийн тохиргоо", icon: "key", perm: "core.roles.manage" },
    { path: "/audit", label: "Audit лог", icon: "scroll", perm: "core.audit.read" },
  ].filter((i) => perms[i.perm]);

  return (
    <ShellCtx.Provider value={data}>
      <header className="topbar">
        <div className="topbar__brand">
          <Link href="/dashboard"><span className="brand-square">N</span></Link>
        </div>
        <div className="topbar__context">
          <Boxes size={19} />
          <b>nexus-mini</b>
        </div>
        {tenant && (
          <span className="topbar__tenant">
            <span className="dot" />
            {tenant.name}
          </span>
        )}
        <div className="topbar__spacer" />
        <UserMenu me={me} />
      </header>

      <aside className="rail">
        {me.tenant_id && (
          <>
            <Link href="/dashboard" className={`rail__tile${isOn("/dashboard") ? " is-on" : ""}`}
              title="Дашбоард"><Icon name="dashboard" size={20} /></Link>
            <Link href="/store" className={`rail__tile${isOn("/store") ? " is-on" : ""}`}
              title="Апп дэлгүүр"><Icon name="store" size={20} /></Link>
          </>
        )}
        {menu.map((m) => (
          <Link key={m.app_id} href={m.items[0]?.path || "#"}
            className={`rail__tile${m.items.some((i) => isOn(i.path)) ? " is-on" : ""}`}
            title={m.name}>
            <Icon name={m.items[0]?.icon || "package"} size={20} />
          </Link>
        ))}
        <div style={{ flex: 1 }} />
        {me.user.platform_admin && (
          <Link href="/admin" className={`rail__tile${isOn("/admin") ? " is-on" : ""}`}
            title="Платформын админ"><ShieldCheck size={20} strokeWidth={1.8} /></Link>
        )}
      </aside>

      <nav className="panel">
        {me.tenant_id && (
          <>
            <div className="panel__title">Цэс</div>
            <Link href="/dashboard" className={`nav__item${isOn("/dashboard") ? " is-on" : ""}`}>
              <Icon name="dashboard" size={17} /> Дашбоард
            </Link>
            <Link href="/store" className={`nav__item${isOn("/store") ? " is-on" : ""}`}>
              <Icon name="store" size={17} /> Апп дэлгүүр
            </Link>
          </>
        )}
        {menu.flatMap((m) =>
          m.items.map((i) => (
            <Link key={m.app_id + i.id} href={i.path}
              className={`nav__item${isOn(i.path) ? " is-on" : ""}`}>
              <Icon name={i.icon} size={17} /> {i.label}
            </Link>
          ))
        )}
        {adminItems.length > 0 && (
          <>
            <div className="panel__title" style={{ marginTop: "1rem" }}>Удирдлага</div>
            {adminItems.map((i) => (
              <Link key={i.path} href={i.path}
                className={`nav__item${isOn(i.path) ? " is-on" : ""}`}>
                <Icon name={i.icon} size={17} /> {i.label}
              </Link>
            ))}
          </>
        )}
        {me.user.platform_admin && (
          <>
            <div className="panel__title" style={{ marginTop: "1rem" }}>Платформ</div>
            <Link href="/admin" className={`nav__item${isOn("/admin") ? " is-on" : ""}`}>
              <Icon name="shield" size={17} /> Админ панель
            </Link>
          </>
        )}
      </nav>

      <main className="main">{children}</main>
    </ShellCtx.Provider>
  );
}
