"use client";

// Ажлын мужийн chrome — gerege-ui-ийн «Rail + panel» (dual) загвар
// (ui.gecore.mn → Templates → Admin → Rail + panel). 56px icon rail = аппууд
// (платформ + суусан модулиуд), 240px panel = идэвхтэй аппын цэс, TopNav =
// breadcrumb (≥lg) / хуудсын нэр (<lg) + ⌘K palette + хэрэглэгчийн цэс.
// <lg: Sheet drawer (модулийн таб + цэс + хэрэглэгч). Идэвхтэй = accent зураас +
// фон + жин, хэзээ ч зөвхөн өнгөөр биш. Токеноос гадуур өнгө/зай байхгүй.

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  Alert,
  Avatar,
  Breadcrumbs,
  Button,
  Card,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
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
  cn,
  formatDate,
  useCommandPaletteShortcut,
  useModifierKey,
} from "@gerege-systems/ui";
import { api, ApiError, type Me, type MenuApp } from "@/lib/api";
import { Icon } from "./icons";
import { PasswordGate } from "./password-gate";
import { UserMenu } from "./usermenu";
import { useT } from "@/lib/i18n";

type ShellData = { me: Me; menu: MenuApp[]; refresh: () => void; blocked?: boolean; mustChange?: boolean };
const ShellCtx = createContext<ShellData | null>(null);
export const useShell = () => useContext(ShellCtx)!;

type NavItem = { path: string; label: string; icon: string };
type NavSection = { label: string; items: NavItem[] };
/** Rail-ийн нэг нүд: платформ эсвэл суусан модуль. */
type Module = { key: string; name: string; icon: string; sections: NavSection[] };
const PLATFORM = "platform";

