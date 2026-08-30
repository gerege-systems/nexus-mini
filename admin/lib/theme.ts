'use client';

import { useEffect, useState } from 'react';

export type ThemeMode = 'light' | 'dark' | 'system';

/** Сан `dark` классаар theming хийдэг (data-theme биш) — root layout-ийн
 *  pre-paint скрипт мөн үүнийг тавьдаг. */
export function applyTheme(mode: ThemeMode) {
  const dark =
    mode === 'dark' || (mode === 'system' && matchMedia('(prefers-color-scheme: dark)').matches);
  document.documentElement.classList.toggle('dark', dark);
}

export function useThemeMode(): [ThemeMode, (m: ThemeMode) => void] {
  const [mode, setMode] = useState<ThemeMode>('system');
  useEffect(() => {
    const saved = localStorage.getItem('nexus_theme') as ThemeMode | null;
    if (saved === 'light' || saved === 'dark') setMode(saved);
  }, []);
  const set = (m: ThemeMode) => {
    setMode(m);
    if (m === 'system') localStorage.removeItem('nexus_theme');
    else localStorage.setItem('nexus_theme', m);
    applyTheme(m);
  };
  return [mode, set];
}
