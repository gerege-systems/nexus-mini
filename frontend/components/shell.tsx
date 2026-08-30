'use client';

// Ажлын мужийн chrome — @craftzbay/ui дээр. Хоёр түвшний навигац:
// rail = идэвхтэй АПП сонгогч (платформ + суусан модулиуд), panel = тэр аппын цэс.
// Сан нэг түвшний Sidebar өгдөг тул rail-ыг токеноор өөрсдөө угсарна.

import { createContext, useCallback, useContext, useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname, useRouter } from 'next/navigation';
import {
  Alert,
  Button,
  Card,
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
  cn,
  formatDate,
} from '@craftzbay/ui';
import { api, ApiError, type Me, type MenuApp } from '@/lib/api';
import { Icon } from './icons';
import { UserMenu } from './usermenu';
import { useT } from '@/lib/i18n';

type ShellData = { me: Me; menu: MenuApp[]; refresh: () => void; blocked?: boolean };
const ShellCtx = createContext<ShellData | null>(null);
export const useShell = () => useContext(ShellCtx)!;

export function Shell({ children }: { children: React.ReactNode }) {
  const [data, setData] = useState<ShellData | null>(null);
  const [failed, setFailed] = useState(false);
  const [drawer, setDrawer] = useState(false);
  const router = useRouter();
  const pathname = usePathname();
  const { t } = useT();

  const load = useCallback(async () => {
    try {
      const me = await api.get<Me>('/api/me');
      if (!me.tenant_id) {
        // Байгууллагагүй/сонгоогүй: жагсаалтаас эхнийхийг идэвхжүүлнэ.
        if (me.tenants.length > 0) {
          await api.post('/api/session/tenant', { tenant_id: me.tenants[0].id });
          return load();
        }
        router.replace('/org/new');
        return;
      }
      let menu: { apps: MenuApp[] } = { apps: [] };
      try {
        menu = await api.get<{ apps: MenuApp[] }>('/api/menu');
      } catch (e) {
        // Түдгэлзүүлсэн байгууллага → 403. Login руу шидвэл /api/me амжилттай
        // тул дахин dashboard руу буцаж ТӨГСГӨЛГҮЙ ДАВТАЛТ үүсдэг байсан.
        // Оронд нь хаагдсаны тайлбартай дэлгэц.
        if (e instanceof ApiError && e.status === 403) {
          setData({ me, menu: [], blocked: true, refresh: () => void load() });
          return;
        }
        throw e;
      }
      setData({ me, menu: menu.apps || [], refresh: () => void load() });
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        router.replace('/login');
        return;
      }
      setFailed(true);
    }
  }, [router]);

  useEffect(() => {
    void load();
  }, [load]);

  if (failed) {
    return (
      <div className="flex min-h-dvh items-center justify-center p-6">
        <ErrorState variant="500" onRetry={() => window.location.reload()} />
      </div>
    );
  }

  if (!data) {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Spinner label={t('Уншиж байна…')} />
      </div>
    );
  }

  const { me, menu } = data;

  if (data.blocked) return <BlockedScreen me={me} onSwitched={() => void load()} />;

  const perms = me.permissions;
  const tenant = me.tenants.find((x) => x.id === me.tenant_id);
  const isOn = (p: string) => pathname === p || pathname.startsWith(p + '/');

  const adminItems = [
    { path: '/members', label: t('Гишүүд'), icon: 'users', perm: 'core.members.manage' },
    { path: '/roles', label: t('Эрхийн тохиргоо'), icon: 'key', perm: 'core.roles.manage' },
    { path: '/audit', label: t('Audit лог'), icon: 'scroll', perm: 'core.audit.read' },
    { path: '/sso-clients', label: t('SSO клиентүүд'), icon: 'key', perm: 'core.sso.manage' },
    // унших — гишүүн бүр
    { path: '/settings', label: t('Байгууллагын тохиргоо'), icon: 'settings', perm: '' },
  ].filter((i) => !i.perm || perms[i.perm]);

  // Одоогийн зам аль нэг модулийн цэст харьяалагдвал тэр модуль идэвхтэй,
  // үгүй бол платформ — panel зөвхөн идэвхтэй аппын цэсийг харуулна.
  const activeModule = menu.find((m) => m.items.some((i) => isOn(i.path)));

  const panel = (onNavigate?: () => void) =>
    activeModule ? (
      <SidebarSection label={activeModule.name}>
        {activeModule.items.map((i) => (
          <SidebarItem
            key={i.id}
            asChild
            icon={<Icon name={i.icon} />}
            active={isOn(i.path)}
            tooltip={i.label}
          >
            <Link href={i.path} onClick={onNavigate}>
              {i.label}
            </Link>
          </SidebarItem>
        ))}
      </SidebarSection>
    ) : (
      <>
        <SidebarSection label={t('Цэс')}>
          <SidebarItem
            asChild
            icon={<Icon name="dashboard" />}
            active={isOn('/dashboard')}
            tooltip={t('Дашбоард')}
          >
            <Link href="/dashboard" onClick={onNavigate}>
              {t('Дашбоард')}
            </Link>
          </SidebarItem>
          <SidebarItem
            asChild
            icon={<Icon name="store" />}
            active={isOn('/store')}
            tooltip={t('Апп дэлгүүр')}
          >
            <Link href="/store" onClick={onNavigate}>
              {t('Апп дэлгүүр')}
            </Link>
          </SidebarItem>
        </SidebarSection>
        {adminItems.length > 0 && (
          <SidebarSection label={t('Удирдлага')}>
            {adminItems.map((i) => (
              <SidebarItem
                key={i.path}
                asChild
                icon={<Icon name={i.icon} />}
                active={isOn(i.path)}
                tooltip={i.label}
              >
                <Link href={i.path} onClick={onNavigate}>
                  {i.label}
                </Link>
              </SidebarItem>
            ))}
          </SidebarSection>
        )}
      </>
    );

  return (
    <ShellCtx.Provider value={data}>
      <div className="flex min-h-dvh flex-col">
        <TopNav
          logo={
            <div className="flex min-w-0 items-center gap-2">
              <IconButton
                aria-label={t('Цэс нээх')}
                icon={<Icons.Menu />}
                variant="ghost"
                size="sm"
                className="lg:hidden"
                onClick={() => setDrawer(true)}
              />
              <Link
                href="/dashboard"
                className="focus-visible:ring-ring focus-visible:ring-offset-background flex min-w-0 items-center gap-2 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
              >
                <span className="bg-accent text-accent-foreground grid size-7 shrink-0 place-items-center rounded-md text-sm font-semibold">
                  N
                </span>
                <span className="text-foreground truncate text-sm font-semibold">nexus-mini</span>
              </Link>
              {tenant && (
                <>
                  <span className="bg-border hidden h-4 w-px shrink-0 sm:block" aria-hidden />
                  <span className="text-foreground-muted hidden truncate text-sm sm:block">
                    {tenant.name}
                  </span>
                </>
              )}
            </div>
          }
          actions={<UserMenu me={me} onTenantChange={() => void load()} />}
        />

        <div className="flex min-h-0 flex-1">
          {/* rail — аппын сонгогч */}
          <nav
            aria-label={t('Аппууд')}
            className="border-border bg-background hidden w-14 shrink-0 flex-col items-center gap-1 border-r py-2 lg:flex"
          >
            <RailTile
              href="/dashboard"
              label={t('Нүүр')}
              icon={<Icon name="home" className="size-5" />}
              active={!activeModule}
            />
            {menu.map((m) => (
              <RailTile
                key={m.app_id}
                href={m.items[0]?.path || '#'}
                label={m.name}
                icon={<Icon name={m.items[0]?.icon || 'package'} className="size-5" />}
                active={activeModule?.app_id === m.app_id}
              />
            ))}
          </nav>

          <Sidebar className="hidden lg:flex">{panel()}</Sidebar>

          <Sheet open={drawer} onOpenChange={setDrawer}>
            <SheetContent side="left" className="w-72 p-0">
              <SheetTitle className="px-4 py-3 text-sm">{t('Цэс')}</SheetTitle>
              <div className="flex gap-1 overflow-x-auto px-2 pb-2">
                <RailTile
                  href="/dashboard"
                  label={t('Нүүр')}
                  icon={<Icon name="home" className="size-5" />}
                  active={!activeModule}
                  onClick={() => setDrawer(false)}
                />
                {menu.map((m) => (
                  <RailTile
                    key={m.app_id}
                    href={m.items[0]?.path || '#'}
                    label={m.name}
                    icon={<Icon name={m.items[0]?.icon || 'package'} className="size-5" />}
                    active={activeModule?.app_id === m.app_id}
                    onClick={() => setDrawer(false)}
                  />
                ))}
              </div>
              <Sidebar className="border-0">{panel(() => setDrawer(false))}</Sidebar>
            </SheetContent>
          </Sheet>

          <main
            id="main"
            tabIndex={-1}
            className="bg-background-subtle min-w-0 flex-1 overflow-y-auto p-4 outline-none md:p-6"
          >
            <div className="mx-auto max-w-[1440px] space-y-4">
              <TenantBanners me={me} />
              {children}
            </div>
          </main>
        </div>
      </div>
    </ShellCtx.Provider>
  );
}

