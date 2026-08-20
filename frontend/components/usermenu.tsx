"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  ChevronDown,
  LogOut,
  Sun,
  Moon,
  Monitor,
  Building2,
  Check,
  Plus,
} from "lucide-react";
import { api, type Me } from "@/lib/api";
import { useThemeMode } from "@/lib/theme";

export function UserMenu({ me, onTenantChange }: { me: Me; onTenantChange?: () => void }) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const router = useRouter();
  const [theme, setTheme] = useThemeMode();

  useEffect(() => {
    const onDoc = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, []);

  const initials = me.user.name
    .split(" ")
    .map((s) => s[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();

  const selectTenant = async (id: string) => {
    await api.post("/api/session/tenant", { tenant_id: id });
    setOpen(false);
    if (onTenantChange) onTenantChange();
    else window.location.reload();
  };

  const logout = async () => {
    await api.post("/api/logout");
    router.replace("/login");
  };

  return (
    <div className="topbar__user" ref={ref}>
      <button className={`um__btn${open ? " is-open" : ""}`} onClick={() => setOpen(!open)}>
        <span className="um__avatar">{initials}</span>
        <span>{me.user.name}</span>
        <ChevronDown size={15} className="um__chev" />
      </button>
      {open && (
        <div className="um__menu">
          <div className="um__head">
            <b>{me.user.name}</b>
            <span>{me.user.email}</span>
          </div>

          <div className="um__sect">Байгууллага</div>
          {me.tenants.map((t) => (
            <button key={t.id} className={`um__item${t.id === me.tenant_id ? " is-on" : ""}`}
              onClick={() => selectTenant(t.id)}>
              <Building2 size={16} />
              <span style={{ flex: 1 }}>{t.name}</span>
              {t.id === me.tenant_id && <Check size={15} />}
            </button>
          ))}
          <button className="um__item" onClick={() => { setOpen(false); router.push("/org/new"); }}>
            <Plus size={16} />
            Байгууллага нэмэх
          </button>

          <div className="um__sect">Тохиргоо</div>
          <div className="um__prefrow">
            <span>Загвар</span>
            <div className="um__pills">
              <button className={`um__pill${theme === "light" ? " is-on" : ""}`}
                onClick={() => setTheme("light")} title="Цайвар"><Sun size={14} /></button>
              <button className={`um__pill${theme === "dark" ? " is-on" : ""}`}
                onClick={() => setTheme("dark")} title="Бараан"><Moon size={14} /></button>
              <button className={`um__pill${theme === "system" ? " is-on" : ""}`}
                onClick={() => setTheme("system")} title="Систем"><Monitor size={14} /></button>
            </div>
          </div>

          <button className="um__item um__logout" onClick={logout}>
            <LogOut size={16} />
            Гарах
          </button>
        </div>
      )}
    </div>
  );
}
