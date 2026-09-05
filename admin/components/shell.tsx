"use client";

// Админ аппын chrome — gerege-ui-ийн «Sidebar rail» загвар (ui.gecore.mn →
// Templates → Admin → Sidebar rail): 240px sidebar ≥lg (хумихад 56px icon rail,
// төлөв хөтчид хадгалагдана), <lg Sheet drawer; TopNav = breadcrumb (≥lg) / хуудсын нэр
// (<lg) + хэл + загвар + хэрэглэгчийн цэс. Идэвхтэй = accent зураас + фон + жин.

import { createContext, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import {
  Avatar,
  Breadcrumbs,
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
  cn,
  useSidebar,
} from "@gerege-systems/ui";
import { Icon } from "@gerege-systems/ui/icon";
import { api, ApiError, type Me } from "@/lib/api";
import { locales, setLocale, useT } from "@/lib/i18n";
import { useThemeMode, type ThemeMode } from "@/lib/theme";

const MeCtx = createContext<Me | null>(null);
export const useMe = () => useContext(MeCtx)!;

/** label нь t()-ийн түлхүүр — render үедээ орчуулагдана. */
const NAV: { path: string; label: string; icon: ReactNode }[] = [
  { path: "/", label: "Тойм", icon: <Icons.LayoutGrid /> },
  { path: "/tenants", label: "Байгууллагууд", icon: <Icon name="building-2" /> },
  { path: "/users", label: "Хэрэглэгчид", icon: <Icons.Users /> },
  { path: "/apps", label: "Каталог", icon: <Icons.Package /> },
  { path: "/audit", label: "Audit", icon: <Icons.FileText /> },
  { path: "/profile", label: "Профайл", icon: <Icons.User /> },
];

const COLLAPSED_KEY = "nexus-admin:sidebar-collapsed";

const THEME_ICON: Record<ThemeMode, ReactNode> = {
  light: <Icons.Sun />,
  dark: <Icons.Moon />,
  system: <Icon name="monitor" />,
};

type Status = "loading" | "ok" | "denied" | "error";

export function Shell({ children }: { children: ReactNode }) {
  const [me, setMe] = useState<Me | null>(null);
  const [status, setStatus] = useState<Status>("loading");
  const router = useRouter();
  const { t } = useT();

  useEffect(() => {
    let alive = true;
    api
      .get<Me>("/api/me")
      .then((m) => {
        if (!alive) return;
        if (!m.user.platform_admin) {
          setStatus("denied");
          return;
        }
        setMe(m);
        setStatus("ok");
      })
      .catch((e) => {
        if (!alive) return;
        // 401-ийг api клиент өөрөө /login руу шилжүүлнэ.
        if (e instanceof ApiError && e.status === 401) return;
        setStatus("error");
      });
    return () => {
      alive = false;
    };
  }, []);

  if (status === "loading") {
    return (
      <div className="flex min-h-dvh items-center justify-center">
        <Spinner label={t("Уншиж байна…")} />
      </div>
    );
  }
  if (status === "denied") {
    return (
      <div className="flex min-h-dvh items-center justify-center p-6">
        <ErrorState
          variant="403"
          description={t("Энэ систем зөвхөн платформын админд зориулагдсан")}
          action={
            <Button variant="secondary" onClick={() => api.post("/api/logout").finally(() => router.replace("/login"))}>
              {t("Гарах")}
            </Button>
          }
        />
      </div>
    );
  }
  if (status === "error" || !me) {
    return (
      <div className="flex min-h-dvh items-center justify-center p-6">
        <ErrorState variant="500" onRetry={() => window.location.reload()} />
      </div>
    );
  }
  return (
    <MeCtx.Provider value={me}>
      <Workspace me={me}>{children}</Workspace>
    </MeCtx.Provider>
  );
}

function Workspace({ me, children }: { me: Me; children: ReactNode }) {
  const pathname = usePathname();
  const { t } = useT();
  const isOn = (p: string) => (p === "/" ? pathname === "/" : pathname.startsWith(p));
  const current = NAV.find((n) => isOn(n.path));

  // Хумьсан төлөв — төхөөрөмжийн харагдац, localStorage (эхний render-д биш:
  // SSR-тэй таарахгүй байх эрсдэл).
  const [collapsed, setCollapsed] = useState(false);
  useEffect(() => {
    try {
      setCollapsed(localStorage.getItem(COLLAPSED_KEY) === "1");
    } catch {
      /* private mode */
    }
  }, []);
  const onCollapsedChange = (next: boolean) => {
    setCollapsed(next);
    try {
      localStorage.setItem(COLLAPSED_KEY, next ? "1" : "0");
    } catch {
      /* private mode */
    }
  };

  const [drawer, setDrawer] = useState(false);
  const drawerTriggerRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    const mq = window.matchMedia("(min-width: 1024px)");
    const sync = () => {
      if (mq.matches) setDrawer(false);
    };
    sync();
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);
  useEffect(() => setDrawer(false), [pathname]);

  const crumbs =
    !current || current.path === "/" ? null : [{ label: t("Тойм"), href: "/" }, { label: t(current.label) }];

  const nav = (
    <SidebarSection>
      {NAV.map((n) => (
        <SidebarItem
          key={n.path}
          asChild
          icon={n.icon}
          active={isOn(n.path)}
          tooltip={t(n.label)}
          className={cn(
            "before:bg-accent relative before:absolute before:top-1.5 before:bottom-1.5 before:left-0 before:w-0.5 before:rounded-full before:opacity-0",
            isOn(n.path) && "text-foreground before:opacity-100",
          )}
        >
          <Link href={n.path}>{t(n.label)}</Link>
        </SidebarItem>
      ))}
    </SidebarSection>
  );

  return (
    <div className="bg-background text-foreground flex h-dvh overflow-hidden">
      <Sidebar
        collapsed={collapsed}
        onCollapsedChange={onCollapsedChange}
        header={<Brand />}
        footer={<UserCard me={me} />}
        // Сангийн default `hidden md:flex`; энд lg хүртэл drawer.
        className="md:hidden lg:flex"
      >
        {nav}
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
                <span className="text-foreground hidden truncate text-sm font-semibold lg:block">{t("Тойм")}</span>
              )}
              <span className="text-foreground truncate text-sm font-semibold lg:hidden">
                {t(current?.label ?? "Тойм")}
              </span>
            </div>
          }
          actions={<Actions me={me} />}
        />
        <main
          id="main"
          tabIndex={-1}
          className="bg-background-subtle relative min-h-0 flex-1 overflow-y-auto p-4 pb-[max(1rem,env(safe-area-inset-bottom))] outline-none md:p-6 md:pb-[max(1.5rem,env(safe-area-inset-bottom))]"
        >
          <div className="mx-auto max-w-[1440px]">{children}</div>
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
          <Sidebar header={<Brand />} footer={<UserCard me={me} />} className="flex h-full w-full border-r-0 [&>button:last-child]:hidden">
            {nav}
          </Sidebar>
        </SheetContent>
      </Sheet>
    </div>
  );
}

