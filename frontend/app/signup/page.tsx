'use client';

import { useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Alert, Button, Input } from '@craftzbay/ui';
import { AuthCard } from '@/components/auth-card';
import { api, ApiError } from '@/lib/api';
import { slugify } from '@/lib/slug';
import { useT } from '@/lib/i18n';

export default function SignupPage() {
  const [form, setForm] = useState({
    name: '',
    email: '',
    password: '',
    tenant_name: '',
    tenant_slug: '',
  });
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
      await api.post('/api/signup', form);
      // Шинэ байгууллага — app store-оос эхэлнэ.
      router.replace('/store');
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : t('Алдаа гарлаа'));
      setBusy(false);
    }
  };

  return (
    <AuthCard
      title={t('Байгууллагаа бүртгүүлэх')}
      subtitle={t('Бүртгүүлмэгц app store-оос модулиа сонгоно')}
      footer={
        <>
          {t('Бүртгэлтэй юу?')}{' '}
          <Link href="/login" className="text-accent hover:underline">
            {t('Нэвтрэх')}
          </Link>
        </>
      }
    >
      {err && (
        <Alert variant="danger" className="mb-4">
          {err}
        </Alert>
      )}

      <form onSubmit={submit} className="space-y-4">
        <Input
          label={t('Таны нэр')}
          autoComplete="name"
          value={form.name}
          autoFocus
          required
          onChange={(e) => setForm({ ...form, name: e.target.value })}
        />
        <Input
          type="email"
          label={t('Имэйл')}
          autoComplete="username"
          value={form.email}
          required
          onChange={(e) => setForm({ ...form, email: e.target.value })}
        />
        <Input
          type="password"
          label={t('Нууц үг')}
          autoComplete="new-password"
          value={form.password}
          required
          minLength={8}
          helperText={t('8+ тэмдэгт: латин үсэг, тоо, тусгай тэмдэгт (кирилл хориотой)')}
          onChange={(e) => setForm({ ...form, password: e.target.value })}
        />
        <Input
          label={t('Байгууллагын нэр')}
          value={form.tenant_name}
          required
          onChange={(e) =>
            setForm({
              ...form,
              tenant_name: e.target.value,
              tenant_slug: slugTouched ? form.tenant_slug : slugify(e.target.value),
            })
          }
        />
        <Input
          label={t('Богино нэр (slug)')}
          value={form.tenant_slug}
          required
          helperText={t('Жижиг латин үсэг, тоо, зураас')}
          onChange={(e) => {
            setSlugTouched(true);
            setForm({ ...form, tenant_slug: slugify(e.target.value) });
          }}
        />
        <Button type="submit" className="w-full" loading={busy}>
          {t('Бүртгүүлэх')}
        </Button>
      </form>
    </AuthCard>
  );
}
