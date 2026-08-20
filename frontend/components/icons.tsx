"use client";

// Цэсний icon нэр → lucide. Модулиуд string нэрээр зарладаг тул энд буулгана.
import {
  LayoutGrid,
  Home,
  Store,
  Users,
  ShieldCheck,
  ScrollText,
  MonitorSmartphone,
  Package,
  KeyRound,
  Settings,
  Building2,
  type LucideIcon,
} from "lucide-react";

const map: Record<string, LucideIcon> = {
  dashboard: LayoutGrid,
  home: Home,
  store: Store,
  users: Users,
  shield: ShieldCheck,
  scroll: ScrollText,
  device: MonitorSmartphone,
  package: Package,
  key: KeyRound,
  settings: Settings,
  building: Building2,
};

export function Icon({ name, size = 18 }: { name: string; size?: number }) {
  const C = map[name] || Package;
  return <C size={size} strokeWidth={1.8} />;
}
