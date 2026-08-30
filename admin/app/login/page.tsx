'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { Alert, Button, Card, Input } from '@craftzbay/ui';
import { api, ApiError, type Me } from '@/lib/api';
import { useT } from '@/lib/i18n';

// Зөвхөн платформын админ нэвтэрнэ — жирийн хэрэглэгчийг буцаана.
export default function AdminLoginPage() {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);
  const router = useRouter();
  const { t } = useT();

  useEffect(() => {
    api
      .get<Me>('/api/me')
      .then((m) => {
        if (m.user.platform_admin) router.replace('/');
      })
      .catch(() => {});
  }, [router]);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setBusy(true);
    try {
      await api.post('/api/login', { email, password });
      const me = await api.get<Me>('/api/me');
      if (!me.user.platform_admin) {
        await api.post('/api/logout');
        setErr(t('Энэ систем зөвхөн платформын админд зориулагдсан'));
        setBusy(false);
        return;
      }
      router.replace('/');
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : t('Алдаа гарлаа'));
      setBusy(false);
    }
  };

  return (
    <main className="bg-background-subtle flex min-h-dvh items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <div className="mb-6 flex items-center gap-3">
          <span className="bg-accent text-accent-foreground grid size-9 shrink-0 place-items-center rounded-md font-semibold">
            N
          </span>
          <div className="min-w-0">
            <h1 className="text-foreground text-base font-semibold">{t('Платформын админ')}</h1>
            <p className="text-foreground-muted truncate text-sm">
              {t('nexus-mini удирдлагын систем')}
            </p>
          </div>
        </div>

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
      </Card>
    </main>
  );
}
