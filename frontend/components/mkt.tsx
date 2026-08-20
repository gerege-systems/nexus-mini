"use client";

// Landing-ийн хуудсуудын (нүүр / апп дэлгүүр / хөгжүүлэгч) нийтлэг chrome.

import Link from "next/link";
import { usePathname } from "next/navigation";

export function MktHeader() {
  const pathname = usePathname();
  const on = (p: string) =>
    (p === "/" ? pathname === "/" : pathname.startsWith(p))
      ? { color: "var(--text)", fontWeight: 600 as const }
      : undefined;
  return (
    <header className="mkt-top">
      <Link href="/" style={{ display: "flex", alignItems: "center", gap: "0.7rem" }}>
        <span className="brand-square">N</span>
        <b>nexus-mini</b>
      </Link>
      <nav>
        <Link href="/" style={on("/")}>Нүүр</Link>
        <Link href="/apps" style={on("/apps")}>Апп дэлгүүр</Link>
        <Link href="/developers" style={on("/developers")}>Модуль хөгжүүлэх</Link>
      </nav>
      <span className="spacer" />
      <a href="https://github.com/gerege-systems/nexus-mini" className="btn btn--ghost btn--sm">
        GitHub
      </a>
      <Link href="/login" className="btn btn--ghost btn--sm">Нэвтрэх</Link>
      <Link href="/signup" className="btn btn--sm">Бүртгүүлэх</Link>
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
