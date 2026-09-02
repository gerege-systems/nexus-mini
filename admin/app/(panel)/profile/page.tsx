'use client';

import { useEffect, useState } from 'react';
import { Button, Card, CardTitle, Input, Spinner, toast } from '@gerege-systems/ui';
import { PageHead } from '@/components/states';
import { api, ApiError, type Me } from '@/lib/api';
import { useT } from '@/lib/i18n';

const fail = (e: unknown, fallback: string) =>
  toast({ title: e instanceof ApiError ? e.message : fallback, variant: 'danger' });

export default function ProfilePage() {
  const { t } = useT();
  const [loaded, setLoaded] = useState(false);
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [savingName, setSavingName] = useState(false);
  const [pw, setPw] = useState({ current_password: '', new_password: '', confirm: '' });
  const [confirmErr, setConfirmErr] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    let alive = true;
    api
      .get<Me>('/api/me')
      .then((m) => {
        if (!alive) return;
        setName(m.user.name);
        setEmail(m.user.email);
        setLoaded(true);
      })
      .catch((e) => alive && fail(e, t('Алдаа гарлаа')));
    return () => {
      alive = false;
    };
  }, [t]);

  const saveName = async (e: React.FormEvent) => {
    e.preventDefault();
    setSavingName(true);
    try {
      await api.put('/api/me', { name });
      toast({ title: t('Хадгалагдлаа'), variant: 'success' });
    } catch (ex) {
      fail(ex, t('Алдаа гарлаа'));
    } finally {
      setSavingName(false);
    }
  };

  const savePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (pw.new_password !== pw.confirm) {
      setConfirmErr(t('Шинэ нууц үг давталттайгаа таарахгүй байна'));
      return;
    }
    setConfirmErr('');
    setBusy(true);
    try {
      await api.post('/api/me/password', {
        current_password: pw.current_password,
        new_password: pw.new_password,
      });
      setPw({ current_password: '', new_password: '', confirm: '' });
      toast({
        title: t('Нууц үг солигдлоо — бусад төхөөрөмжийн нэвтрэлт хаагдсан'),
        variant: 'success',
      });
    } catch (ex) {
      fail(ex, t('Алдаа гарлаа'));
    } finally {
      setBusy(false);
    }
  };

  if (!loaded) {
    return (
      <div className="flex justify-center py-16">
        <Spinner label={t('Уншиж байна…')} />
      </div>
    );
  }

  return (
    <>
      <PageHead title={t('Профайл')} description={email} />

      <div className="max-w-lg space-y-4">
        <Card>
          <CardTitle className="mb-4">{t('Ерөнхий мэдээлэл')}</CardTitle>
          <form onSubmit={saveName} className="space-y-4">
            <Input
              label={t('Нэр')}
              value={name}
              required
              autoComplete="name"
              onChange={(e) => setName(e.target.value)}
            />
            <Input
              label={t('Имэйл')}
              value={email}
              disabled
              helperText={t('Имэйлийг энэ панелаас солих боломжгүй')}
            />
            <Button type="submit" loading={savingName}>
              {t('Хадгалах')}
            </Button>
          </form>
        </Card>

        <Card>
          <CardTitle className="mb-4">{t('Нууц үг солих')}</CardTitle>
          <form onSubmit={savePassword} className="space-y-4">
            <Input
              type="password"
              label={t('Одоогийн нууц үг')}
              autoComplete="current-password"
              value={pw.current_password}
              required
              onChange={(e) => setPw({ ...pw, current_password: e.target.value })}
            />
            <Input
              type="password"
              label={t('Шинэ нууц үг')}
              autoComplete="new-password"
              value={pw.new_password}
              required
              minLength={8}
              helperText={t('8+ тэмдэгт: латин үсэг, тоо, тусгай тэмдэгт (кирилл хориотой)')}
              onChange={(e) => setPw({ ...pw, new_password: e.target.value })}
            />
            <Input
              type="password"
              label={t('Шинэ нууц үг (давталт)')}
              autoComplete="new-password"
              value={pw.confirm}
              required
              error={confirmErr || undefined}
              onChange={(e) => {
                setPw({ ...pw, confirm: e.target.value });
                if (confirmErr) setConfirmErr('');
              }}
            />
            <Button type="submit" loading={busy}>
              {t('Солих')}
            </Button>
          </form>
        </Card>
      </div>
    </>
  );
}
