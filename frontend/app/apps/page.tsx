'use client';

import Link from 'next/link';
import { Badge, Button, Card, EmptyState, Icons } from '@craftzbay/ui';
import { MktHeader, MktFooter } from '@/components/mkt';
import { Resource } from '@/components/states';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

type CatalogApp = {
  id: string;
  short_id: string;
  name: string;
  version: string;
  description: string;
  publisher: string;
  compiled: boolean;
};

// Нийтийн апп дэлгүүрийн хуудас — нэвтрэлт шаардахгүй каталог.
export default function PublicAppsPage() {
  const { t } = useT();
  const res = useResource<{ apps: CatalogApp[] }>('/api/catalog');

  const steps = [
    {
      n: 1,
      title: t('Дэлгүүрээс сонгоно'),
      body: t('Каталогоос аппаа сонгоод «Суулгах» — хамаарлуудыг нь платформ өөрөө цэгцэлнэ.'),
    },
    {
      n: 2,
      title: t('Эрх автоматаар'),
      body: t(
        'Аппын permission-ууд role-уудад тунхагласан ёсоороо оноогдоно; админ дараа нь чөлөөтэй өөрчилнө.',
      ),
    },
    {
      n: 3,
      title: t('Цэс гарч ирнэ'),
      body: t(
        'Эрхтэй хэрэглэгчид л аппын цэсийг харна. Rail дээр аппын icon нэмэгдэж, өөрийн цэстэйгээ ирнэ.',
      ),
    },
  ];

  return (
    <div className="flex min-h-dvh flex-col">
      <MktHeader />

      <main className="mx-auto w-full max-w-6xl flex-1 px-4 py-12 md:px-6 md:py-16">
        <h1 className="text-foreground text-3xl font-semibold text-balance">{t('Апп дэлгүүр')}</h1>
        <p className="text-foreground-muted mt-3 max-w-3xl text-pretty">
          {t(
            'Байгууллага бүр өөрт хэрэгтэй модулиа л суулгана. Суусан апп нь permission-уудаа role-уудад тунхагласан ёсоор оноож, цэсээ эрхтэй хүнд л харуулна; унтраавал бүгд эргэж алга болно.',
          )}
        </p>

        <div className="mt-10 grid gap-4 md:grid-cols-3">
          {steps.map((s) => (
            <Card key={s.n}>
              <span className="bg-accent text-accent-foreground grid size-7 place-items-center rounded-md text-sm font-semibold">
                {s.n}
              </span>
              <h2 className="text-foreground mt-3 font-semibold">{s.title}</h2>
              <p className="text-foreground-muted mt-2 text-sm">{s.body}</p>
            </Card>
          ))}
        </div>

        <h2 className="text-foreground mt-12 mb-4 text-xl font-semibold">
          {t('Одоо байгаа аппууд')}
        </h2>

        <Resource
          state={res}
          skeleton={
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {Array.from({ length: 3 }).map((_, i) => (
                <Card key={i} className="h-40" />
              ))}
            </div>
          }
          isEmpty={(d) => d.apps.length === 0}
          empty={<EmptyState icon={<Icons.Package />} title={t('Каталог хоосон байна.')} />}
        >
          {(d) => (
            <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
              {d.apps.map((a) => (
                <Card key={a.id} className="flex flex-col gap-3">
                  <div className="flex items-start gap-3">
                    <span className="bg-background-muted text-foreground-muted grid size-10 shrink-0 place-items-center rounded-md">
                      <Icons.Package className="size-5" aria-hidden />
                    </span>
                    <div className="min-w-0">
                      <span className="text-foreground block truncate font-medium">{a.name}</span>
                      <span className="text-foreground-subtle block truncate text-xs">
                        {a.publisher || '—'}
                      </span>
                    </div>
                  </div>
                  <p className="text-foreground-muted flex-1 text-sm">
                    {a.description || t('Тайлбар оруулаагүй.')}
                  </p>
                  <div className="flex flex-wrap items-center gap-2">
                    {a.compiled ? (
                      <Badge tone="success">{t('Бэлэн')}</Badge>
                    ) : (
                      <span className="flex flex-wrap items-center gap-1.5">
                        <Badge tone="neutral">{t('Registry-д бүртгэлтэй')}</Badge>
                        <code className="text-foreground-subtle font-mono text-xs">
                          nexus add {a.short_id}
                        </code>
                      </span>
                    )}
                    <span className="text-foreground-subtle ml-auto text-xs tabular-nums">
                      v{a.version}
                    </span>
                  </div>
                </Card>
              ))}
            </div>
          )}
        </Resource>

        <Card className="mt-10 flex flex-wrap items-center gap-4">
          <div className="min-w-[15rem] flex-1">
            <p className="text-foreground font-medium">
              {t('Өөрийн модулиа энд гаргамаар байна уу?')}
            </p>
            <p className="text-foreground-muted text-sm">
              {t('Гарын авлагыг дагаад модулиа бичээд каталогт PR илгээгээрэй.')}
            </p>
          </div>
          <Button asChild>
            <Link href="/developers">{t('Модуль хөгжүүлэх заавар')}</Link>
          </Button>
        </Card>
      </main>

      <MktFooter />
    </div>
  );
}