function RailTile({
  href,
  label,
  icon,
  active,
  onClick,
}: {
  href: string;
  label: string;
  icon: React.ReactNode;
  active?: boolean;
  onClick?: () => void;
}) {
  return (
    <Tooltip label={label} side="right">
      <Link
        href={href}
        aria-label={label}
        aria-current={active ? 'page' : undefined}
        onClick={onClick}
        className={cn(
          'grid size-10 shrink-0 place-items-center rounded-md outline-none transition-colors',
          'focus-visible:ring-ring focus-visible:ring-offset-background focus-visible:ring-2 focus-visible:ring-offset-2',
          active
            ? 'bg-background-muted text-foreground'
            : 'text-foreground-muted hover:bg-background-muted hover:text-foreground',
        )}
      >
        {icon}
      </Link>
    </Tooltip>
  );
}

/** Түдгэлзүүлсэн / устгалд товлогдсон / зөвхөн унших / impersonation мэдэгдэл. */
function TenantBanners({ me }: { me: Me }) {
  const { t } = useT();
  const st = me.tenant_state;
  return (
    <>
      {st?.deletion_at && (
        <Alert variant="danger" title={t('Энэ байгууллага устгалд товлогдсон:')}>
          {formatDate(st.deletion_at, { pattern: 'yyyy-MM-dd' })}.{' '}
          {t('Буцаахыг хүсвэл платформын админтай холбогдоно уу.')}
        </Alert>
      )}
      {st?.suspended && !st.deletion_at && (
        <Alert variant="danger" title={t('Энэ байгууллагыг платформ түдгэлзүүлсэн байна.')}>
          {st.reason ? `${t('Шалтгаан')}: ${st.reason}. ` : ''}
          {t('Өгөгдөлд хандах боломжгүй — платформын админтай холбогдоно уу.')}
        </Alert>
      )}
      {st?.read_only && !st.suspended && (
        <Alert variant="warning">
          {t('Байгууллага зөвхөн уншигдах горимд байна — өөрчлөлт түр хадгалагдахгүй.')}
        </Alert>
      )}
      {me.impersonated_by && (
        <Alert variant="warning">
          {t(
            'Платформын админ энэ хэрэглэгчийн нэрийн өмнөөс нэвтэрсэн байна — бүх үйлдэл audit-д тэмдэглэгдэнэ (30 минутын session).',
          )}
        </Alert>
      )}
    </>
  );
}

