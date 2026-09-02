'use client';

import { useEffect, useState } from 'react';
import { Alert, Button, Card, Input, Spinner, toast } from '@gerege-systems/ui';
import { PageHead } from '@/components/states';
import { api, ApiError } from '@/lib/api';
import { useShell } from '@/components/shell';
import { useT } from '@/lib/i18n';

type Profile = {
  name: string;
  slug: string;
  legal_name: string;
  registration_number: string;
  tax_number: string;
  address: string;
  phone: string;
  email: string;
  website: string;
};

const FIELDS: { key: keyof Profile; label: string; max: number; type?: string }[] = [
  { key: 'name', label: 'Байгууллагын нэр', max: 120 },
  { key: 'legal_name', label: 'Хуулийн нэр', max: 200 },
  { key: 'registration_number', label: 'Регистрийн дугаар', max: 32 },
  { key: 'tax_number', label: 'ТТД', max: 32 },
  { key: 'address', label: 'Хаяг', max: 500 },
  { key: 'phone', label: 'Утас', max: 32, type: 'tel' },
  { key: 'email', label: 'Имэйл', max: 255, type: 'email' },
  { key: 'website', label: 'Вэб сайт', max: 255, type: 'url' },
];

export default function SettingsPage() {
  const { t } = useT();
  const { me, refresh } = useShell();
  const canEdit = !!me.permissions['core.settings.manage'];
  const [p, setP] = useState<Profile | null>(null);
  const [loadErr, setLoadErr] = useState('');
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let alive = true;
    api
      .get<Profile>('/api/tenant/profile')
      .then((d) => alive && setP(d))
      .catch((e) => alive && setLoadErr(e instanceof ApiError ? e.message : t('Алдаа гарлаа')));
    return () => {
      alive = false;
    };
  }, [t]);

  const save = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!p) return;
    setErr('');
    setBusy(true);
    try {
      await api.put('/api/tenant/profile', p);
      toast({ title: t('Хадгалагдлаа'), variant: 'success' });
      refresh();
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : t('Алдаа гарлаа'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <PageHead
        title={t('Байгууллагын тохиргоо')}
        description={t('Нэр, хуулийн мэдээлэл, холбоо барих')}
      />

      <Card className="max-w-2xl">
        {loadErr ? (
          <Alert variant="danger">{loadErr}</Alert>
        ) : !p ? (
          <div className="flex justify-center py-8">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        ) : (
          <form onSubmit={save} className="space-y-4">
            {err && <Alert variant="danger">{err}</Alert>}

            <Input label="Slug" value={p.slug} disabled readOnly helperText={t('Slug өөрчлөгдөхгүй')} />

            {FIELDS.map((f) => (
              <Input
                key={f.key}
                type={f.type}
                label={t(f.label)}
                value={p[f.key]}
                maxLength={f.max}
                disabled={!canEdit}
                onChange={(e) => setP({ ...p, [f.key]: e.target.value })}
              />
            ))}

            {canEdit ? (
              <Button type="submit" loading={busy}>
                {t('Хадгалах')}
              </Button>
            ) : (
              <p className="text-foreground-subtle text-sm">{t('Засах эрхгүй — зөвхөн харах')}</p>
            )}
          </form>
        )}
      </Card>
    </>
  );
}
