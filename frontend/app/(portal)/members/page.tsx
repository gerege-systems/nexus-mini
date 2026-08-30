'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
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
import { api, ApiError, type Member, type Role } from '@/lib/api';
import { useShell } from '@/components/shell';
import { useT } from '@/lib/i18n';

const err = (e: unknown, fallback: string) => (e instanceof ApiError ? e.message : fallback);

/** Role-ийг pill toggle-оор — багц бага, checkbox жагсаалт хэт урт болно. */
function RoleToggles({
  codes,
  selected,
  disabled,
  disabledHint,
  onToggle,
}: {
  codes: string[];
  selected: string[];
  disabled?: boolean;
  disabledHint?: string;
  onToggle: (code: string, on: boolean) => void;
}) {
  const pills = (
    <span className="flex flex-wrap gap-1.5">
      {codes.map((code) => {
        const on = selected.includes(code);
        return (
          <Button
            key={code}
            type="button"
            size="sm"
            variant={on ? 'primary' : 'outline'}
            disabled={disabled}
            aria-pressed={on}
            onClick={() => onToggle(code, !on)}
          >
            {code}
          </Button>
        );
      })}
    </span>
  );
  return disabled && disabledHint ? <Tooltip label={disabledHint}>{pills}</Tooltip> : pills;
}