/** Sidebar-ийн толгой: N тэмдэг + нэр (хумьсан үед зөвхөн тэмдэг). */
function Brand() {
  const { collapsed } = useSidebar();
  const { t } = useT();
  return (
    <Link
      href="/"
      aria-label={t("Платформын админ")}
      className={cn(
        "focus-visible:ring-ring focus-visible:ring-offset-background flex min-w-0 items-center gap-2 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-offset-2",
        collapsed ? "justify-center" : "w-full px-1",
      )}
    >
      <span className="bg-foreground text-background inline-flex size-8 shrink-0 items-center justify-center rounded-md text-sm font-semibold">
        N
      </span>
      {!collapsed && <span className="text-foreground truncate text-sm font-semibold">{t("Платформын админ")}</span>}
    </Link>
  );
}

function UserCard({ me }: { me: Me }) {
  const { collapsed } = useSidebar();
  const initials = me.user.name.split(" ").map((s) => s[0]).join("").slice(0, 2).toUpperCase();
  return (
    <div className={cn("flex items-center gap-2", collapsed ? "justify-center" : "px-1")}>
      <Avatar size="sm" fallback={initials} alt={me.user.name} />
      {!collapsed && (
        <div className="min-w-0 leading-tight">
          <div className="text-foreground truncate text-sm font-medium" title={me.user.name}>
            {me.user.name}
          </div>
          <div className="text-foreground-subtle truncate text-xs" title={me.user.email}>
            {me.user.email}
          </div>
        </div>
      )}
    </div>
  );
}

function Actions({ me }: { me: Me }) {
  const router = useRouter();
  const { t, locale } = useT();
  const [theme, setTheme] = useThemeMode();
  const logout = async () => {
    await api.post("/api/logout").catch(() => {});
    router.replace("/login");
  };
  return (
    <>
      <DropdownMenu>
        <Tooltip label={t("Хэл")}>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="px-2 tabular-nums">
              {locale.toUpperCase()}
            </Button>
          </DropdownMenuTrigger>
        </Tooltip>
        <DropdownMenuContent align="end" className="w-36">
          <DropdownMenuRadioGroup value={locale} onValueChange={(v) => setLocale(v as (typeof locales)[number]["code"])}>
            {locales.map((l) => (
              <DropdownMenuRadioItem key={l.code} value={l.code}>
                {l.label}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      <DropdownMenu>
        <Tooltip label={t("Өнгө")}>
          <DropdownMenuTrigger asChild>
            <IconButton aria-label={t("Өнгө")} icon={THEME_ICON[theme]} variant="ghost" size="sm" />
          </DropdownMenuTrigger>
        </Tooltip>
        <DropdownMenuContent align="end" className="w-40">
          <DropdownMenuRadioGroup value={theme} onValueChange={(v) => setTheme(v as ThemeMode)}>
            <DropdownMenuRadioItem value="light">{t("Цайвар")}</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="dark">{t("Бараан")}</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="system">{t("Системийн")}</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <button
            type="button"
            aria-label={me.user.name}
            className="hover:bg-background-muted focus-visible:ring-ring focus-visible:ring-offset-background ml-1 inline-flex size-8 items-center justify-center rounded-full outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
          >
            <Avatar size="sm" fallback={me.user.name.slice(0, 1)} alt="" />
          </button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-60">
          <DropdownMenuLabel className="font-normal">
            <span className="text-foreground block truncate text-sm font-medium">{me.user.name}</span>
            <span className="text-foreground-subtle block truncate text-xs">{me.user.email}</span>
          </DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuItem asChild>
            <Link href="/profile">
              <Icons.User className="size-4" aria-hidden />
              {t("Профайл")}
            </Link>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onSelect={logout}>
            <Icons.LogOut className="size-4" aria-hidden />
            {t("Гарах")}
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
