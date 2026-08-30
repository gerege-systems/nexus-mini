'use client';

// Хуудсуудын давтагддаг хэсэг: гарчиг + ачаалалт/алдаа/хоосон төлөв.
// design-research: төлөв бүр (loading / empty / error) ил байх ёстой —
// чимээгүй хоосон хүснэгт үлдээхгүй.

import type { ReactNode } from 'react';
import { Button, EmptyState, ErrorState, Skeleton } from '@craftzbay/ui';
import { useT } from '@/lib/i18n';
import type { ResourceState } from '@/lib/use-resource';

export function PageHead({
  title,
  description,
  actions,
}: {
  title: ReactNode;
  description?: ReactNode;
  actions?: ReactNode;
}) {
  return (
    <div className="mb-4 flex flex-wrap items-start justify-between gap-3 md:mb-6">
      <div className="min-w-0">
        <h1 className="text-foreground text-xl font-semibold">{title}</h1>
        {description && <p className="text-foreground-muted mt-1 text-sm">{description}</p>}
      </div>
      {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
    </div>
  );
}

/** Хүснэгтийн ачаалалтын хувилбар — layout үсрэхээс сэргийлж мөр бүрийг барина. */
export function TableSkeleton({ rows = 6, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div className="space-y-2" aria-hidden>
      {Array.from({ length: rows }).map((_, r) => (
        <div key={r} className="flex gap-3">
          {Array.from({ length: cols }).map((_, c) => (
            <Skeleton key={c} className="h-9 flex-1" />
          ))}
        </div>
      ))}
    </div>
  );
}

/**
 * Нэг GET-ийн гурван төлөвийг зурна. `isEmpty` өгвөл хоосон төлөвийг
 * `empty` slot-оор харуулна.
 */
export function Resource<T>({
  state,
  children,
  skeleton,
  isEmpty,
  empty,
}: {
  state: ResourceState<T> & { reload: () => void };
  children: (data: T) => ReactNode;
  skeleton?: ReactNode;
  isEmpty?: (data: T) => boolean;
  empty?: ReactNode;
}) {
  const { t } = useT();

  if (state.status === 'loading') return <>{skeleton ?? <TableSkeleton />}</>;

  if (state.status === 'error') {
    return (
      <ErrorState
        variant="generic"
        description={state.error.message}
        action={
          <Button variant="secondary" onClick={state.reload}>
            {t('Дахин оролдох')}
          </Button>
        }
      />
    );
  }

  if (isEmpty?.(state.data)) {
    return <>{empty ?? <EmptyState title={t('Бичлэг алга')} />}</>;
  }

  return <>{children(state.data)}</>;
}