export default function MembersPage() {
  const { me } = useShell();
  const { t } = useT();
  const [members, setMembers] = useState<Member[] | null>(null);
  const [roles, setRoles] = useState<Role[]>([]);
  const [loadErr, setLoadErr] = useState('');
  const [adding, setAdding] = useState(false);
  const [removing, setRemoving] = useState<Member | null>(null);

  const load = useCallback(async () => {
    try {
      const [m, r] = await Promise.all([
        api.get<{ members: Member[] }>('/api/members'),
        me.permissions['core.roles.manage']
          ? api.get<{ roles: Role[] }>('/api/roles')
          : Promise.resolve({ roles: [] as Role[] }),
      ]);
      setMembers(m.members);
      setRoles(r.roles);
    } catch (e) {
      setLoadErr(err(e, t('Алдаа гарлаа')));
    }
  }, [me.permissions, t]);

  useEffect(() => {
    void load();
  }, [load]);

  const roleCodes = roles.length > 0 ? roles.map((r) => r.code) : ['admin', 'manager', 'user'];

  const setMemberRoles = async (m: Member, code: string, on: boolean) => {
    const next = on ? [...m.roles, code] : m.roles.filter((r) => r !== code);
    try {
      await api.put(`/api/members/${m.membership_id}/roles`, { roles: next });
      toast({ title: t('Role шинэчлэгдлээ'), variant: 'success' });
    } catch (e) {
      toast({ title: err(e, t('Алдаа гарлаа')), variant: 'danger' });
    }
    await load();
  };

  return (
    <>
      <PageHead
        title={t('Гишүүд')}
        description={t('Байгууллагын гишүүд ба role оноолт')}
        actions={
          <Button leadingIcon={<Icons.Plus />} onClick={() => setAdding(true)}>
            {t('Гишүүн нэмэх')}
          </Button>
        }
      />

      <Card padding="none">
        {loadErr ? (
          <div className="p-5">
            <Alert variant="danger">{loadErr}</Alert>
          </div>
        ) : members === null ? (
          <div className="flex justify-center py-12">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        ) : members.length === 0 ? (
          <EmptyState icon={<Icons.Users />} title={t('Гишүүн алга')} />
        ) : (
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
              {members.map((m) => {
                const self = m.user_id === me.user.id;
                return (
                  <TableRow key={m.membership_id}>
                    <TableCell className="font-medium">{m.name}</TableCell>
                    <TableCell className="text-foreground-muted">{m.email}</TableCell>
                    <TableCell>
                      <RoleToggles
                        codes={roleCodes}
                        selected={m.roles}
                        disabled={self}
                        disabledHint={t('Өөрийн role-г эндээс өөрчлөхгүй')}
                        onToggle={(code, on) => setMemberRoles(m, code, on)}
                      />
                    </TableCell>
                    <TableCell align="right">
                      {!self && (
                        <Tooltip label={t('Хасах')}>
                          <IconButton
                            aria-label={`${m.name} — ${t('Хасах')}`}
                            icon={<Icons.Trash2 />}
                            variant="ghost"
                            size="sm"
                            onClick={() => setRemoving(m)}
                          />
                        </Tooltip>
                      )}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </Card>

      {adding && (
        <AddMemberDialog
          roleCodes={roleCodes}
          onClose={() => setAdding(false)}
          onAdded={() => {
            setAdding(false);
            void load();
          }}
        />
      )}

      <ConfirmationDialog
        open={removing !== null}
        onOpenChange={(o) => !o && setRemoving(null)}
        title={removing ? `${removing.name} ${t('хасах уу?')}` : ''}
        confirmLabel={t('Хасах')}
        confirmVariant="destructive"
        onConfirm={async () => {
          if (!removing) return;
          await api.del(`/api/members/${removing.membership_id}`);
          toast({ title: t('Гишүүн хасагдлаа'), variant: 'success' });
          await load();
        }}
      />
    </>
  );
}

function AddMemberDialog({
  roleCodes,
  onClose,
  onAdded,
}: {
  roleCodes: string[];
  onClose: () => void;
  onAdded: () => void;
}) {
  const { t } = useT();
  const [form, setForm] = useState({ email: '', name: '', password: '', roles: ['user'] });
  const [msg, setMsg] = useState('');
  const [busy, setBusy] = useState(false);
  // null = хараахан хайгаагүй / хүчингүй имэйл
  const [lookup, setLookup] = useState<{ exists: boolean; name?: string; member?: boolean } | null>(
    null,
  );
  const emailOk = /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email.trim());

  useEffect(() => {
    if (!emailOk) {
      setLookup(null);
      return;
    }
    const id = setTimeout(() => {
      api
        .get<{ exists: boolean; name?: string; member?: boolean }>(
          `/api/members/lookup?email=${encodeURIComponent(form.email.trim())}`,
        )
        .then(setLookup)
        .catch(() => setLookup(null));
    }, 350);
    return () => clearTimeout(id);
  }, [form.email, emailOk]);

  const hint = !emailOk
    ? t('Имэйлээр хайна: бүртгэлтэй бол нэр нь гарна, үгүй бол шинээр үүсгэнэ')
    : lookup === null
      ? t('Хайж байна…')
      : lookup.exists
        ? lookup.member
          ? t('Аль хэдийн энэ байгууллагын гишүүн')
          : t('Бүртгэлтэй хэрэглэгч — role өгөөд нэмнэ')
        : t('Бүртгэлгүй — нэр, түр нууц үг өгч шинээр үүсгэнэ');

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setMsg('');
    setBusy(true);
    try {
      await api.post('/api/members', form);
      toast({ title: t('Гишүүн нэмэгдлээ'), variant: 'success' });
      onAdded();
    } catch (ex) {
      setMsg(err(ex, t('Алдаа гарлаа')));
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{t('Гишүүн нэмэх')}</DialogTitle>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          {msg && <Alert variant="danger">{msg}</Alert>}

          <Input
            type="email"
            label={t('Имэйл')}
            value={form.email}
            autoFocus
            required
            helperText={hint}
            error={lookup?.member ? t('Аль хэдийн энэ байгууллагын гишүүн') : undefined}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
          />

          {lookup?.exists && <Input label={t('Нэр')} value={lookup.name ?? ''} readOnly disabled />}

          {lookup && !lookup.exists && (
            <>
              <Input
                label={t('Нэр')}
                value={form.name}
                required
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
              <Input
                type="password"
                label={t('Түр нууц үг')}
                autoComplete="new-password"
                value={form.password}
                required
                minLength={8}
                helperText={t('8+ тэмдэгт: латин үсэг, тоо, тусгай тэмдэгт (кирилл хориотой)')}
                onChange={(e) => setForm({ ...form, password: e.target.value })}
              />
            </>
          )}

          <div className="space-y-1.5">
            <span className="text-foreground block text-sm font-medium">Role</span>
            <RoleToggles
              codes={roleCodes}
              selected={form.roles}
              onToggle={(code, on) =>
                setForm({
                  ...form,
                  roles: on ? [...form.roles, code] : form.roles.filter((r) => r !== code),
                })
              }
            />
          </div>

          <DialogFooter>
            <Button type="button" variant="secondary" onClick={onClose}>
              {t('Болих')}
            </Button>
            <Button type="submit" loading={busy} disabled={!lookup || lookup.member}>
              {t('Нэмэх')}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
