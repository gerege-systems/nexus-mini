'use client';

import { useRouter } from 'next/navigation';
import {
  Avatar,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  Icons,
} from '@craftzbay/ui';
import { api, type Me } from '@/lib/api';
import { useThemeMode, type ThemeMode } from '@/lib/theme';
import { locales, setLocale, useT } from '@/lib/i18n';

export function UserMenu({ me, onTenantChange }: { me: Me; onTenantChange?: () => void }) {
  const router = useRouter();
  const [theme, setTheme] = useThemeMode();
  const { t, locale } = useT();

  const initials = me.user.name
    .split(' ')
    .map((s) => s[0])
    .join('')
    .slice(0, 2)
    .toUpperCase();

  const selectTenant = async (id: string) => {
    if (id === me.tenant_id) return;
    await api.post('/api/session/tenant', { tenant_id: id });
    if (onTenantChange) onTenantChange();
    else window.location.reload();
  };

  const logout = async () => {
    await api.post('/api/logout').catch(() => {});
    router.replace('/login');
  };

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="hover:bg-background-muted focus-visible:ring-ring focus-visible:ring-offset-background flex h-9 max-w-[14rem] items-center gap-2 rounded-md px-1.5 outline-none transition-colors focus-visible:ring-2 focus-visible:ring-offset-2"
        >
          <Avatar size="sm" fallback={initials} alt="" />
          <span className="text-foreground hidden truncate text-sm sm:block">{me.user.name}</span>
          <Icons.ChevronDown className="text-foreground-subtle size-4 shrink-0" aria-hidden />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" className="w-64">
        <DropdownMenuLabel className="font-normal">
          <span className="text-foreground block truncate text-sm font-medium">
            {me.user.name}
          </span>
          <span className="text-foreground-subtle block truncate text-xs">{me.user.email}</span>
        </DropdownMenuLabel>

        <DropdownMenuSeparator />
        <DropdownMenuLabel className="text-foreground-subtle text-xs font-normal">
          {t('Байгууллага')}
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={me.tenant_id ?? ''}
          onValueChange={(v) => void selectTenant(v)}
        >
          {me.tenants.map((x) => (
            <DropdownMenuRadioItem key={x.id} value={x.id}>
              {x.name}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        <DropdownMenuItem onSelect={() => router.push('/org/new')}>
          <Icons.Plus aria-hidden />
          {t('Байгууллага нэмэх')}
        </DropdownMenuItem>

        <DropdownMenuSeparator />
        <DropdownMenuLabel className="text-foreground-subtle text-xs font-normal">
          {t('Хэл')}
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup
          value={locale}
          onValueChange={(v) => setLocale(v as (typeof locales)[number]['code'])}
        >
          {locales.map((l) => (
            <DropdownMenuRadioItem key={l.code} value={l.code}>
              {l.label}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>

        <DropdownMenuSeparator />
        <DropdownMenuLabel className="text-foreground-subtle text-xs font-normal">
          {t('Загвар')}
        </DropdownMenuLabel>
        <DropdownMenuRadioGroup value={theme} onValueChange={(v) => setTheme(v as ThemeMode)}>
          <DropdownMenuRadioItem value="light">{t('Цайвар')}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="dark">{t('Бараан')}</DropdownMenuRadioItem>
          <DropdownMenuRadioItem value="system">{t('Систем')}</DropdownMenuRadioItem>
        </DropdownMenuRadioGroup>

        <DropdownMenuSeparator />
        <DropdownMenuItem onSelect={logout}>
          <Icons.LogOut aria-hidden />
          {t('Гарах')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
