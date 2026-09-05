'use client';

// Админ түр нууц үгтэй үүсгэсэн данс: сервер tenant-ийн бүх route-ыг 403
// (password_change_required) өгдөг тул ажлын мужийн оронд энэ л харагдана.
// Сольсны дараа Shell дахин ачаална.

import { useState } from 'react';
import { Alert, Button, Input } from '@gerege-systems/ui';
import { AuthCard } from '@/components/auth-card';
import { api, ApiError, type Me } from '@/lib/api';
import { useT } from '@/lib/i18n';

export function PasswordGate({ me, onDone }: { me: Me; onDone: () => void }) {
  const { t } = useT();
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [again, setAgain] = useState('');
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    if (next !== again) {
      setErr(t('Шинэ нууц үг хоёр талд таарахгүй байна'));
      return;
    }
    setBusy(true);
    try {
      await api.post('/api/me/password', { current_password: current, new_password: next });
      onDone();
    } catch (ex) {
      setErr(ex instanceof ApiError ? ex.message : t('Алдаа гарлаа'));
      setBusy(false);
    }
  };

  return (
    <AuthCard
      title={t('Нууц үгээ солино уу')}
      subtitle={me.user.email}
      footer={
        <Button variant="ghost" size="sm" onClick={() => api.post('/api/logout').finally(() => window.location.assign('/login'))}>
          {t('Гарах')}
        </Button>
      }
    >
      <p className="text-foreground-muted mb-4 text-sm">
        {t('Админ танд түр нууц үг өгсөн. Үргэлжлүүлэхийн өмнө зөвхөн өөрт мэдэгдэх нууц үг тавина уу.')}
      </p>
      {err && (
        <Alert variant="danger" className="mb-4">
          {err}
        </Alert>
      )}
      <form onSubmit={submit} className="space-y-4">
        <Input
          type="password"
          label={t('Түр нууц үг')}
          autoComplete="current-password"
          value={current}
          autoFocus
          required
          onChange={(e) => setCurrent(e.target.value)}
        />
        <Input
          type="password"
          label={t('Шинэ нууц үг')}
          autoComplete="new-password"
          value={next}
          required
          minLength={10}
          onChange={(e) => setNext(e.target.value)}
        />
        <Input
          type="password"
          label={t('Шинэ нууц үг (давтах)')}
          autoComplete="new-password"
          value={again}
          required
          onChange={(e) => setAgain(e.target.value)}
        />
        <Button type="submit" className="w-full" loading={busy}>
          {t('Солих')}
        </Button>
      </form>
    </AuthCard>
  );
}