export function Shell({ children }: { children: React.ReactNode }) {
  const [data, setData] = useState<ShellData | null>(null);
  const [failed, setFailed] = useState(false);
  const router = useRouter();
  const { t } = useT();

  const load = useCallback(async () => {
    try {
      const me = await api.get<Me>("/api/me");
      // Түр нууц үг солиогүй: tenant-ийн API бүр 403 тул menu-г дуудахгүй.
      if (me.must_change_password) {
        setData({ me, menu: [], mustChange: true, refresh: () => void load() });
        return;
      }
      if (!me.tenant_id) {
        // Байгууллагагүй/сонгоогүй: жагсаалтаас эхнийхийг идэвхжүүлнэ.
        if (me.tenants.length > 0) {
          await api.post("/api/session/tenant", { tenant_id: me.tenants[0].id });
          return load();
        }
        router.replace("/org/new");
        return;
      }
      let menu: { apps: MenuApp[] } = { apps: [] };
      try {
        menu = await api.get<{ apps: MenuApp[] }>("/api/menu");
      } catch (e) {
        // Түдгэлзүүлсэн байгууллага → 403. Login руу шидвэл /api/me амжилттай
        // тул дахин dashboard руу буцаж ТӨГСГӨЛГҮЙ ДАВТАЛТ үүсдэг байсан.
        if (e instanceof ApiError && e.status === 403) {
          setData({ me, menu: [], blocked: true, refresh: () => void load() });
          return;
        }
        throw e;
      }
      setData({ me, menu: menu.apps || [], refresh: () => void load() });
    } catch (e) {
      if (e instanceof ApiError && e.status === 401) {
        router.replace("/login");
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
        <Spinner label={t("Уншиж байна…")} />
      </div>
    );
  }
  if (data.mustChange) return <PasswordGate me={data.me} onDone={() => void load()} />;
  if (data.blocked) return <BlockedScreen me={data.me} onSwitched={() => void load()} />;

  return (
    <ShellCtx.Provider value={data}>
      <Workspace data={data}>{children}</Workspace>
    </ShellCtx.Provider>
  );
}

/* ---------------------------------------------------------------------------
 *  Workspace — rail + panel + top bar + drawer + palette
 * ------------------------------------------------------------------------ */

function Workspace({ data, children }: { data: ShellData; children: ReactNode }) {
  const { me, menu } = data;
  const pathname = usePathname();
  const router = useRouter();
  const { t } = useT();
  const perms = me.permissions;
  const isOn = useCallback((p: string) => pathname === p || pathname.startsWith(p + "/"), [pathname]);

  const modules = useMemo<Module[]>(() => {
    const admin: NavItem[] = [
      { path: "/members", label: t("Гишүүд"), icon: "users" },
      { path: "/roles", label: t("Эрхийн тохиргоо"), icon: "key" },
      { path: "/audit", label: t("Audit лог"), icon: "scroll" },
      { path: "/sso-clients", label: t("SSO клиентүүд"), icon: "key" },
      { path: "/settings", label: t("Байгууллагын тохиргоо"), icon: "settings" },
    ].filter((i) => {
      const need: Record<string, string> = {
        "/members": "core.members.manage",
        "/roles": "core.roles.manage",
        "/audit": "core.audit.read",
        "/sso-clients": "core.sso.manage",
      };
      return !need[i.path] || perms[need[i.path]];
    });
    const platform: Module = {
      key: PLATFORM,
      name: t("Нүүр"),
      icon: "home",
      sections: [
        {
          label: t("Цэс"),
          items: [
            { path: "/dashboard", label: t("Дашбоард"), icon: "dashboard" },
            { path: "/store", label: t("Апп дэлгүүр"), icon: "store" },
          ],
        },
        ...(admin.length ? [{ label: t("Удирдлага"), items: admin }] : []),
      ],
    };
    const apps: Module[] = menu.map((m) => ({
      key: m.app_id,
      name: m.name,
      icon: m.items[0]?.icon || "package",
      sections: [{ label: m.name, items: m.items.map((i) => ({ path: i.path, label: i.label, icon: i.icon })) }],
    }));
    return [platform, ...apps];
  }, [menu, perms, t]);

  // Идэвхтэй модуль = одоогийн зам аль модулийн цэст байна; олдохгүй бол платформ.
  const activeModule = useMemo(
    () => modules.find((m) => m.sections.some((s) => s.items.some((i) => isOn(i.path)))) ?? modules[0],
    [modules, isOn],
  );
  const activeItem = activeModule.sections.flatMap((s) => s.items).find((i) => isOn(i.path));

  // Panel-д харуулах модуль: зам солигдоход идэвхтэй модуль руу; rail дарахад
  // зөвхөн panel солигдоно (загварын дүрэм — навигац хийхгүй).
  const [panelKey, setPanelKey] = useState(activeModule.key);
  useEffect(() => setPanelKey(activeModule.key), [activeModule.key]);
  const panelModule = modules.find((m) => m.key === panelKey) ?? activeModule;

  const [drawer, setDrawer] = useState(false);
  const drawerTriggerRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    // Drawer зөвхөн <lg; цонх томровол хаана — rail-ийн дээр үлдэхгүй.
    const mq = window.matchMedia("(min-width: 1024px)");
    const sync = () => {
      if (mq.matches) setDrawer(false);
    };
    sync();
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);
  useEffect(() => setDrawer(false), [pathname]);

  const [palette, setPalette] = useState(false);
  useCommandPaletteShortcut(setPalette);
  const mod = useModifierKey();

  const crumbs =
    pathname === "/dashboard" || !activeItem
      ? null
      : [
          { label: t("Нүүр"), href: "/dashboard" },
          ...(activeModule.key !== PLATFORM
            ? [{ label: activeModule.name, href: activeModule.sections[0].items[0].path }]
            : []),
          { label: activeItem.label },
        ];

  const navList = (m: Module) =>
    m.sections.map((s) => (
      <SidebarSection key={s.label} label={s.label}>
        {s.items.map((i) => (
          <SidebarItem
            key={i.path}
            asChild
            icon={<Icon name={i.icon} />}
            active={isOn(i.path)}
            className={cn(
              "before:bg-accent relative before:absolute before:top-1.5 before:bottom-1.5 before:left-0 before:w-0.5 before:rounded-full before:opacity-0",
              isOn(i.path) && "text-foreground before:opacity-100",
            )}
          >
            <Link href={i.path}>{i.label}</Link>
          </SidebarItem>
        ))}
      </SidebarSection>
    ));

  return (
    <div className="bg-background text-foreground flex h-dvh overflow-hidden">
      <AppRail modules={modules} value={panelKey} onChange={setPanelKey} />

      <Sidebar
        aria-label={t("Модулийн цэс") + ": " + panelModule.name}
        header={
          <>
            <h2 className="sr-only">{panelModule.name}</h2>
            <TenantSwitcher me={me} onSwitched={data.refresh} />
          </>
        }
        // Panel тогтмол өргөнтэй: хумих товчгүй (rail нь угаас icon давхарга).
        className="md:hidden lg:flex [&>button:last-child]:hidden"
      >
        {navList(panelModule)}
      </Sidebar>

      <div className="flex min-h-0 min-w-0 flex-1 flex-col">
        <TopNav
          logo={
            <div className="flex min-w-0 items-center gap-2">
              <IconButton
                ref={drawerTriggerRef}
                aria-label={t("Цэс нээх")}
                icon={<Icons.Menu />}
                variant="ghost"
                size="sm"
                className="lg:hidden"
                onClick={() => setDrawer(true)}
              />
              {crumbs ? (
                <Breadcrumbs
                  items={crumbs}
                  className="hidden min-w-0 lg:block"
                  renderLink={(href, label) => (
                    <Link
                      href={href}
                      className="hover:text-foreground focus-visible:ring-ring rounded-sm transition-colors outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
                    >
                      {label}
                    </Link>
                  )}
                />
              ) : (
                <span className="text-foreground hidden truncate text-sm font-semibold lg:block">
                  {activeItem?.label ?? t("Нүүр")}
                </span>
              )}
              <span className="text-foreground truncate text-sm font-semibold lg:hidden">
                {activeItem?.label ?? t("Нүүр")}
              </span>
            </div>
          }
          actions={
            <>
              <Tooltip label={`${t("Хайх, шилжих")} · ${mod.label}K`}>
                <IconButton
                  aria-label={t("Хайх, шилжих")}
                  icon={<Icons.Search />}
                  variant="ghost"
                  size="sm"
                  onClick={() => setPalette(true)}
                />
              </Tooltip>
              <UserMenu me={me} />
            </>
          }
        />
        <main
          id="main"
          tabIndex={-1}
          // `relative`: scroll pane нь absolute (sr-only) удамдаа containing block байх ёстой.
          className="bg-background-subtle relative min-h-0 flex-1 overflow-y-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))] outline-none md:p-6 md:pb-[max(1.5rem,env(safe-area-inset-bottom))]"
        >
          <div className="mx-auto max-w-[1440px] space-y-4">
            <TenantBanners me={me} />
            {children}
          </div>
        </main>
      </div>

      <Sheet open={drawer} onOpenChange={setDrawer}>
        <SheetContent
          aria-describedby={undefined}
          side="left"
          className="w-64 p-0"
          showClose={false}
          onCloseAutoFocus={(e) => {
            if (!drawerTriggerRef.current) return;
            e.preventDefault();
            drawerTriggerRef.current.focus();
          }}
        >
          <SheetTitle className="sr-only">{t("Навигац")}</SheetTitle>
          <Sidebar
            header={<ModuleTabs modules={modules} value={panelKey} onChange={setPanelKey} />}
            footer={<UserCard me={me} />}
            className="flex h-full w-full border-r-0 [&>button:last-child]:hidden"
          >
            <div className="px-4 pb-2">
              <h2 className="text-foreground truncate text-sm font-semibold">{panelModule.name}</h2>
            </div>
            {navList(panelModule)}
          </Sidebar>
        </SheetContent>
      </Sheet>

      <CommandDialog open={palette} onOpenChange={setPalette} title={t("Хайх, шилжих")}>
        <CommandInput placeholder={t("Хуудас хайх…")} />
        <CommandList>
          <CommandEmpty>{t("Олдсонгүй")}</CommandEmpty>
          {modules.flatMap((m) =>
            m.sections.map((s) => (
              <CommandGroup key={m.key + s.label} heading={m.key === PLATFORM ? s.label : m.name}>
                {s.items.map((i) => (
                  <CommandItem
                    key={i.path}
                    value={`${m.name} ${s.label} ${i.label}`}
                    onSelect={() => {
                      setPalette(false);
                      router.push(i.path);
                    }}
                  >
                    <Icon name={i.icon} /> {i.label}
                  </CommandItem>
                ))}
              </CommandGroup>
            )),
          )}
        </CommandList>
      </CommandDialog>
    </div>
  );
}

/* ---------------------------------------------------------------------------
 *  Rail — модулиудын icon давхарга (≥lg). Brand дээр, модулиуд дундаа гүйлгэгдэнэ.
 * ------------------------------------------------------------------------ */

function AppRail({ modules, value, onChange }: { modules: Module[]; value: string; onChange: (k: string) => void }) {
  const { t } = useT();
  return (
    <nav
      aria-label={t("Аппууд")}
      className="border-border bg-background-subtle hidden w-14 shrink-0 flex-col items-center gap-1 border-r py-2 lg:flex"
    >
      <Link
        href="/dashboard"
        aria-label="nexus-mini"
        className="bg-foreground text-background focus-visible:ring-ring focus-visible:ring-offset-background mb-2 inline-flex size-8 items-center justify-center rounded-md text-sm font-semibold outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
      >
        N
      </Link>
      <div className="relative min-h-0 w-full flex-1">
        <div
          className="flex h-full w-full [scrollbar-width:none] flex-col items-center gap-1 overflow-y-auto overscroll-contain py-0.5 [&::-webkit-scrollbar]:hidden"
          style={{ maskImage: "linear-gradient(to bottom, transparent, #000 10px, #000 calc(100% - 10px), transparent)" }}
        >
          {modules.map((m) => (
            <Tooltip key={m.key} label={m.name} side="right">
              <button
                type="button"
                aria-label={m.name}
                aria-current={m.key === value ? "page" : undefined}
                onClick={() => onChange(m.key)}
                className={cn(
                  "relative inline-flex size-10 items-center justify-center rounded-md outline-none [&_svg]:size-5",
                  "transition-colors duration-[var(--duration-fast)]",
                  "focus-visible:ring-ring focus-visible:ring-offset-background focus-visible:ring-2 focus-visible:ring-offset-2",
                  // Идэвхтэй = rail-ийн зүүн ирмэгт accent зураас + фон + accent icon.
                  "before:bg-accent before:absolute before:top-2 before:bottom-2 before:-left-2 before:w-0.5 before:rounded-r-full before:opacity-0",
                  m.key === value
                    ? "bg-background-muted text-accent before:opacity-100"
                    : "text-foreground-muted hover:bg-surface-hover hover:text-foreground",
                )}
              >
                <Icon name={m.icon} className="size-5" />
              </button>
            </Tooltip>
          ))}
        </div>
      </div>
    </nav>
  );
}

/** Drawer доторх rail-ийн орлуулагч — сегмент таб. */
function ModuleTabs({ modules, value, onChange }: { modules: Module[]; value: string; onChange: (k: string) => void }) {
  const { t } = useT();
  return (
    <div
      role="group"
      aria-label={t("Аппууд")}
      className="bg-background-muted flex w-full snap-x [scrollbar-width:none] gap-0.5 overflow-x-auto rounded-md p-0.5 [&::-webkit-scrollbar]:hidden"
    >
      {modules.map((m) => {
        const active = m.key === value;
        return (
          <button
            key={m.key}
            type="button"
            aria-pressed={active}
            onClick={() => onChange(m.key)}
            ref={active ? (el) => el?.scrollIntoView({ block: "nearest", inline: "nearest" }) : undefined}
            className={cn(
              "inline-flex h-8 shrink-0 snap-start items-center justify-center gap-1.5 rounded-md px-2.5 text-sm whitespace-nowrap outline-none [&_svg]:size-4",
              "transition-colors duration-[var(--duration-fast)]",
              "focus-visible:ring-ring focus-visible:ring-offset-background focus-visible:ring-2 focus-visible:ring-offset-2",
              active ? "bg-background text-foreground font-medium shadow-sm" : "text-foreground-muted hover:text-foreground",
            )}
          >
            <Icon name={m.icon} />
            {m.name}
          </button>
        );
      })}
    </div>
  );
}

/* ---------------------------------------------------------------------------
 *  Tenant switcher — panel-ийн толгой (загварын WorkspaceSwitcher)
 * ------------------------------------------------------------------------ */

function TenantSwitcher({ me, onSwitched }: { me: Me; onSwitched: () => void }) {
  const { t } = useT();
  const router = useRouter();
  const current = me.tenants.find((x) => x.id === me.tenant_id) ?? me.tenants[0];
  const select = async (id: string) => {
    if (id === me.tenant_id) return;
    await api.post("/api/session/tenant", { tenant_id: id });
    onSwitched();
  };
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          className="hover:bg-background-muted focus-visible:ring-ring focus-visible:ring-offset-background flex h-10 w-full items-center gap-2 rounded-md px-1.5 text-left transition-colors outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
        >
          <Avatar size="md" fallback={(current?.name ?? "?").slice(0, 1).toUpperCase()} alt="" className="rounded-md [&_span]:rounded-md" />
          <span className="text-foreground min-w-0 flex-1 truncate text-sm font-semibold" title={current?.name}>
            {current?.name}
          </span>
          <Icons.ChevronsUpDown className="text-foreground-subtle size-4 shrink-0" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-[232px]">
        <DropdownMenuLabel>{t("Байгууллага")}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {me.tenants.map((x) => (
          <DropdownMenuItem key={x.id} onSelect={() => void select(x.id)} className="gap-2">
            <Avatar size="sm" fallback={x.name.slice(0, 1).toUpperCase()} alt="" />
            <span className="text-foreground min-w-0 flex-1 truncate text-sm" title={x.name}>
              {x.name}
            </span>
            {x.id === me.tenant_id && <Icons.Check className="text-accent size-4" aria-hidden />}
          </DropdownMenuItem>
        ))}
        <DropdownMenuSeparator />
        <DropdownMenuItem className="gap-2" onSelect={() => router.push("/org/new")}>
          <span className="border-border text-foreground-subtle inline-flex size-6 items-center justify-center rounded-md border border-dashed">
            <Icons.Plus className="size-3.5" aria-hidden />
          </span>
          {t("Байгууллага нэмэх")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function UserCard({ me }: { me: Me }) {
  const initials = me.user.name.split(" ").map((s) => s[0]).join("").slice(0, 2).toUpperCase();
  return (
    <div className="flex items-center gap-2 px-1">
      <Avatar size="sm" fallback={initials} alt={me.user.name} />
      <div className="min-w-0 leading-tight">
        <div className="text-foreground truncate text-sm font-medium" title={me.user.name}>
          {me.user.name}
        </div>
        <div className="text-foreground-subtle truncate text-xs" title={me.user.email}>
          {me.user.email}
        </div>
      </div>
    </div>
  );
}

/** Түдгэлзүүлсэн / устгалд товлогдсон / зөвхөн унших / impersonation мэдэгдэл. */
function TenantBanners({ me }: { me: Me }) {
  const { t } = useT();
  const st = me.tenant_state;
  return (
    <>
      {st?.deletion_at && (
        <Alert variant="danger" title={t("Энэ байгууллага устгалд товлогдсон:")}>
          {formatDate(st.deletion_at, { pattern: "yyyy-MM-dd" })}. {t("Буцаахыг хүсвэл платформын админтай холбогдоно уу.")}
        </Alert>
      )}
      {st?.suspended && !st.deletion_at && (
        <Alert variant="danger" title={t("Энэ байгууллагыг платформ түдгэлзүүлсэн байна.")}>
          {st.reason ? `${t("Шалтгаан")}: ${st.reason}. ` : ""}
          {t("Өгөгдөлд хандах боломжгүй — платформын админтай холбогдоно уу.")}
        </Alert>
      )}
      {st?.read_only && !st.suspended && (
        <Alert variant="warning">{t("Байгууллага зөвхөн уншигдах горимд байна — өөрчлөлт түр хадгалагдахгүй.")}</Alert>
      )}
      {me.impersonated_by && (
        <Alert variant="warning">
          {t("Платформын админ энэ хэрэглэгчийн нэрийн өмнөөс нэвтэрсэн байна — бүх үйлдэл audit-д тэмдэглэгдэнэ (30 минутын session).")}
        </Alert>
      )}
    </>
  );
}

/** Байгууллага хаагдсан үед — өгөгдөл харуулахгүй, гарц л үлдээнэ. */
function BlockedScreen({ me, onSwitched }: { me: Me; onSwitched: () => void }) {
  const { t } = useT();
  const st = me.tenant_state;
  const tenantName = me.tenants.find((x) => x.id === me.tenant_id)?.name ?? "";
  const others = me.tenants.filter((x) => x.id !== me.tenant_id);
  return (
    <main className="bg-background-subtle flex min-h-dvh items-center justify-center p-4">
      <Card className="w-full max-w-lg">
        <h1 className="text-foreground mb-3 text-lg font-semibold">{tenantName}</h1>
        <Alert variant="danger" className="mb-4">
          {st?.deletion_at
            ? `${t("Энэ байгууллага устгалд товлогдсон:")} ${formatDate(st.deletion_at, { pattern: "yyyy-MM-dd" })}`
            : t("Энэ байгууллагыг платформ түдгэлзүүлсэн байна.")}
          {st?.reason ? ` ${t("Шалтгаан")}: ${st.reason}.` : ""}
        </Alert>
        <p className="text-foreground-muted mb-4 text-sm">{t("Өгөгдөлд хандах боломжгүй — платформын админтай холбогдоно уу.")}</p>
        <div className="flex flex-wrap gap-2">
          {others.map((x) => (
            <Button
              key={x.id}
              variant="secondary"
              onClick={async () => {
                await api.post("/api/session/tenant", { tenant_id: x.id });
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
              await api.post("/api/logout").catch(() => {});
              window.location.assign("/login");
            }}
          >
            {t("Гарах")}
          </Button>
        </div>
      </Card>
    </main>
  );
}
