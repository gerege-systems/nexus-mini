'use client';

import { useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { Alert, Button, Input, Separator } from '@gerege-systems/ui';
import { AuthCard } from '@/components/auth-card';
import { api, ApiError } from '@/lib/api';
import { useT } from '@/lib/i18n';
import { setupStatus } from '@/lib/setup';

export default function LoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);
  const router = useRouter();
  const { t } = useT();
  // ?next= — зөвхөн энэ сайтын харьцангуй зам (open redirect хаалттай).
  const [next, setNext] = useState('/dashboard');
  const [providers, setProviders] = useState<{ key: string; name: string }[]>([]);

  useEffect(() => {
    void setupStatus().then((s) => s.required && router.replace('/setup'));
  }, [router]);

  useEffect(() => {
    const sp = new URLSearchParams(window.location.search);
    const n = sp.get('next') || '';
    if (n.startsWith('/') && !n.startsWith('//')) setNext(n);
    const e = sp.get('error');
    if (e) setErr(e);
    api
      .get<{ providers: { key: string; name: string }[] }>('/api/auth/sso/providers')
      .then((r) => setProviders(r.providers))
      .catch(() => {});
  }, []);

  // Аль хэдийн нэвтэрсэн хүнээс дахин нууц үг нэхэхгүй. Хаагдсан
  // байгууллагатай хэрэглэгчид shell нь хаагдсан дэлгэц үзүүлдэг тул
  // давталт үүсэхгүй.
  useEffect(() => {
    api
      .get('/api/me')
      .then(() => window.location.assign(next))
      .catch(() => {});
  }, [next]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setBusy(true);
    try {
      await api.post('/api/login', { email, password });
      // /api/... (OIDC authorize) руу буцах бол бүтэн navigation.
      if (next.startsWith('/api/')) window.location.assign(next);
      else router.replace(next);
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : t('Алдаа гарлаа'));
      setBusy(false);
    }
  };

  return (
    <AuthCard
      title={t('Нэвтрэх')}
      subtitle={t('nexus-mini ажлын талбар')}
      footer={
        <>
          {t('Бүртгэлгүй юу?')}{' '}
          <Link href="/signup" className="text-accent hover:underline">
            {t('Байгууллагаа бүртгүүлэх')}
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
          type="email"
          label={t('Имэйл')}
          autoComplete="username"
          value={email}
          autoFocus
          required
          onChange={(e) => setEmail(e.target.value)}
        />
        <Input
          type="password"
          label={t('Нууц үг')}
          autoComplete="current-password"
          value={password}
          required
          onChange={(e) => setPassword(e.target.value)}
        />
        <Button type="submit" className="w-full" loading={busy}>
          {t('Нэвтрэх')}
        </Button>
      </form>

      {providers.length > 0 && (
        <div className="mt-6 space-y-3">
          <div className="flex items-center gap-3">
            <Separator className="flex-1" />
            <span className="text-foreground-subtle text-xs">{t('эсвэл')}</span>
            <Separator className="flex-1" />
          </div>
          {providers.map((p) => (
            <Button key={p.key} variant="secondary" className="w-full" asChild>
              <a href={`/api/auth/sso/${p.key}/start?next=${encodeURIComponent(next)}`}>
                {p.key === 'google' ? t('Google-ээр нэвтрэх') : `${p.name} — ${t('SSO-оор нэвтрэх')}`}
              </a>
            </Button>
          ))}
        </div>
      )}
    </AuthCard>
  );
}
