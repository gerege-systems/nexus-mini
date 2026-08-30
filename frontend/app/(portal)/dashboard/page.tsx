'use client';

import type { ReactNode } from 'react';
import Link from 'next/link';
import { Card, CardTitle, Icons, Separator } from '@craftzbay/ui';
import { useShell } from '@/components/shell';
import { Icon } from '@/components/icons';
import { PageHead } from '@/components/states';
import { useT } from '@/lib/i18n';

export default function Dashboard() {
  const { me, menu } = useShell();
  const { t } = useT();
  const tenant = me.tenants.find((x) => x.id === me.tenant_id);
  const hasApps = menu.length > 0;

  const steps = [
    {
      done: hasApps,
      title: t('Апп дэлгүүрээс модуль суулгах'),
      desc: t('Байгууллагад тань хэрэгтэй модулиудыг сонгож суулгана'),
      href: '/store',
      show: !!me.permissions['core.apps.manage'],
    },
    {
      done: false,
      title: t('Гишүүдээ урих'),
      desc: t('Ажилтнуудаа нэмээд role оноогоорой'),
      href: '/members',
      show: !!me.permissions['core.members.manage'],
    },
    {
      done: false,
      title: t('Эрхийн тохиргоо'),
      desc: t('Role бүрийн permission-ийг өөрийн бүтцэд тааруулна'),
      href: '/roles',
      show: !!me.permissions['core.roles.manage'],
    },
  ].filter((s) => s.show);

  return (
    <>
      <PageHead title={tenant?.name} description={`${t('Сайн байна уу,')} ${me.user.name}`} />

      {!hasApps && steps.length > 0 && (
        <Card padding="none">
          <CardTitle className="px-5 pt-5 pb-3">{t('Эхлэхэд туслах')}</CardTitle>
          <Separator />
          <ul className="divide-border divide-y">
            {steps.map((s) => (
              <li key={s.title}>
                <Link
                  href={s.href}
                  className="hover:bg-background-muted focus-visible:ring-ring focus-visible:ring-offset-background flex items-center gap-3 px-5 py-3 outline-none transition-colors focus-visible:ring-2 focus-visible:ring-offset-2"
                >
                  {s.done ? (
                    <Icons.CheckCircle2 className="text-success-text size-5 shrink-0" aria-hidden />
                  ) : (
                    <Icons.Circle className="text-foreground-subtle size-5 shrink-0" aria-hidden />
                  )}
                  <span className="min-w-0 flex-1">
                    <span className="text-foreground block font-medium">{s.title}</span>
                    <span className="text-foreground-muted block text-sm">{s.desc}</span>
                  </span>
                  <Icons.ArrowRight className="text-foreground-subtle size-4 shrink-0" aria-hidden />
                </Link>
              </li>
            ))}
          </ul>
        </Card>
      )}

      <div className="grid gap-4 sm:grid-cols-3">
        <Stat icon={<Icon name="store" className="size-5" />} value={menu.length} label={t('Идэвхтэй апп')} />
        <Stat
          icon={<Icon name="key" className="size-5" />}
          value={Object.keys(me.permissions).length}
          label={t('Таны эрх')}
        />
        <Stat
          icon={<Icon name="building" className="size-5" />}
          value={me.tenants.length}
          label={t('Байгууллага')}
        />
      </div>

      {hasApps && (
        <Card padding="none">
          <CardTitle className="px-5 pt-5 pb-3">{t('Суусан аппууд')}</CardTitle>
          <Separator />
          <ul className="divide-border divide-y">
            {menu.map((m) => (
              <li key={m.app_id}>
                <Link
                  href={m.items[0]?.path || '#'}
                  className="hover:bg-background-muted focus-visible:ring-ring focus-visible:ring-offset-background flex items-center gap-3 px-5 py-3 outline-none transition-colors focus-visible:ring-2"
                >
                  <span className="bg-background-muted text-foreground-muted grid size-9 shrink-0 place-items-center rounded-md">
                    <Icon name={m.items[0]?.icon || 'package'} className="size-4" />
                  </span>
                  <span className="text-foreground min-w-0 flex-1 truncate">{m.name}</span>
                  <Icons.ArrowRight className="text-foreground-subtle size-4 shrink-0" aria-hidden />
                </Link>
              </li>
            ))}
          </ul>
        </Card>
      )}
    </>
  );
}

function Stat({ icon, value, label }: { icon: ReactNode; value: number; label: string }) {
  return (
    <Card className="flex items-center gap-3">
      <span className="bg-background-muted text-foreground-muted grid size-10 shrink-0 place-items-center rounded-md">
        {icon}
      </span>
      <span className="min-w-0">
        <span className="text-foreground block text-2xl font-semibold tabular-nums">{value}</span>
        <span className="text-foreground-muted block truncate text-sm">{label}</span>
      </span>
    </Card>
  );
}
