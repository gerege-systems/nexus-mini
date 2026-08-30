'use client';

// Цэсний icon нэр → сангийн нэрээр хандах <Icon>. Модулиуд icon-оо богино
// string нэрээр зарладаг (modules.json / manifest) тул энд lucide-ийн бодит
// нэр рүү буулгана. Сангийн Icon нь lazy — бүх icon багц bundle-д ордоггүй.
import { Icon as UiIcon } from '@craftzbay/ui/icon';

const MAP: Record<string, string> = {
  dashboard: 'layout-grid',
  home: 'house',
  store: 'store',
  users: 'users',
  shield: 'shield-check',
  scroll: 'scroll-text',
  device: 'monitor-smartphone',
  package: 'package',
  key: 'key-round',
  settings: 'settings',
  building: 'building-2',
};

export function Icon({ name, className }: { name: string; className?: string }) {
  return <UiIcon name={(MAP[name] ?? 'package') as never} className={className ?? 'size-4'} />;
}
