'use client';

// Landing-ийн хуудсуудын (нүүр / апп дэлгүүр / хөгжүүлэгч) нийтлэг chrome.

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';
import {
  Button,
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
  Icons,
  IconButton,
  Tooltip,
  cn,
} from '@craftzbay/ui';
import { locales, setLocale, useT } from '@/lib/i18n';
import { useThemeMode } from '@/lib/theme';

const REPO = 'https://github.com/gerege-systems/nexus-mini';

function GitHubGlyph() {
  return (
    <svg viewBox="0 0 16 16" fill="currentColor" className="size-4" aria-hidden>
      <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27s1.36.09 2 .27c1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8Z" />
    </svg>
  );
}

function LangMenu() {
  const { locale } = useT();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="px-2">
          {locale.toUpperCase()}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-32">
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
  );
}

function ThemeToggle() {
  const [theme, setTheme] = useThemeMode();
  const { t } = useT();
  // Системийн dark эсэхийг render-д биш mount-ын дараа уншина — үгүй бол
  // сервер Moon, клиент Sun render хийж hydration зөрдөг (React #418).
  const [systemDark, setSystemDark] = useState(false);
  useEffect(() => {
    const mq = matchMedia('(prefers-color-scheme: dark)');
    setSystemDark(mq.matches);
    const onChange = (e: MediaQueryListEvent) => setSystemDark(e.matches);
    mq.addEventListener('change', onChange);
    return () => mq.removeEventListener('change', onChange);
  }, []);
  const dark = theme === 'dark' || (theme === 'system' && systemDark);
  return (
    <Tooltip label={t('Загвар')}>
      <IconButton
        aria-label={t('Загвар')}
        variant="ghost"
        size="sm"
        icon={dark ? <Icons.Sun /> : <Icons.Moon />}
        onClick={() => setTheme(dark ? 'light' : 'dark')}
      />
    </Tooltip>
  );
}

export function MktHeader() {
  const pathname = usePathname();
  const { t } = useT();
  const nav = [
    { href: '/', label: t('Нүүр') },
    { href: '/apps', label: t('Апп дэлгүүр') },
    { href: '/developers', label: t('Хөгжүүлэгчид') },
  ];
  const on = (p: string) => (p === '/' ? pathname === '/' : pathname.startsWith(p));

  return (
    <header className="border-border bg-background sticky top-0 z-[var(--z-sticky)] border-b">
      <div className="mx-auto flex h-14 max-w-6xl items-center gap-4 px-4 md:px-6">
        <Link
          href="/"
          className="focus-visible:ring-ring focus-visible:ring-offset-background flex shrink-0 items-center gap-2 rounded-md outline-none focus-visible:ring-2 focus-visible:ring-offset-2"
        >
          <span className="bg-accent text-accent-foreground grid size-7 place-items-center rounded-md text-sm font-semibold">
            N
          </span>
          <span className="text-foreground font-semibold">nexus-mini</span>
        </Link>

        <nav className="hidden items-center gap-1 sm:flex" aria-label={t('Цэс')}>
          {nav.map((n) => (
            <Link
              key={n.href}
              href={n.href}
              aria-current={on(n.href) ? 'page' : undefined}
              className={cn(
                'rounded-md px-3 py-1.5 text-sm outline-none transition-colors',
                'focus-visible:ring-ring focus-visible:ring-offset-background focus-visible:ring-2 focus-visible:ring-offset-2',
                on(n.href)
                  ? 'text-foreground font-medium'
                  : 'text-foreground-muted hover:bg-background-muted hover:text-foreground',
              )}
            >
              {n.label}
            </Link>
          ))}
        </nav>

        <div className="ml-auto flex shrink-0 items-center gap-1">
          <LangMenu />
          <Tooltip label="GitHub">
            <Button variant="ghost" size="icon" aria-label="GitHub" asChild>
              <a href={REPO} target="_blank" rel="noreferrer">
                <GitHubGlyph />
              </a>
            </Button>
          </Tooltip>
          <ThemeToggle />
          <span className="bg-border mx-1 h-5 w-px" aria-hidden />
          <Button size="sm" asChild>
            <Link href="/login">{t('Нэвтрэх')}</Link>
          </Button>
        </div>
      </div>
    </header>
  );
}

export function MktFooter() {
  return (
    <footer className="border-border mt-16 border-t">
      <div className="text-foreground-muted mx-auto flex max-w-6xl items-center gap-3 px-4 py-6 text-sm md:px-6">
        <span className="bg-accent text-accent-foreground grid size-6 shrink-0 place-items-center rounded-md text-xs font-semibold">
          N
        </span>
        <span>
          nexus-mini · Apache 2.0 ·{' '}
          <a href={REPO} className="text-accent hover:underline">
            gerege-systems/nexus-mini
          </a>
        </span>
      </div>
    </footer>
  );
}
