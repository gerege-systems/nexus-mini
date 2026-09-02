'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  Alert,
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
} from '@gerege-systems/ui';
import { PageHead } from '@/components/states';
import { api, ApiError } from '@/lib/api';
import { useShell } from '@/components/shell';
import { useT } from '@/lib/i18n';

type Person = {
  membership_id: string;
  user_id: string;
  name: string;
  department_id: string | null;
  department_name: string;
  job_title: string;
};
type Dept = { id: string; name: string; active: boolean };
type Form = { membership_id: string; name: string; department_id: string; job_title: string };

// Radix Select нь хоосон string value зөвшөөрдөггүй — «хэлтэсгүй»-г маркераар.
const NO_DEPT = '__none__';
const msg = (e: unknown, fallback: string) => (e instanceof ApiError ? e.message : fallback);

export default function PeoplePage() {
  const { t } = useT();
  const { me } = useShell();
  const manage = !!me.permissions['organisation.manage'];
  const [people, setPeople] = useState<Person[] | null>(null);
  const [depts, setDepts] = useState<Dept[]>([]);
  const [loadErr, setLoadErr] = useState('');
  const [form, setForm] = useState<Form | null>(null);

  const load = useCallback(async () => {
    try {
      const [p, d] = await Promise.all([
        api.get<{ people: Person[] }>('/api/apps/organisation/people'),
        api.get<{ departments: Dept[] }>('/api/apps/organisation/departments'),
      ]);
      setPeople(p.people);
      setDepts(d.departments.filter((x) => x.active));
      setLoadErr('');
    } catch (e) {
      setLoadErr(msg(e, 'Алдаа гарлаа'));
      setPeople([]);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <>
      <PageHead title={t('Ажилтнууд')} description={t('Гишүүн бүрийн хэлтэс, албан тушаал')} />

      <Card padding="none">
        {loadErr ? (
          <div className="p-5">
            <Alert variant="danger">{t(loadErr)}</Alert>
          </div>
        ) : people === null ? (
          <div className="flex justify-center py-12">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        ) : people.length === 0 ? (
          <EmptyState icon={<Icons.Users />} title={t('Гишүүн байхгүй')} />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Нэр')}</TableHead>
                <TableHead>{t('Хэлтэс')}</TableHead>
                <TableHead>{t('Албан тушаал')}</TableHead>
                <TableHead align="right" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {people.map((p) => (
                <TableRow key={p.membership_id}>
                  <TableCell className="font-medium">{p.name}</TableCell>
                  <TableCell>{p.department_name || '—'}</TableCell>
                  <TableCell>{p.job_title || '—'}</TableCell>
                  <TableCell align="right">
                    {manage && (
                      <Tooltip label={t('засах')}>
                        <IconButton
                          aria-label={`${p.name} — ${t('засах')}`}
                          icon={<Icons.Pencil />}
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            setForm({
                              membership_id: p.membership_id,
                              name: p.name,
                              department_id: p.department_id ?? '',
                              job_title: p.job_title,
                            })
                          }
                        />
                      </Tooltip>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {form && (
        <PersonDialog
          form={form}
          depts={depts}
          onClose={() => setForm(null)}
          onSaved={() => {
            setForm(null);
            void load();
          }}
        />
      )}
    </>
  );
}

function PersonDialog({
  form: initial,
  depts,
  onClose,
  onSaved,
}: {
  form: Form;
  depts: Dept[];
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
    try {
      await api.put(`/api/apps/organisation/people/${form.membership_id}`, {
        department_id: form.department_id || null,
        job_title: form.job_title,
      });
      toast({ title: t('Хадгалагдлаа'), variant: 'success' });
      onSaved();
    } catch (ex) {
      setErr(msg(ex, t('Алдаа гарлаа')));
      setBusy(false);
    }
  };

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent size="sm">
        <DialogHeader>
          <DialogTitle>{form.name}</DialogTitle>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          {err && <Alert variant="danger">{err}</Alert>}

          <div className="space-y-1.5">
            <label htmlFor="dept" className="text-foreground block text-sm font-medium">
              {t('Хэлтэс')}
            </label>
            <Select
              value={form.department_id || NO_DEPT}
              onValueChange={(v) =>
                setForm({ ...form, department_id: v === NO_DEPT ? '' : v })
              }
            >
              <SelectTrigger id="dept" placeholder="—" />
              <SelectContent>
                <SelectItem value={NO_DEPT}>—</SelectItem>
                {depts.map((d) => (
                  <SelectItem key={d.id} value={d.id}>
                    {d.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <Input
            label={t('Албан тушаал')}
            value={form.job_title}
            maxLength={120}
            autoFocus
            onChange={(e) => setForm({ ...form, job_title: e.target.value })}
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
