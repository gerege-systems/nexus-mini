"use client";

// Landing-ийн хуудсуудын (нүүр / апп дэлгүүр / хөгжүүлэгч) нийтлэг chrome.
// Баруун тал: хэл (dropdown) · GitHub · загвар (нэг icon toggle) — товч
// icon хэлбэрээр (shadcn-маягийн жишгээр).

import { useEffect, useRef, useState } from "react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ChevronDown, Languages, Moon, Sun } from "lucide-react";
import { locales, setLocale, useT } from "@/lib/i18n";
import { useThemeMode } from "@/lib/theme";

function GitHubIcon({ size = 17 }: { size?: number }) {
  return (
    <svg width={size} height={size} viewBox="0 0 16 16" fill="currentColor" aria-hidden>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
    </svg>
  );
}

function LangMenu() {
  const { locale } = useT();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);
  return (
    <div ref={ref} style={{ position: "relative" }}>
      <button className="mkt-iconbtn" style={{ width: "auto", padding: "0 0.55rem", gap: "0.3rem" }}
        aria-label="Language" onClick={() => setOpen(!open)}>
        <Languages size={15} />
        <span style={{ fontSize: "0.8rem", fontWeight: 600 }}>{locale.toUpperCase()}</span>
        <ChevronDown size={13} />
      </button>
      {open && (
        <div className="mkt-menu">
          {locales.map((l) => (
            <button key={l.code} className={`um__item${l.code === locale ? " is-on" : ""}`}
              onClick={() => setLocale(l.code)}>
              {l.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

function ThemeToggle() {
  const [theme, setTheme] = useThemeMode();
  const toggle = () => {
    const dark =
      theme === "dark" ||
      (theme === "system" && matchMedia("(prefers-color-scheme: dark)").matches);
    setTheme(dark ? "light" : "dark");
  };
  const showSun =
    theme === "dark" ||
    (theme === "system" &&
      typeof window !== "undefined" &&
      matchMedia("(prefers-color-scheme: dark)").matches);
  return (
    <button className="mkt-iconbtn" aria-label="Theme" onClick={toggle}>
      {showSun ? <Sun size={17} /> : <Moon size={17} />}
    </button>
  );
}

export function MktHeader() {
  const pathname = usePathname();
  const { t } = useT();
  const on = (p: string) =>
    (p === "/" ? pathname === "/" : pathname.startsWith(p))
      ? { color: "var(--text)", fontWeight: 600 as const }
      : undefined;
  return (
    <header className="mkt-top">
      <div className="mkt-top__in">
      <Link href="/" style={{ display: "flex", alignItems: "center", gap: "0.7rem" }}>
        <span className="brand-square">N</span>
        <b>nexus-mini</b>
      </Link>
      <nav>
        <Link href="/" style={on("/")}>{t("Нүүр")}</Link>
        <Link href="/apps" style={on("/apps")}>{t("Апп дэлгүүр")}</Link>
        <Link href="/developers" style={on("/developers")}>{t("Модуль хөгжүүлэх")}</Link>
      </nav>
      <span className="spacer" />
      <LangMenu />
      <a href="https://github.com/gerege-systems/nexus-mini" className="mkt-iconbtn" aria-label="GitHub">
        <GitHubIcon />
      </a>
      <ThemeToggle />
      <span style={{ width: 1, height: "1.4rem", background: "var(--border)", margin: "0 0.4rem" }} />
      <Link href="/login" className="btn btn--sm">{t("Нэвтрэх")}</Link>
      </div>
    </header>
  );
}

export function MktFooter() {
  return (
    <footer className="mkt-foot">
      <span className="brand-square" style={{ width: "1.6rem", height: "1.6rem", fontSize: "0.8rem" }}>N</span>
      <span>
        nexus-mini · Apache 2.0 ·{" "}
        <a href="https://github.com/gerege-systems/nexus-mini" style={{ color: "var(--accent)" }}>
          gerege-systems/nexus-mini
        </a>
      </span>
    </footer>
  );
}
