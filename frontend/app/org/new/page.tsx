'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Button, Input } from '@craftzbay/ui';
import { AuthCard } from '@/components/auth-card';
import { api, ApiError } from '@/lib/api';
import { slugify } from '@/lib/slug';
import { useT } from '@/lib/i18n';

// Нэвтэрсэн ч байгууллагагүй (эсвэл шинээр нэмэх) хэрэглэгчид.
export default function NewOrgPage() {
  const [org, setOrg] = useState({ name: '', slug: '' });
  const [slugTouched, setSlugTouched] = useState(false);
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);
  const router = useRouter();
  const { t } = useT();

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setBusy(true);
    try {
      const r = await api.post<{ tenant_id: string }>('/api/tenants', org);
      await api.post('/api/session/tenant', { tenant_id: r.tenant_id });
      router.replace('/store');
    } catch (ex) {
      if (ex instanceof ApiError && ex.status === 401) router.replace('/login');
      else setErr(ex instanceof ApiError ? ex.message : t('Алдаа гарлаа'));
      setBusy(false);
    }
  };

  return (
    <AuthCard
      title={t('Байгууллага үүсгэх')}
      subtitle={t('Ажлын талбараа үүсгээд store-оос модулиа сонгоно')}
    >
      {err && (
        <Alert variant="danger" className="mb-4">
          {err}
        </Alert>
      )}

      <form onSubmit={submit} className="space-y-4">
        <Input
          label={t('Байгууллагын нэр')}
          value={org.name}
          autoFocus
          required
          onChange={(e) =>
            setOrg({
              name: e.target.value,
              slug: slugTouched ? org.slug : slugify(e.target.value),
            })
          }
        />
        <Input
          label={t('Богино нэр (slug)')}
          value={org.slug}
          required
          helperText={t('Жижиг латин үсэг, тоо, зураас')}
          onChange={(e) => {
            setSlugTouched(true);
            setOrg({ ...org, slug: slugify(e.target.value) });
          }}
        />
        <Button type="submit" className="w-full" loading={busy}>
          {t('Үүсгэх')}
        </Button>
      </form>
    </AuthCard>
  );
}
