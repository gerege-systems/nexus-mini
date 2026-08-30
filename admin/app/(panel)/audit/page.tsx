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
  formatDate,
} from '@craftzbay/ui';
import { PageHead, Resource } from '@/components/states';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

type Row = {
  id: number;
  tenant: string;
  user_name: string;
  action: string;
  object: string;
  occurred_at: string;
};

export default function AuditPage() {
  const { t } = useT();
  const res = useResource<{ entries: Row[] }>('/api/admin/audit');

  return (
    <>
      <PageHead title="Audit" description={t('Бүх байгууллагын сүүлийн үйлдлүүд')} />

      <Card padding="none">
        <Resource
          state={res}
          isEmpty={(d) => d.entries.length === 0}
          empty={
            <EmptyState
              icon={<Icons.FileText />}
              title={t('Бичлэг алга')}
              description={t('Хараахан бүртгэгдсэн үйлдэл байхгүй байна.')}
            />
          }
        >
          {(d) => (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead align="right">#</TableHead>
                  <TableHead>{t('Байгууллага')}</TableHead>
                  <TableHead>{t('Үйлдэл')}</TableHead>
                  <TableHead>{t('Объект')}</TableHead>
                  <TableHead>{t('Хэн')}</TableHead>
                  <TableHead>{t('Хэзээ')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {d.entries.map((e) => (
                  <TableRow key={e.id}>
                    <TableCell align="right" className="text-foreground-subtle">
                      {e.id}
                    </TableCell>
                    <TableCell>
                      <code className="text-foreground-muted font-mono text-xs">{e.tenant}</code>
                    </TableCell>
                    <TableCell>
                      <Badge tone="accent">{e.action}</Badge>
                    </TableCell>
                    <TableCell>{e.object || '—'}</TableCell>
                    <TableCell>{e.user_name || t('систем')}</TableCell>
                    <TableCell className="text-foreground-muted whitespace-nowrap">
                      {formatDate(e.occurred_at)}
                    </TableCell>
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
