'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Checkbox,
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
  cn,
  toast,
} from '@craftzbay/ui';
import { PageHead } from '@/components/states';
import { api, ApiError } from '@/lib/api';
import { useShell } from '@/components/shell';
import { useT } from '@/lib/i18n';

type Dept = {
  id: string;
  code: string;
  name: string;
  parent_id: string | null;
  manager_membership_id: string | null;
  manager_name: string;
  active: boolean;
  people: number;
};
type Person = { membership_id: string; name: string };
type Form = {
  id?: string;
  code: string;
  name: string;
  parent_id: string;
  manager_membership_id: string;
  active: boolean;
};

const EMPTY: Form = {
  code: '',
  name: '',
  parent_id: '',
  manager_membership_id: '',
  active: true,
};
// Radix Select нь хоосон string value зөвшөөрдөггүй.
const NONE = '__none__';
const msg = (e: unknown, fallback: string) => (e instanceof ApiError ? e.message : fallback);

// Хавтгай жагсаалтыг мод болгож, гүнтэй нь дарааллуулна.
function flatten(list: Dept[]): { d: Dept; depth: number }[] {
  const byParent = new Map<string | null, Dept[]>();
  for (const d of list) {
    const k = d.parent_id && list.some((x) => x.id === d.parent_id) ? d.parent_id : null;
    byParent.set(k, [...(byParent.get(k) ?? []), d]);
  }
  const out: { d: Dept; depth: number }[] = [];
  const walk = (parent: string | null, depth: number) => {
    for (const d of byParent.get(parent) ?? []) {
      out.push({ d, depth });
      if (depth < 32) walk(d.id, depth + 1);
    }
  };
  walk(null, 0);
  return out;
}

