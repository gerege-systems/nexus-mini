'use client';

import {
  Badge,
  Card,
  EmptyState,
  Icons,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@craftzbay/ui';
import { PageHead, Resource } from '@/components/states';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

type Row = {
  id: string;
  short_id: string;
  name: string;
  version: string;
  compiled: boolean;
  publisher: string;
  installs: number;
};

export default function AppsPage() {
  const { t } = useT();
  const res = useResource<{ apps: Row[] }>('/api/admin/apps');

  return (
    <>
      <PageHead title={t('Каталог')} description={t('App store-ийн бүх апп, суулгалтын тоо')} />

      <Card padding="none">
        <Resource
          state={res}
          isEmpty={(d) => d.apps.length === 0}
          empty={
            <EmptyState
              icon={<Icons.Package />}
              title={t('Апп алга')}
              description={t('Каталогт хараахан апп нэмэгдээгүй байна.')}
            />
          }
        >
          {(d) => (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Апп')}</TableHead>
                  <TableHead>ID</TableHead>
                  <TableHead>{t('Хувилбар')}</TableHead>
                  <TableHead>{t('Бинарид')}</TableHead>
                  <TableHead align="right">{t('Суулгалт')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {d.apps.map((a) => (
                  <TableRow key={a.id}>
                    <TableCell>
                      <span className="text-foreground block font-medium">{a.name}</span>
                      <span className="text-foreground-subtle block text-xs">{a.publisher}</span>
                    </TableCell>
                    <TableCell>
                      <code className="text-foreground-muted font-mono text-xs">{a.id}</code>
                    </TableCell>
                    <TableCell className="whitespace-nowrap">v{a.version}</TableCell>
                    <TableCell>
                      {a.compiled ? (
                        <Badge tone="success">{t('Тийм')}</Badge>
                      ) : (
                        <Badge tone="neutral">{t('Үгүй')}</Badge>
                      )}
                    </TableCell>
                    <TableCell align="right">{a.installs}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </Resource>
      </Card>
    </>
  );
}
