"use client";

// Ажлын мужийн chrome — open-gerege-nexus-ийн дизайныг жишиг болгосон:
// харанхуй топбар (хоёр горимд адил), icon rail + цэсний panel.

import { createContext, useCallback, useContext, useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Boxes } from "lucide-react";
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
        router.replace("/org/new");
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

  // Gerege загвар: rail нь идэвхтэй АПП сонгогч. Одоогийн зам аль нэг
  // модулийн цэст харьяалагдвал тэр модуль, үгүй бол Платформ идэвхтэй —
  // panel зөвхөн идэвхтэй аппын цэсийг харуулна.
  const activeModule = menu.find((m) => m.items.some((i) => isOn(i.path)));

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
        <Link href="/dashboard"
          className={`rail__tile${!activeModule ? " is-on" : ""}`}
          title="Платформ" aria-label="Платформ">
          <Icon name="home" size={20} />
        </Link>
        {menu.map((m) => (
          <Link key={m.app_id} href={m.items[0]?.path || "#"}
            className={`rail__tile${activeModule?.app_id === m.app_id ? " is-on" : ""}`}
            title={m.name} aria-label={m.name}>
            <Icon name={m.items[0]?.icon || "package"} size={20} />
          </Link>
        ))}
      </aside>

      <nav className="panel">
        {activeModule ? (
          <>
            <div className="panel__title">{activeModule.name}</div>
            {activeModule.items.map((i) => (
              <Link key={i.id} href={i.path}
                className={`nav__item${isOn(i.path) ? " is-on" : ""}`}>
                <Icon name={i.icon} size={17} /> {i.label}
              </Link>
            ))}
          </>
        ) : (
          <>
            <div className="panel__title">Цэс</div>
            <Link href="/dashboard" className={`nav__item${isOn("/dashboard") ? " is-on" : ""}`}>
              <Icon name="dashboard" size={17} /> Дашбоард
            </Link>
            <Link href="/store" className={`nav__item${isOn("/store") ? " is-on" : ""}`}>
              <Icon name="store" size={17} /> Апп дэлгүүр
            </Link>
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
          </>
        )}
      </nav>

      <main className="main">{children}</main>
    </ShellCtx.Provider>
  );
}
