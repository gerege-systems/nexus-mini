'use client';

import { useEffect, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  ConfirmationDialog,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  EmptyState,
  Icons,
  Input,
  Separator,
  Spinner,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  formatDate,
  toast,
} from '@gerege-systems/ui';
import { Icon } from '@gerege-systems/ui/icon';
import { PageHead, Resource } from '@/components/states';
import { api, ApiError } from '@/lib/api';
import { useT } from '@/lib/i18n';
import { useResource } from '@/lib/use-resource';

type Row = {
  id: string;
  slug: string;
  name: string;
  created_at: string;
  suspended: boolean;
  reason: string;
  read_only: boolean;
  deletion_at: string | null;
  members: number;
  apps: number;
};
type Member = { id: string; name: string; email: string; platform_admin: boolean; roles: string[] };

const msg = (e: unknown, fallback: string) => (e instanceof ApiError ? e.message : fallback);

export default function TenantsPage() {
  const { t } = useT();
  const res = useResource<{ tenants: Row[] }>('/api/admin/tenants');
  const [membersOf, setMembersOf] = useState<Row | null>(null);
  const [stateOf, setStateOf] = useState<Row | null>(null);

  return (
    <>
      <PageHead title={t('Байгууллагууд')} description={t('Платформ дээрх бүх байгууллага')} />

      <Card padding="none">
        <Resource
          state={res}
          isEmpty={(d) => d.tenants.length === 0}
          empty={
            <EmptyState
              icon={<Icon name="building-2" />}
              title={t('Байгууллага алга')}
              description={t('Платформ дээр хараахан байгууллага үүсээгүй байна.')}
            />
          }
        >
          {(d) => (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Нэр')}</TableHead>
                  <TableHead>Slug</TableHead>
                  <TableHead align="right">{t('Гишүүд')}</TableHead>
                  <TableHead align="right">{t('Аппууд')}</TableHead>
                  <TableHead>{t('Үүссэн')}</TableHead>
                  <TableHead align="right" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {d.tenants.map((r) => (
                  <TableRow key={r.id}>
                    <TableCell>
                      <span className="flex flex-wrap items-center gap-2">
                        <span className="text-foreground font-medium">{r.name}</span>
                        {r.suspended && <Badge tone="danger">{t('түдгэлзүүлсэн')}</Badge>}
                        {r.read_only && !r.suspended && (
                          <Badge tone="warning">{t('зөвхөн унших')}</Badge>
                        )}
                        {r.deletion_at && (
                          <Badge tone="danger" variant="outline">
                            {t('устгал')}: {formatDate(r.deletion_at, { pattern: 'yyyy-MM-dd' })}
                          </Badge>
                        )}
                      </span>
                    </TableCell>
                    <TableCell>
                      <code className="text-foreground-muted font-mono text-xs">{r.slug}</code>
                    </TableCell>
                    <TableCell align="right">{r.members}</TableCell>
                    <TableCell align="right">{r.apps}</TableCell>
                    <TableCell className="text-foreground-muted whitespace-nowrap">
                      {formatDate(r.created_at, { pattern: 'yyyy-MM-dd' })}
                    </TableCell>
                    <TableCell align="right">
                      <span className="flex justify-end gap-1 whitespace-nowrap">
                        <Button
                          variant="ghost"
                          size="sm"
                          leadingIcon={<Icons.Users />}
                          onClick={() => setMembersOf(r)}
                        >
                          {t('Гишүүд')}
                        </Button>
                        <Button
                          variant="ghost"
                          size="sm"
                          leadingIcon={<Icons.Lock />}
                          onClick={() => setStateOf(r)}
                        >
                          {t('Төлөв')}
                        </Button>
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </Resource>
      </Card>

      {stateOf && (
        <StateDialog
          row={stateOf}
          onClose={() => setStateOf(null)}
          onSaved={() => {
            setStateOf(null);
            res.reload();
          }}
        />
      )}

      {membersOf && <MembersDialog row={membersOf} onClose={() => setMembersOf(null)} />}
    </>
  );
}

/* -------------------------------------------------------------------------- */
/*  Төлөв — түдгэлзүүлэх / зөвхөн унших / устгалын товлолт                     */
/* -------------------------------------------------------------------------- */

function StateDialog({
  row,
  onClose,
  onSaved,
}: {
  row: Row;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t } = useT();
  const [suspended, setSuspended] = useState(row.suspended);
  const [reason, setReason] = useState(row.reason);
  const [readOnly, setReadOnly] = useState(row.read_only);
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);
  const [confirmDeletion, setConfirmDeletion] = useState<'schedule' | 'cancel' | null>(null);

  const save = async () => {
    setErr('');
    setBusy(true);
    try {
      await api.put(`/api/admin/tenants/${row.id}/state`, {
        suspended,
        reason,
        read_only: readOnly,
      });
      toast({ title: t('Төлөв хадгалагдлаа'), variant: 'success' });
      onSaved();
    } catch (e) {
      setErr(msg(e, t('Алдаа гарлаа')));
      setBusy(false);
    }
  };

  const applyDeletion = async (schedule: boolean) => {
    await api.post(`/api/admin/tenants/${row.id}/delete${schedule ? '' : '/cancel'}`);
    toast({
      title: schedule ? t('Устгалд товлогдлоо (30 хоног)') : t('Устгал цуцлагдлаа'),
      variant: schedule ? 'warning' : 'success',
    });
    onSaved();
  };

  return (
    <>
      <Dialog open onOpenChange={(o) => !o && onClose()}>
        <DialogContent size="md">
          <DialogHeader>
            <DialogTitle>
              {row.name} — {t('Төлөв')}
            </DialogTitle>
          </DialogHeader>

          {err && <Alert variant="danger">{err}</Alert>}

          <div className="space-y-4">
            <Switch
              checked={suspended}
              onCheckedChange={setSuspended}
              label={t('Түдгэлзүүлэх')}
              description={t('Түдгэлзүүлэх — гишүүд өгөгдөлдөө хандаж чадахгүй')}
            />

            {suspended && (
              <Input
                label={t('Шалтгаан (гишүүдэд харагдана)')}
                value={reason}
                maxLength={300}
                onChange={(e) => setReason(e.target.value)}
              />
            )}

            <Switch
              checked={readOnly}
              onCheckedChange={setReadOnly}
              label={t('Зөвхөн унших')}
              description={t('Зөвхөн унших — бичих хүсэлт 503 (засвар, төлбөр)')}
            />

            <Separator />

            <div className="space-y-2">
              <span className="text-foreground block text-sm font-medium">
                {t('Устгал (30 хоногийн хүлээлт)')}
              </span>
              {row.deletion_at ? (
                <div className="flex flex-wrap items-center gap-3">
                  <span className="text-danger-text text-sm">{formatDate(row.deletion_at)}</span>
                  <Button variant="secondary" size="sm" onClick={() => setConfirmDeletion('cancel')}>
                    {t('Устгалыг цуцлах')}
                  </Button>
                </div>
              ) : (
                <Button
                  variant="destructive"
                  size="sm"
                  leadingIcon={<Icons.Trash2 />}
                  onClick={() => setConfirmDeletion('schedule')}
                >
                  {t('Устгалд товлох')}
                </Button>
              )}
            </div>
          </div>

          <DialogFooter>
            <Button variant="secondary" onClick={onClose}>
              {t('Болих')}
            </Button>
            <Button onClick={save} loading={busy}>
              {t('Хадгалах')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmationDialog
        open={confirmDeletion !== null}
        onOpenChange={(o) => !o && setConfirmDeletion(null)}
        title={row.name}
        description={
          confirmDeletion === 'schedule'
            ? t(
                '30 хоногийн дараа бүрмөсөн устгахаар товлох уу? Гишүүд тэр дороо хандах боломжгүй болно; хүртэл нь буцааж болно.',
              )
            : t('устгалыг цуцлах уу?')
        }
        confirmLabel={
          confirmDeletion === 'schedule' ? t('Устгалд товлох') : t('Устгалыг цуцлах')
        }
        confirmVariant={confirmDeletion === 'schedule' ? 'destructive' : 'primary'}
        onConfirm={() => applyDeletion(confirmDeletion === 'schedule')}
      />
    </>
  );
}

/* -------------------------------------------------------------------------- */
/*  Гишүүд + нэрийн өмнөөс нэвтрэх (handover)                                  */
/* -------------------------------------------------------------------------- */

function MembersDialog({ row, onClose }: { row: Row; onClose: () => void }) {
  const { t } = useT();
  const [members, setMembers] = useState<Member[] | null>(null);
  const [err, setErr] = useState('');
  const [target, setTarget] = useState<Member | null>(null);

  useEffect(() => {
    let alive = true;
    api
      .get<{ members: Member[] }>(`/api/admin/tenants/${row.id}/members`)
      .then((r) => alive && setMembers(r.members))
      .catch((e) => alive && setErr(msg(e, t('Алдаа гарлаа'))));
    return () => {
      alive = false;
    };
  }, [row.id, t]);

  // Нэг удаагийн handover-ыг шинэ таб-д. Token нь URL-д биш POST биед —
  // access log/түүхэнд үлдэхгүй.
  const impersonate = async (m: Member) => {
    const r = await api.post<{ url: string; token: string }>('/api/admin/impersonate', {
      tenant_id: row.id,
      user_id: m.id,
    });
    const f = document.createElement('form');
    f.method = 'POST';
    f.action = r.url;
    f.target = '_blank';
    f.rel = 'noopener';
    const i = document.createElement('input');
    i.type = 'hidden';
    i.name = 'token';
    i.value = r.token;
    f.appendChild(i);
    document.body.appendChild(f);
    f.submit();
    f.remove();
    toast({ title: t('Handover холбоос нээгдлээ (60 секунд хүчинтэй)'), variant: 'info' });
  };

  return (
    <>
      <Dialog open onOpenChange={(o) => !o && onClose()}>
        <DialogContent size="lg">
          <DialogHeader>
            <DialogTitle>
              {row.name} — {t('Гишүүд')}
            </DialogTitle>
          </DialogHeader>

          {err && <Alert variant="danger">{err}</Alert>}

          {members === null && !err ? (
            <div className="flex justify-center py-8">
              <Spinner label={t('Уншиж байна…')} />
            </div>
          ) : members && members.length === 0 ? (
            <EmptyState icon={<Icons.Users />} title={t('Гишүүн байхгүй')} />
          ) : (
            members && (
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>{t('Нэр')}</TableHead>
                    <TableHead>{t('Имэйл')}</TableHead>
                    <TableHead>Role</TableHead>
                    <TableHead align="right" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {members.map((m) => (
                    <TableRow key={m.id}>
                      <TableCell>{m.name}</TableCell>
                      <TableCell className="text-foreground-muted">{m.email}</TableCell>
                      <TableCell>{m.roles.join(', ') || '—'}</TableCell>
                      <TableCell align="right">
                        {!m.platform_admin && (
                          <Button
                            variant="ghost"
                            size="sm"
                            leadingIcon={<Icons.LogOut className="rotate-180" />}
                            onClick={() => setTarget(m)}
                          >
                            {t('Нэрийн өмнөөс нэвтрэх')}
                          </Button>
                        )}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            )
          )}

          <DialogFooter>
            <Button variant="secondary" onClick={onClose}>
              {t('Хаах')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <ConfirmationDialog
        open={target !== null}
        onOpenChange={(o) => !o && setTarget(null)}
        title={target ? `${target.name} (${target.email})` : ''}
        description={t(
          'нэрийн өмнөөс нэвтрэх үү? Үйлдэл бүр audit-д таны нэрээр тэмдэглэгдэнэ.',
        )}
        confirmLabel={t('Нэрийн өмнөөс нэвтрэх')}
        onConfirm={async () => {
          if (target) await impersonate(target);
        }}
      />
    </>
  );
}
