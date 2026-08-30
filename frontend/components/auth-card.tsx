'use client';

import type { ReactNode } from 'react';
import { Card } from '@craftzbay/ui';

/** login / signup / org-new хуудсуудын нийтлэг бүрхүүл. */
export function AuthCard({
  title,
  subtitle,
  children,
  footer,
}: {
  title: ReactNode;
  subtitle?: ReactNode;
  children: ReactNode;
  footer?: ReactNode;
}) {
  return (
    <main className="bg-background-subtle flex min-h-dvh items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <div className="mb-6 flex items-center gap-3">
          <span className="bg-accent text-accent-foreground grid size-9 shrink-0 place-items-center rounded-md font-semibold">
            N
          </span>
          <div className="min-w-0">
            <h1 className="text-foreground text-base font-semibold">{title}</h1>
            {subtitle && (
              <p className="text-foreground-muted truncate text-sm">{subtitle}</p>
            )}
          </div>
        </div>
        {children}
        {footer && (
          <p className="text-foreground-muted mt-6 text-center text-sm">{footer}</p>
        )}
      </Card>
    </main>
  );
}
