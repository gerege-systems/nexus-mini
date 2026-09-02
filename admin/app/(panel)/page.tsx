'use client';

import type { ReactNode } from 'react';
import { Card, Icons, Skeleton } from '@gerege-systems/ui';
import { Icon } from '@gerege-systems/ui/icon';
import { PageHead, Resource } from '@/components/states';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

type Overview = { tenants: number; users: number; apps: number; installations: number };

export default function OverviewPage() {
  const { t } = useT();
  const ov = useResource<Overview>('/api/admin/overview');

  return (
    <>
      <PageHead title={t('Тойм')} description={t('Платформын ерөнхий үзүүлэлтүүд')} />

      <Resource
        state={ov}
        skeleton={
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            {Array.from({ length: 4 }).map((_, i) => (
              <Skeleton key={i} className="h-24" />
            ))}
          </div>
        }
      >
        {(d) => (
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
            <Stat icon={<Icon name="building-2" />} value={d.tenants} label={t('Байгууллага')} />
            <Stat icon={<Icons.Users />} value={d.users} label={t('Хэрэглэгч')} />
            <Stat icon={<Icons.Package />} value={d.apps} label={t('Бэлэн апп')} />
            <Stat icon={<Icons.FileText />} value={d.installations} label={t('Суулгалт')} />
          </div>
        )}
      </Resource>
    </>
  );
}

function Stat({ icon, value, label }: { icon: ReactNode; value: number; label: string }) {
  return (
    <Card className="flex items-center gap-3">
      <span className="bg-background-muted text-foreground-muted grid size-10 shrink-0 place-items-center rounded-md [&_svg]:size-5">
        {icon}
      </span>
      <span className="min-w-0">
        <span className="text-foreground block text-2xl font-semibold tabular-nums">{value}</span>
        <span className="text-foreground-muted block truncate text-sm">{label}</span>
      </span>
    </Card>
  );
}
