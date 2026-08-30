'use client';

// Админ аппын chrome — бүхэлдээ @craftzbay/ui дээр: TopNav + Sidebar + Sheet.
// Гараар бичсэн CSS класс байхгүй; өнгө/зай/радиус бүгд theme.css-ийн токеноос.

import { createContext, useContext, useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  Avatar,
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
  ErrorState,
  Icons,
  IconButton,
  Sheet,
  SheetContent,
  SheetTitle,
  Sidebar,
  SidebarItem,
  SidebarSection,
  Spinner,
  Tooltip,
  TopNav,
} from '@craftzbay/ui';
import { Icon } from '@craftzbay/ui/icon';
import { api, ApiError, type Me } from '@/lib/api';
import { locales, setLocale, useT } from '@/lib/i18n';
import { useThemeMode, type ThemeMode } from '@/lib/theme';

const MeCtx = createContext<Me | null>(null);
export const useMe = () => useContext(MeCtx)!;

/** label нь t()-ийн түлхүүр — render үедээ орчуулагдана. */
const NAV = [
  { path: '/', label: 'Тойм', icon: <Icons.LayoutGrid /> },
  { path: '/tenants', label: 'Байгууллагууд', icon: <Icon name="building-2" /> },
  { path: '/users', label: 'Хэрэглэгчид', icon: <Icons.Users /> },
  { path: '/apps', label: 'Каталог', icon: <Icons.Package /> },
  { path: '/audit', label: 'Audit', icon: <Icons.FileText /> },
  { path: '/profile', label: 'Профайл', icon: <Icons.User /> },
];

const THEME_ICON: Record<ThemeMode, React.ReactNode> = {
  light: <Icons.Sun />,
  dark: <Icons.Moon />,
  system: <Icon name="monitor" />,
};

type Status = 'loading' | 'ok' | 'denied' | 'error';

export function Shell({ children }: { children: React.ReactNode }) {
  const [me, setMe] = useState<Me | null>(null);
  const [status, setStatus] = useState<Status>('loading');
  const [drawer, setDrawer] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const { t } = useT();

  useEffect(() => {
    let alive = true;
    api
      .get<Me>('/api/me')
      .then((m) => {
        if (!alive) return;
        if (!m.user.platform_admin) {
          setStatus('denied');
          return;
        }
        setMe(m);
        setStatus('ok');
      })
      .catch((e) => {
        if (!alive) return;
        // 401-ийг api клиент өөрөө /login руу шилжүүлнэ.
        if (e instanceof ApiError && e.status === 401) return;
        setStatus('error');
      });
    return () => {
      alive = false;
    };
  }, []);

  if (status === 'loading') {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Spinner label={t('Уншиж байна…')} />
      </div>
    );
  }

  if (status === 'denied') {
    return (
      <div className="flex min-h-dvh items-center justify-center p-6">
        <ErrorState
          variant="403"
          description={t('Энэ систем зөвхөн платформын админд зориулагдсан')}
          action={
            <Button
              variant="secondary"
              onClick={() => api.post('/api/logout').finally(() => router.replace('/login'))}
            >
              {t('Гарах')}
            </Button>
          }
        />
      </div>
    );
  }

  if (status === 'error' || !me) {
    return (
      <div className="flex min-h-dvh items-center justify-center p-6">
        <ErrorState variant="500" onRetry={() => window.location.reload()} />
      </div>
    );
  }

  const isOn = (p: string) => (p === '/' ? pathname === '/' : pathname.startsWith(p));

  const navItems = (onNavigate?: () => void) => (
    <SidebarSection>
      {NAV.map((n) => (
        <SidebarItem key={n.path} asChild icon={n.icon} active={isOn(n.path)} tooltip={t(n.label)}>
          <Link href={n.path} onClick={onNavigate}>
            {t(n.label)}
          </Link>
        </SidebarItem>
      ))}
    </SidebarSection>
  );

  return (
    <MeCtx.Provider value={me}>
      <div className="flex min-h-dvh flex-col">
        <AdminTopNav me={me} onOpenDrawer={() => setDrawer(true)} />

        <div className="flex min-h-0 flex-1">
          <Sidebar className="hidden lg:flex">{navItems()}</Sidebar>

          <Sheet open={drawer} onOpenChange={setDrawer}>
            <SheetContent side="left" className="w-64 p-0">
              <SheetTitle className="px-4 py-3 text-sm">{t('Цэс')}</SheetTitle>
              <Sidebar className="border-0">{navItems(() => setDrawer(false))}</Sidebar>
            </SheetContent>
          </Sheet>

          <main
            id="main"
            tabIndex={-1}
            className="bg-background-subtle min-w-0 flex-1 overflow-y-auto p-4 outline-none md:p-6"
          >
            <div className="mx-auto max-w-[1440px]">{children}</div>
          </main>
        </div>
      </div>
    </MeCtx.Provider>
  );
}

