'use client';

import { useCallback, useEffect, useState } from 'react';
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
  IconButton,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Tooltip,
  toast,
} from '@craftzbay/ui';
import { PageHead } from '@/components/states';
import { api, ApiError, type Device } from '@/lib/api';
import { useShell } from '@/components/shell';
import { useT } from '@/lib/i18n';

type Tone = 'success' | 'warning' | 'danger' | 'neutral';
const STATUS: Record<Device['status'], { label: string; tone: Tone }> = {
  active: { label: 'Ашиглагдаж байгаа', tone: 'success' },
  repair: { label: 'Засварт', tone: 'warning' },
  lost: { label: 'Алдагдсан', tone: 'danger' },
  retired: { label: 'Хассан', tone: 'neutral' },
};

type FormState = {
  id?: string;
  name: string;
  kind: string;
  serial: string;
  status: Device['status'];
  note: string;
};

const EMPTY: FormState = { name: '', kind: '', serial: '', status: 'active', note: '' };
const msg = (e: unknown, fallback: string) => (e instanceof ApiError ? e.message : fallback);

export default function DevicesPage() {
  const { me } = useShell();
  const { t } = useT();
  const [devices, setDevices] = useState<Device[] | null>(null);
  const [loadErr, setLoadErr] = useState('');
  const [q, setQ] = useState('');
  const [form, setForm] = useState<FormState | null>(null);
  const [removing, setRemoving] = useState<Device | null>(null);
  const manage = me.permissions['devices.manage']; // undefined | "all" | "own"

  const load = useCallback(async (query: string) => {
    try {
      const r = await api.get<{ devices: Device[] }>(
        `/api/apps/devices/?q=${encodeURIComponent(query)}`,
      );
      setDevices(r.devices);
      setLoadErr('');
    } catch (e) {
      setLoadErr(msg(e, 'Алдаа гарлаа'));
      setDevices([]);
    }
  }, []);

  useEffect(() => {
    const id = setTimeout(() => void load(q), q ? 250 : 0);
    return () => clearTimeout(id);
  }, [q, load]);

  const canEdit = (d: Device) =>
    manage === 'all' || (manage === 'own' && d.created_by === me.user.id);

  return (
    <>
      <PageHead
        title={t('Төхөөрөмжүүд')}
        description={t('Байгууллагын төхөөрөмжийн бүртгэл')}
        actions={
          <>
            <Input
              type="search"
              label={t('Хайх…')}
              hideLabel
              size="sm"
              placeholder={t('Хайх…')}
              prefix={<Icons.Search className="size-4" aria-hidden />}
              value={q}
              clearable
              onClear={() => setQ('')}
              onChange={(e) => setQ(e.target.value)}
              className="w-48"
            />
            {manage && (
              <Button leadingIcon={<Icons.Plus />} onClick={() => setForm(EMPTY)}>
                {t('Бүртгэх')}
              </Button>
            )}
          </>
        }
      />

      <Card padding="none">
        {loadErr ? (
          <div className="p-5">
            <Alert variant="danger">{t(loadErr)}</Alert>
          </div>
        ) : devices === null ? (
          <div className="flex justify-center py-12">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        ) : devices.length === 0 ? (
          <EmptyState
            icon={<Icons.Package />}
            title={q ? t('Илэрц алга') : t('Бүртгэл хоосон')}
            description={q ? undefined : t('Эхний төхөөрөмжөө бүртгээрэй')}
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Нэр')}</TableHead>
                <TableHead>{t('Төрөл')}</TableHead>
                <TableHead>{t('Сериал')}</TableHead>
                <TableHead>{t('Статус')}</TableHead>
                <TableHead>{t('Бүртгэсэн')}</TableHead>
                <TableHead align="right" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {devices.map((d) => (
                <TableRow key={d.id}>
                  <TableCell>
                    <span className="text-foreground block font-medium">{d.name}</span>
                    {d.note && (
                      <span className="text-foreground-subtle block text-xs">{d.note}</span>
                    )}
                  </TableCell>
                  <TableCell>{d.kind || '—'}</TableCell>
                  <TableCell>
                    <code className="font-mono text-xs">{d.serial}</code>
                  </TableCell>
                  <TableCell>
                    <Badge tone={STATUS[d.status]?.tone ?? 'neutral'}>
                      {t(STATUS[d.status]?.label ?? d.status)}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-foreground-muted">{d.owner_name}</TableCell>
                  <TableCell align="right">
                    {canEdit(d) && (
                      <span className="flex justify-end gap-1">
                        <Tooltip label={t('засах')}>
                          <IconButton
                            aria-label={`${d.name} — ${t('засах')}`}
                            icon={<Icons.Pencil />}
                            variant="ghost"
                            size="sm"
                            onClick={() =>
                              setForm({
                                id: d.id,
                                name: d.name,
                                kind: d.kind,
                                serial: d.serial,
                                status: d.status,
                                note: d.note,
                              })
                            }
                          />
                        </Tooltip>
                        <Tooltip label={t('устгах')}>
                          <IconButton
                            aria-label={`${d.name} — ${t('устгах')}`}
                            icon={<Icons.Trash2 />}
                            variant="ghost"
                            size="sm"
                            onClick={() => setRemoving(d)}
                          />
                        </Tooltip>
                      </span>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {form && (
        <DeviceDialog
          form={form}
          onClose={() => setForm(null)}
          onSaved={() => {
            setForm(null);
            void load(q);
          }}
        />
      )}

      <ConfirmationDialog
        open={removing !== null}
        onOpenChange={(o) => !o && setRemoving(null)}
        title={removing ? `"${removing.name}"` : ''}
        description={t('төхөөрөмжийг устгах уу?')}
        confirmLabel={t('устгах')}
        confirmVariant="destructive"
        onConfirm={async () => {
          if (!removing) return;
          await api.del(`/api/apps/devices/${removing.id}`);
          toast({ title: t('Устгагдлаа'), variant: 'success' });
          await load(q);
        }}
      />
    </>
  );
}

function DeviceDialog({
  form: initial,
  onClose,
  onSaved,
}: {
  form: FormState;
  onClose: () => void;
  onSaved: () => void;
}) {
  const { t } = useT();
  const [form, setForm] = useState(initial);
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setBusy(true);
    const body = {
      name: form.name,
      kind: form.kind,
      serial: form.serial,
      status: form.status,
      note: form.note,
    };
    try {
      if (form.id) await api.put(`/api/apps/devices/${form.id}`, body);
      else await api.post('/api/apps/devices/', body);
      toast({ title: form.id ? t('Хадгалагдлаа') : t('Бүртгэгдлээ'), variant: 'success' });
      onSaved();
    } catch (ex) {
      setErr(msg(ex, t('Алдаа гарлаа')));
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{form.id ? t('Төхөөрөмж засах') : t('Төхөөрөмж бүртгэх')}</DialogTitle>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          {err && <Alert variant="danger">{err}</Alert>}

          <Input
            label={t('Нэр')}
            value={form.name}
            required
            autoFocus
            placeholder="Dell Latitude 5540"
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <Input
            label={t('Төрөл')}
            value={form.kind}
            placeholder="laptop, printer…"
            onChange={(e) => setForm({ ...form, kind: e.target.value })}
          />
          <Input
            label={t('Сериал')}
            value={form.serial}
            onChange={(e) => setForm({ ...form, serial: e.target.value })}
          />

          <div className="space-y-1.5">
            <label htmlFor="device-status" className="text-foreground block text-sm font-medium">
              {t('Статус')}
            </label>
            <Select
              value={form.status}
              onValueChange={(v) => setForm({ ...form, status: v as Device['status'] })}
            >
              <SelectTrigger id="device-status" placeholder={t('Статус')} />
              <SelectContent>
                {Object.entries(STATUS).map(([k, v]) => (
                  <SelectItem key={k} value={k}>
                    {t(v.label)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Input
            label={t('Тэмдэглэл')}
            value={form.note}
            onChange={(e) => setForm({ ...form, note: e.target.value })}
          />

          <DialogFooter>
            <Button type="button" variant="secondary" onClick={onClose}>
              {t('Болих')}
            </Button>
            <Button type="submit" loading={busy}>
              {t('Хадгалах')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
