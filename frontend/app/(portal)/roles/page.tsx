'use client';

import { Fragment, useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  Icons,
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
import { api, ApiError, type Permission, type Role } from '@/lib/api';
import { useT } from '@/lib/i18n';

// Radix Select нь хоосон string value зөвшөөрдөггүй — «өвлөхгүй»-г маркераар.
const NO_IMPLIES = '__none__';

const msg = (e: unknown, fallback: string) => (e instanceof ApiError ? e.message : fallback);

// Role × permission матриц: нүд бүр — / all / own гэсэн 3 төлөвт шилжинэ.
export default function RolesPage() {
  const { t } = useT();
  const [roles, setRoles] = useState<Role[] | null>(null);
  const [perms, setPerms] = useState<Permission[]>([]);
  const [err, setErr] = useState('');
  const [creating, setCreating] = useState(false);

  const load = useCallback(async () => {
    try {
      const [r, p] = await Promise.all([
        api.get<{ roles: Role[] }>('/api/roles'),
        api.get<{ permissions: Permission[] }>('/api/permissions'),
      ]);
      setRoles(r.roles);
      setPerms(p.permissions);
    } catch (e) {
      setErr(msg(e, t('Алдаа гарлаа')));
      setRoles([]);
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const groups = useMemo(() => {
    const g = new Map<string, Permission[]>();
    for (const p of perms) {
      // Модулийн ID урт тул permission кодын prefix-ээр нэрлэнэ.
      const key = p.module_id === 'core' ? 'core' : p.code.split('.')[0];
      if (!g.has(key)) g.set(key, []);
      g.get(key)!.push(p);
    }
    return [...g.entries()];
  }, [perms]);

  const cycle = async (role: Role, p: Permission) => {
    if (role.code === 'admin') return; // admin үргэлж бүгдийг эзэмшинэ
    const cur = role.grants[p.code];
    let next: 'all' | 'own' | undefined;
    if (!cur) next = 'all';
    else if (cur === 'all' && p.own_scope) next = 'own';
    else next = undefined;
    const grants = { ...role.grants };
    if (next) grants[p.code] = next;
    else delete grants[p.code];
    try {
      await api.put(`/api/roles/${role.id}/grants`, { grants });
      toast({ title: t('Оноолт хадгалагдлаа'), variant: 'success' });
      await load();
    } catch (e) {
      setErr(msg(e, t('Алдаа гарлаа')));
    }
  };

  return (
    <>
      <PageHead
        title={t('Эрхийн тохиргоо')}
        description={t(
          'Нүд дарж — → бүгд → өөрийн гэж эргэлдэнэ. Role нь implies-ээрээ доод role-ийн эрхийг өвлөнө.',
        )}
        actions={
          <Button leadingIcon={<Icons.Plus />} onClick={() => setCreating(true)}>
            {t('Role нэмэх')}
          </Button>
        }
      />

      {err && (
        <Alert variant="danger" className="mb-4" dismissible onDismiss={() => setErr('')}>
          {err}
        </Alert>
      )}

      <Card padding="none">
        {roles === null ? (
          <div className="flex justify-center py-12">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        ) : (
          <Table containerClassName="overflow-x-auto">
            <TableHeader>
              <TableRow>
                <TableHead>Permission</TableHead>
                {roles.map((r) => (
                  <TableHead key={r.id} align="center" className="border-border min-w-[9rem] border-l">
                    <span className="block">{r.name}</span>
                    <span className="text-foreground-subtle block font-normal">
                      <code className="font-mono text-xs">{r.code}</code>
                      {r.implies && <span> ⊃ {r.implies}</span>}
                    </span>
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {groups.map(([group, list]) => (
                <Fragment key={group}>
                  <TableRow>
                    <TableCell
                      colSpan={roles.length + 1}
                      className="bg-background-subtle text-foreground-muted text-xs font-semibold tracking-wide uppercase"
                    >
                      {group === 'core' ? t('Платформ') : group}
                    </TableCell>
                  </TableRow>
                  {list.map((p) => (
                    <TableRow key={p.code}>
                      <TableCell>
                        <span className="text-foreground block font-medium">{p.name}</span>
                        <code className="text-foreground-subtle font-mono text-xs">{p.code}</code>
                      </TableCell>
                      {roles.map((r) => {
                        const v = r.code === 'admin' ? 'all' : r.grants[p.code];
                        const cell = (
                          <Button
                            type="button"
                            size="sm"
                            variant={v ? 'primary' : 'outline'}
                            className="min-w-[4.5rem]"
                            disabled={r.code === 'admin'}
                            aria-label={`${p.name} — ${r.name}`}
                            onClick={() => cycle(r, p)}
                          >
                            {v === 'all' ? t('Бүгд') : v === 'own' ? t('Өөрийн') : '—'}
                          </Button>
                        );
                        return (
                          <TableCell key={r.id} align="center" className="border-border border-l">
                            {r.code === 'admin' ? (
                              <Tooltip label={t('Админ үргэлж бүх эрхтэй')}>{cell}</Tooltip>
                            ) : (
                              cell
                            )}
                          </TableCell>
                        );
                      })}
                    </TableRow>
                  ))}
                </Fragment>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      <p className="text-foreground-subtle mt-3 flex items-center gap-2 text-sm">
        <Icons.Key className="size-4 shrink-0" aria-hidden />
        {t('«Өөрийн» = зөвхөн өөрийн үүсгэсэн бүртгэл дээр үйлдэл хийнэ (модуль нь дэмждэг бол)')}
      </p>

      {creating && roles && (
        <CreateRoleDialog
          roles={roles}
          onClose={() => setCreating(false)}
          onCreated={() => {
            setCreating(false);
            void load();
          }}
        />
      )}
    </>
  );
}

function CreateRoleDialog({
  roles,
  onClose,
  onCreated,
}: {
  roles: Role[];
  onClose: () => void;
  onCreated: () => void;
}) {
  const { t } = useT();
  const [form, setForm] = useState({ code: '', name: '', implies: '' });
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setBusy(true);
    try {
      await api.post('/api/roles', form);
      toast({ title: t('Role үүслээ'), variant: 'success' });
      onCreated();
    } catch (ex) {
      setErr(msg(ex, t('Алдаа гарлаа')));
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{t('Role нэмэх')}</DialogTitle>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          {err && <Alert variant="danger">{err}</Alert>}

          <Input
            label={t('Код')}
            value={form.code}
            autoFocus
            required
            placeholder="warehouse_staff"
            helperText={t('Жижиг үсэг, тоо, _')}
            onChange={(e) => setForm({ ...form, code: e.target.value })}
          />
          <Input
            label={t('Нэр')}
            value={form.name}
            required
            placeholder={t('Агуулахын ажилтан')}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <div className="space-y-1.5">
            <label htmlFor="implies" className="text-foreground block text-sm font-medium">
              {t('Өвлөх role (сонголттой)')}
            </label>
            <Select
              value={form.implies || NO_IMPLIES}
              onValueChange={(v) => setForm({ ...form, implies: v === NO_IMPLIES ? '' : v })}
            >
              <SelectTrigger id="implies" placeholder={t('— өвлөхгүй —')} />
              <SelectContent>
                <SelectItem value={NO_IMPLIES}>{t('— өвлөхгүй —')}</SelectItem>
                {roles.map((r) => (
                  <SelectItem key={r.id} value={r.code}>
                    {r.name} ({r.code})
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <DialogFooter>
            <Button type="button" variant="secondary" onClick={onClose}>
              {t('Болих')}
            </Button>
            <Button type="submit" loading={busy}>
              {t('Үүсгэх')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