export default function DepartmentsPage() {
  const { t } = useT();
  const { me } = useShell();
  const manage = !!me.permissions['organisation.manage'];
  const [depts, setDepts] = useState<Dept[] | null>(null);
  const [people, setPeople] = useState<Person[]>([]);
  const [loadErr, setLoadErr] = useState('');
  const [form, setForm] = useState<Form | null>(null);
  const [removing, setRemoving] = useState<Dept | null>(null);

  const load = useCallback(async () => {
    try {
      const [d, p] = await Promise.all([
        api.get<{ departments: Dept[] }>('/api/apps/organisation/departments'),
        api
          .get<{ people: Person[] }>('/api/apps/organisation/people')
          .catch(() => ({ people: [] })),
      ]);
      setDepts(d.departments);
      setPeople(p.people);
      setLoadErr('');
    } catch (e) {
      setLoadErr(msg(e, 'Алдаа гарлаа'));
      setDepts([]);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const tree = useMemo(() => (depts ? flatten(depts) : []), [depts]);

  return (
    <>
      <PageHead
        title={t('Хэлтэс, нэгж')}
        description={t('Байгууллагын бүтцийн мод')}
        actions={
          manage && (
            <Button leadingIcon={<Icons.Plus />} onClick={() => setForm(EMPTY)}>
              {t('Нэгж нэмэх')}
            </Button>
          )
        }
      />

      <Card padding="none">
        {loadErr ? (
          <div className="p-5">
            <Alert variant="danger">{t(loadErr)}</Alert>
          </div>
        ) : depts === null ? (
          <div className="flex justify-center py-12">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        ) : depts.length === 0 ? (
          <EmptyState
            icon={<Icons.Folder />}
            title={t('Нэгж байхгүй')}
            description={t('Эхний хэлтэс/нэгжээ үүсгээрэй')}
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Нэгж')}</TableHead>
                <TableHead>{t('Код')}</TableHead>
                <TableHead>{t('Менежер')}</TableHead>
                <TableHead align="right">{t('Ажилтан')}</TableHead>
                <TableHead align="right" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {tree.map(({ d, depth }) => (
                <TableRow key={d.id} className={cn(!d.active && 'opacity-60')}>
                  <TableCell>
                    <span
                      className="flex items-center gap-2"
                      style={{ paddingInlineStart: `${depth * 1.25}rem` }}
                    >
                      <span className="text-foreground font-medium">{d.name}</span>
                      {!d.active && <Badge tone="neutral">{t('идэвхгүй')}</Badge>}
                    </span>
                  </TableCell>
                  <TableCell>
                    <code className="font-mono text-xs">{d.code}</code>
                  </TableCell>
                  <TableCell className="text-foreground-muted">{d.manager_name || '—'}</TableCell>
                  <TableCell align="right">{d.people}</TableCell>
                  <TableCell align="right">
                    {manage && (
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
                                code: d.code,
                                name: d.name,
                                parent_id: d.parent_id ?? '',
                                manager_membership_id: d.manager_membership_id ?? '',
                                active: d.active,
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
        <DeptDialog
          form={form}
          tree={tree}
          people={people}
          onClose={() => setForm(null)}
          onSaved={() => {
            setForm(null);
            void load();
          }}
        />
      )}

      <ConfirmationDialog
        open={removing !== null}
        onOpenChange={(o) => !o && setRemoving(null)}
        title={removing ? `"${removing.name}"` : ''}
        description={t('хэлтсийг устгах уу? Харьяа нэгжүүд дээд түвшингүй болно.')}
        confirmLabel={t('устгах')}
        confirmVariant="destructive"
        onConfirm={async () => {
          if (!removing) return;
          await api.del(`/api/apps/organisation/departments/${removing.id}`);
          toast({ title: t('Устгагдлаа'), variant: 'success' });
          await load();
        }}
      />
    </>
  );
}

function DeptDialog({
  form: initial,
  tree,
  people,
  onClose,
  onSaved,
}: {
  form: Form;
  tree: { d: Dept; depth: number }[];
  people: Person[];
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
      code: form.code,
      name: form.name,
      parent_id: form.parent_id || null,
      manager_membership_id: form.manager_membership_id || null,
      active: form.active,
    };
    try {
      if (form.id) await api.put(`/api/apps/organisation/departments/${form.id}`, body);
      else await api.post('/api/apps/organisation/departments', body);
      toast({ title: t('Хадгалагдлаа'), variant: 'success' });
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
          <DialogTitle>{form.id ? t('Нэгж засах') : t('Нэгж нэмэх')}</DialogTitle>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          {err && <Alert variant="danger">{err}</Alert>}

          <Input
            label={t('Нэр')}
            value={form.name}
            required
            autoFocus
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />
          <Input
            label={t('Код')}
            value={form.code}
            required
            placeholder="hr, it, sales…"
            onChange={(e) => setForm({ ...form, code: e.target.value })}
          />

          <div className="space-y-1.5">
            <label htmlFor="parent" className="text-foreground block text-sm font-medium">
              {t('Дээд нэгж')}
            </label>
            <Select
              value={form.parent_id || NONE}
              onValueChange={(v) => setForm({ ...form, parent_id: v === NONE ? '' : v })}
            >
              <SelectTrigger id="parent" placeholder={t('— байхгүй (дээд түвшин) —')} />
              <SelectContent>
                <SelectItem value={NONE}>{t('— байхгүй (дээд түвшин) —')}</SelectItem>
                {tree
                  .filter((x) => x.d.id !== form.id)
                  .map(({ d, depth }) => (
                    <SelectItem key={d.id} value={d.id}>
                      {' '.repeat(depth * 3)}
                      {d.name}
                    </SelectItem>
                  ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-1.5">
            <label htmlFor="manager" className="text-foreground block text-sm font-medium">
              {t('Менежер')}
            </label>
            <Select
              value={form.manager_membership_id || NONE}
              onValueChange={(v) =>
                setForm({ ...form, manager_membership_id: v === NONE ? '' : v })
              }
            >
              <SelectTrigger id="manager" placeholder="—" />
              <SelectContent>
                <SelectItem value={NONE}>—</SelectItem>
                {people.map((p) => (
                  <SelectItem key={p.membership_id} value={p.membership_id}>
                    {p.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {form.id && (
            <Checkbox
              checked={form.active}
              onCheckedChange={(v) => setForm({ ...form, active: v === true })}
              label={t('Идэвхтэй')}
            />
          )}

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
