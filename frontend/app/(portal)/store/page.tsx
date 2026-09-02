'use client';

import { useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  EmptyState,
  Icons,
  IconButton,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tooltip,
  formatDate,
  toast,
} from '@gerege-systems/ui';
import { PageHead, Resource } from '@/components/states';
import { api, ApiError, type StoreApp } from '@/lib/api';
import { useShell } from '@/components/shell';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

type Hist = {
  releases: { version: string; seen_at: string }[];
  events: {
    action: string;
    from_version: string;
    to_version: string;
    user_name: string;
    at: string;
  }[];
};

const ACTION_LABEL: Record<string, string> = {
  install: 'Суулгасан',
  enable: 'Асаасан',
  disable: 'Унтраасан',
  upgrade: 'Шинэчлэгдсэн',
};

export default function StorePage() {
  const { me, refresh } = useShell();
  const { t } = useT();
  const res = useResource<{ apps: StoreApp[] }>('/api/store/apps');
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState('');
  const [histFor, setHistFor] = useState<StoreApp | null>(null);
  const canManage = !!me.permissions['core.apps.manage'];

  const act = async (app: StoreApp, action: 'install' | 'enable' | 'disable') => {
    setErr('');
    setBusy(app.id);
    try {
      await api.post(`/api/store/apps/${app.id}/${action}`);
      toast({
        title:
          action === 'install'
            ? `${app.name} ${t('суулгагдлаа')}`
            : action === 'enable'
              ? t('Асаалаа')
              : t('Унтраалаа'),
        variant: 'success',
      });
      res.reload();
      refresh(); // цэс шинэчлэгдэнэ
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t('Алдаа гарлаа'));
    } finally {
      setBusy('');
    }
  };

  return (
    <>
      <PageHead
        title={t('Апп дэлгүүр')}
        description={t('Байгууллагадаа хэрэгтэй модулиудыг суулгана')}
      />

      {err && (
        <Alert variant="danger" className="mb-4" dismissible onDismiss={() => setErr('')}>
          {err}
        </Alert>
      )}

      <Resource
        state={res}
        skeleton={
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Card key={i} className="h-44" />
            ))}
          </div>
        }
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
          <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
            {d.apps.map((a) => (
              <Card key={a.id} className="flex flex-col gap-3">
                <div className="flex items-start gap-3">
                  <span className="bg-background-muted text-foreground-muted grid size-10 shrink-0 place-items-center rounded-md">
                    <Icons.Package className="size-5" aria-hidden />
                  </span>
                  <div className="min-w-0 flex-1">
                    <span className="text-foreground block truncate font-medium">{a.name}</span>
                    <span className="text-foreground-subtle block truncate text-xs">
                      {a.publisher}
                    </span>
                  </div>
                  <Tooltip label={t('Хувилбарын түүх')}>
                    <IconButton
                      aria-label={`${a.name} — ${t('Хувилбарын түүх')}`}
                      icon={<Icons.FileText />}
                      variant="ghost"
                      size="sm"
                      onClick={() => setHistFor(a)}
                    />
                  </Tooltip>
                </div>

                <p className="text-foreground-muted flex-1 text-sm">{a.description}</p>

                <div className="flex flex-wrap items-center gap-2">
                  {a.status === 'enabled' ? (
                    <>
                      <Badge tone="success">{t('Суусан')}</Badge>
                      {canManage && (
                        <Button
                          variant="secondary"
                          size="sm"
                          loading={busy === a.id}
                          onClick={() => act(a, 'disable')}
                        >
                          {t('Унтраах')}
                        </Button>
                      )}
                    </>
                  ) : a.status === 'disabled' ? (
                    <>
                      <Badge tone="warning">{t('Унтраасан')}</Badge>
                      {canManage && (
                        <Button size="sm" loading={busy === a.id} onClick={() => act(a, 'enable')}>
                          {t('Асаах')}
                        </Button>
                      )}
                    </>
                  ) : a.compiled ? (
                    canManage ? (
                      <Button
                        size="sm"
                        leadingIcon={<Icons.Download />}
                        loading={busy === a.id}
                        onClick={() => act(a, 'install')}
                      >
                        {t('Суулгах')}
                      </Button>
                    ) : (
                      <Badge tone="neutral">{t('Суулгаагүй')}</Badge>
                    )
                  ) : (
                    <span className="flex flex-wrap items-center gap-1.5">
                      <Badge tone="neutral">{t('Бинарид ороогүй')}</Badge>
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

      {histFor && <HistoryDialog app={histFor} onClose={() => setHistFor(null)} />}
    </>
  );
}

function HistoryDialog({ app, onClose }: { app: StoreApp; onClose: () => void }) {
  const { t } = useT();
  const res = useResource<Hist>(`/api/store/apps/${app.id}/history`);

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>
            {app.name} — {t('Хувилбарын түүх')}
          </DialogTitle>
        </DialogHeader>

        <Resource
          state={res}
          skeleton={
            <div className="flex justify-center py-8">
              <Spinner label={t('Уншиж байна…')} />
            </div>
          }
        >
          {(d) => (
            <div className="space-y-4">
              <section>
                <h3 className="text-foreground mb-2 text-sm font-medium">
                  {t('Нийтлэгчийн хувилбарууд')}
                </h3>
                {d.releases.length === 0 ? (
                  <p className="text-foreground-subtle text-sm">—</p>
                ) : (
                  <ul className="text-foreground-muted space-y-1 text-sm">
                    {d.releases.map((r) => (
                      <li key={r.version} className="flex items-center gap-2">
                        <code className="font-mono text-xs">v{r.version}</code>
                        <span className="text-foreground-subtle">
                          {formatDate(r.seen_at, { pattern: 'yyyy-MM-dd' })}
                        </span>
                        {r.version === app.installed_version && (
                          <Badge tone="success">{t('суусан')}</Badge>
                        )}
                      </li>
                    ))}
                  </ul>
                )}
              </section>

              <section>
                <h3 className="text-foreground mb-2 text-sm font-medium">
                  {t('Энэ байгууллагад')}
                </h3>
                {d.events.length === 0 ? (
                  <p className="text-foreground-subtle text-sm">{t('Үйл явдал байхгүй')}</p>
                ) : (
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>{t('Үйлдэл')}</TableHead>
                        <TableHead>{t('Хувилбар')}</TableHead>
                        <TableHead>{t('Хэн')}</TableHead>
                        <TableHead>{t('Хэзээ')}</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {d.events.map((e, i) => (
                        <TableRow key={i}>
                          <TableCell>{t(ACTION_LABEL[e.action] ?? e.action)}</TableCell>
                          <TableCell>
                            <code className="font-mono text-xs">
                              {e.from_version ? `${e.from_version} → ` : ''}
                              {e.to_version || '—'}
                            </code>
                          </TableCell>
                          <TableCell className="text-foreground-muted">
                            {e.user_name || t('систем')}
                          </TableCell>
                          <TableCell className="text-foreground-muted whitespace-nowrap">
                            {formatDate(e.at)}
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                )}
              </section>
            </div>
          )}
        </Resource>

        <DialogFooter>
          <Button variant="secondary" onClick={onClose}>
            {t('Хаах')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
