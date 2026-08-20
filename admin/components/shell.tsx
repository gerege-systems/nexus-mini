"use client";

// Админ аппын chrome — порталтай ижил загвар: харанхуй топбар,
// icon rail + цэсний panel. Teal accent.

import { createContext, useContext, useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  Building2,
  LayoutGrid,
  LogOut,
  Package,
  ScrollText,
  ShieldCheck,
  UserRound,
  Users,
} from "lucide-react";
import { api, type Me } from "@/lib/api";

const MeCtx = createContext<Me | null>(null);
export const useMe = () => useContext(MeCtx)!;

const nav = [
  { path: "/", label: "Тойм", icon: LayoutGrid },
  { path: "/tenants", label: "Байгууллагууд", icon: Building2 },
  { path: "/users", label: "Хэрэглэгчид", icon: Users },
  { path: "/apps", label: "Каталог", icon: Package },
  { path: "/audit", label: "Audit", icon: ScrollText },
  { path: "/profile", label: "Профайл", icon: UserRound },
];

export function Shell({ children }: { children: React.ReactNode }) {
  const [me, setMe] = useState<Me | null>(null);
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    api
      .get<Me>("/api/me")
      .then((m) => {
        if (!m.user.platform_admin) throw new Error("not admin");
        setMe(m);
      })
      .catch(() => router.replace("/login"));
  }, [router]);

  if (!me) return null;
  const isOn = (p: string) => (p === "/" ? pathname === "/" : pathname.startsWith(p));

  const logout = async () => {
    await api.post("/api/logout");
    router.replace("/login");
  };

  return (
    <MeCtx.Provider value={me}>
      <header className="topbar">
        <div className="topbar__brand">
          <Link href="/"><span className="brand-square">N</span></Link>
        </div>
        <div className="topbar__context">
          <ShieldCheck size={19} />
          <b>Платформын админ</b>
        </div>
        <div className="topbar__spacer" />
        <span style={{ color: "#94a3b8", marginRight: "0.9rem", fontSize: "0.88rem" }}>
          {me.user.name} · {me.user.email}
        </span>
        <button className="um__btn" style={{ marginRight: "1rem" }} onClick={logout}>
          <LogOut size={15} />
          Гарах
        </button>
      </header>

      <nav className="panel">
        {nav.map((n) => (
          <Link key={n.path} href={n.path}
            className={`nav__item${isOn(n.path) ? " is-on" : ""}`}>
            <n.icon size={17} strokeWidth={1.8} /> {n.label}
          </Link>
        ))}
      </nav>

      <main className="main">{children}</main>
    </MeCtx.Provider>
  );
}
