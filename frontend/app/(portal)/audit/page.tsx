'use client';

import { useState } from 'react';
import {
  Alert,
  Badge,
  Button,
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
import { api, type AuditEntry } from '@/lib/api';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

export default function AuditPage() {
  const { t } = useT();
  const res = useResource<{ entries: AuditEntry[] }>('/api/audit?limit=100');
  const [verify, setVerify] = useState<{ intact: boolean; broken_at: number | null } | null>(null);
  const [busy, setBusy] = useState(false);

  const runVerify = async () => {
    setBusy(true);
    try {
      setVerify(await api.get('/api/audit/verify'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <PageHead
        title={t('Audit лог')}
        description={t('Append-only, hash гинжтэй үйлдлийн бүртгэл')}
        actions={
          <Button
            variant="secondary"
            leadingIcon={<Icons.Lock />}
            loading={busy}
            onClick={runVerify}
          >
            {t('Гинж шалгах')}
          </Button>
        }
      />

      {verify && (
        <Alert variant={verify.intact ? 'success' : 'danger'} className="mb-4">
          {verify.intact
            ? t('Гинж бүрэн — бүртгэлд гар хүрээгүй')
            : `#${verify.broken_at} ${t('дээр тасарсан!')}`}
        </Alert>
      )}

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
                  <TableHead>{t('Үйлдэл')}</TableHead>
                  <TableHead>{t('Объект')}</TableHead>
                  <TableHead>{t('Хэн')}</TableHead>
                  <TableHead>{t('Хэзээ')}</TableHead>
                  <TableHead>Hash</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {d.entries.map((e) => (
                  <TableRow key={e.id}>
                    <TableCell align="right" className="text-foreground-subtle">
                      {e.id}
                    </TableCell>
                    <TableCell>
                      <Badge tone="accent">{e.action}</Badge>
                    </TableCell>
                    <TableCell>{e.object || '—'}</TableCell>
                    <TableCell>{e.user_name || t('систем')}</TableCell>
                    <TableCell className="text-foreground-muted whitespace-nowrap">
                      {formatDate(e.occurred_at)}
                    </TableCell>
                    <TableCell>
                      <code className="text-foreground-subtle font-mono text-xs">
                        {e.hash.slice(0, 12)}…
                      </code>
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