function AdminTopNav({ me, onOpenDrawer }: { me: Me; onOpenDrawer: () => void }) {
  const router = useRouter();
  const { t, locale } = useT();
  const [theme, setTheme] = useThemeMode();

  const logout = async () => {
    await api.post('/api/logout').catch(() => {});
    router.replace('/login');
  };

  return (
    <TopNav
      logo={
        <div className="flex min-w-0 items-center gap-2">
          <IconButton
            aria-label={t('Цэс нээх')}
            icon={<Icons.Menu />}
            variant="ghost"
            size="sm"
            className="lg:hidden"
            onClick={onOpenDrawer}
          />
          <Link
            href="/"
            className="focus-visible:ring-ring focus-visible:ring-offset-background flex min-w-0 items-center gap-2 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
          >
            <span className="bg-accent text-accent-foreground grid size-7 shrink-0 place-items-center rounded-md text-sm font-semibold">
              N
            </span>
            <span className="text-foreground truncate text-sm font-semibold">
              {t('Платформын админ')}
            </span>
          </Link>
        </div>
      }
      actions={
        <>
          <DropdownMenu>
            <Tooltip label={t('Хэл')}>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="px-2 tabular-nums">
                  {locale.toUpperCase()}
                </Button>
              </DropdownMenuTrigger>
            </Tooltip>
            <DropdownMenuContent align="end" className="w-36">
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
            </DropdownMenuContent>
          </DropdownMenu>

          <DropdownMenu>
            <Tooltip label={t('Өнгө')}>
              <DropdownMenuTrigger asChild>
                <IconButton aria-label={t('Өнгө')} icon={THEME_ICON[theme]} variant="ghost" size="sm" />
              </DropdownMenuTrigger>
            </Tooltip>
            <DropdownMenuContent align="end" className="w-40">
              <DropdownMenuRadioGroup
                value={theme}
                onValueChange={(v) => setTheme(v as ThemeMode)}
              >
                <DropdownMenuRadioItem value="light">{t('Цайвар')}</DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="dark">{t('Бараан')}</DropdownMenuRadioItem>
                <DropdownMenuRadioItem value="system">{t('Системийн')}</DropdownMenuRadioItem>
              </DropdownMenuRadioGroup>
            </DropdownMenuContent>
          </DropdownMenu>

          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button
                type="button"
                aria-label={me.user.name}
                className="focus-visible:ring-ring focus-visible:ring-offset-background rounded-full outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
              >
                <Avatar size="sm" fallback={me.user.name.slice(0, 1)} alt="" />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-60">
              <DropdownMenuLabel className="font-normal">
                <span className="text-foreground block truncate text-sm font-medium">
                  {me.user.name}
                </span>
                <span className="text-foreground-subtle block truncate text-xs">
                  {me.user.email}
                </span>
              </DropdownMenuLabel>
              <DropdownMenuSeparator />
              <DropdownMenuItem asChild>
                <Link href="/profile">
                  <Icons.User aria-hidden />
                  {t('Профайл')}
                </Link>
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem onSelect={logout}>
                <Icons.LogOut aria-hidden />
                {t('Гарах')}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </>
      }
    />
  );
}
