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
} from '@gerege-systems/ui';
import { PageHead, Resource } from '@/components/states';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

type Row = {
  id: string;
  email: string;
  name: string;
  platform_admin: boolean;
  created_at: string;
  tenants: number;
};

export default function UsersPage() {
  const { t } = useT();
  const res = useResource<{ users: Row[] }>('/api/admin/users');

  return (
    <>
      <PageHead
        title={t('Хэрэглэгчид')}
        description={t('Платформ дээрх бүх бүртгэлтэй хэрэглэгч')}
      />

      <Card padding="none">
        <Resource
          state={res}
          isEmpty={(d) => d.users.length === 0}
          empty={
            <EmptyState
              icon={<Icons.Users />}
              title={t('Хэрэглэгч алга')}
              description={t('Платформ дээр хараахан хэрэглэгч бүртгүүлээгүй байна.')}
            />
          }
        >
          {(d) => (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Нэр')}</TableHead>
                  <TableHead>{t('Имэйл')}</TableHead>
                  <TableHead align="right">{t('Байгууллага')}</TableHead>
                  <TableHead>{t('Бүртгүүлсэн')}</TableHead>
                  <TableHead>{t('Эрх')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {d.users.map((u) => (
                  <TableRow key={u.id}>
                    <TableCell className="font-medium">{u.name}</TableCell>
                    <TableCell className="text-foreground-muted">{u.email}</TableCell>
                    <TableCell align="right">
                      {u.tenants}
                    </TableCell>
                    <TableCell className="text-foreground-muted whitespace-nowrap">
                      {formatDate(u.created_at, { pattern: 'yyyy-MM-dd' })}
                    </TableCell>
                    <TableCell>
                      {u.platform_admin && (
                        <Badge tone="accent">{t('платформ админ')}</Badge>
                      )}
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