/** Байгууллага хаагдсан үед — өгөгдөл харуулахгүй, гарц л үлдээнэ. */
function BlockedScreen({ me, onSwitched }: { me: Me; onSwitched: () => void }) {
  const { t } = useT();
  const st = me.tenant_state;
  const tenantName = me.tenants.find((x) => x.id === me.tenant_id)?.name ?? '';
  const others = me.tenants.filter((x) => x.id !== me.tenant_id);

  return (
    <main className="bg-background-subtle flex min-h-dvh items-center justify-center p-4">
      <Card className="w-full max-w-lg">
        <h1 className="text-foreground mb-3 text-lg font-semibold">{tenantName}</h1>
        <Alert variant="danger" className="mb-4">
          {st?.deletion_at
            ? `${t('Энэ байгууллага устгалд товлогдсон:')} ${formatDate(st.deletion_at, { pattern: 'yyyy-MM-dd' })}`
            : t('Энэ байгууллагыг платформ түдгэлзүүлсэн байна.')}
          {st?.reason ? ` ${t('Шалтгаан')}: ${st.reason}.` : ''}
        </Alert>
        <p className="text-foreground-muted mb-4 text-sm">
          {t('Өгөгдөлд хандах боломжгүй — платформын админтай холбогдоно уу.')}
        </p>
        <div className="flex flex-wrap gap-2">
          {others.map((x) => (
            <Button
              key={x.id}
              variant="secondary"
              onClick={async () => {
                await api.post('/api/session/tenant', { tenant_id: x.id });
                onSwitched();
              }}
            >
              {x.name}
            </Button>
          ))}
          <Button
            variant="ghost"
            leadingIcon={<Icons.LogOut />}
            onClick={async () => {
              await api.post('/api/logout').catch(() => {});
              window.location.assign('/login');
            }}
          >
            {t('Гарах')}
          </Button>
        </div>
      </Card>
    </main>
  );
}
