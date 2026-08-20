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
  UserRound,
  Users,
} from "lucide-react";
import { api, type Me } from "@/lib/api";
import { locales, setLocale, useT } from "@/lib/i18n";
import { useThemeMode } from "@/lib/theme";
import { Monitor, Moon, Sun } from "lucide-react";

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
// label нь t()-ийн түлхүүр — render үедээ орчуулагдана.

export function Shell({ children }: { children: React.ReactNode }) {
  const [me, setMe] = useState<Me | null>(null);
  const router = useRouter();
  const pathname = usePathname();
  const { t, locale } = useT();
  const [theme, setTheme] = useThemeMode();

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
          <b>{t("Платформын админ")}</b>
        </div>
        <div className="topbar__spacer" />
        <span className="um__pills" style={{ marginRight: "0.6rem" }}>
          {locales.map((l) => (
            <button key={l.code}
              className={`um__pill${locale === l.code ? " is-on" : ""}`}
              style={{ background: "transparent", borderColor: "#334155", color: locale === l.code ? "#5fd3c2" : "#94a3b8" }}
              onClick={() => setLocale(l.code)}>{l.label}</button>
          ))}
        </span>
        <span className="um__pills" style={{ marginRight: "0.9rem" }}>
          {([["light", Sun], ["dark", Moon], ["system", Monitor]] as const).map(([m, I]) => (
            <button key={m}
              className={`um__pill${theme === m ? " is-on" : ""}`}
              style={{ background: "transparent", borderColor: "#334155", color: theme === m ? "#5fd3c2" : "#94a3b8" }}
              aria-label={m} onClick={() => setTheme(m)}><I size={14} /></button>
          ))}
        </span>
        <span style={{ color: "#94a3b8", marginRight: "0.9rem", fontSize: "0.88rem" }}>
          {me.user.name} · {me.user.email}
        </span>
        <button className="um__btn" style={{ marginRight: "1rem" }} onClick={logout}>
          <LogOut size={15} />
          {t("Гарах")}
        </button>
      </header>

      <nav className="panel">
        {nav.map((n) => (
          <Link key={n.path} href={n.path}
            className={`nav__item${isOn(n.path) ? " is-on" : ""}`}>
            <n.icon size={17} strokeWidth={1.8} /> {t(n.label)}
          </Link>
        ))}
      </nav>

      <main className="main">{children}</main>
    </MeCtx.Provider>
  );
}
