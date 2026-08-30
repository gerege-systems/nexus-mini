'use client';

import { useEffect, useState } from 'react';
import { Alert, Button, Card, Icons, Spinner } from '@craftzbay/ui';
import { api, ApiError } from '@/lib/api';
import { useShell } from '@/components/shell';
import { useT } from '@/lib/i18n';

// OIDC зөвшөөрлийн хуудас: /api/oauth2/authorize энд шилжүүлнэ (query
// хэвээр). Зөвшөөрвөл сервер код гаргаж клиентийн redirect_uri руу буцаана.
type Info = { client_name: string; tenant_name: string; scopes: string[]; redirect_host: string };

const SCOPE_LABEL: Record<string, string> = {
  openid: 'Таныг таних (ID)',
  profile: 'Нэр',
  email: 'Имэйл хаяг',
  tenant: 'Байгууллагын мэдээлэл',
  roles: 'Байгууллага дахь role-ууд',
  offline_access: 'Таныг байхгүй үед ч хандах (refresh token)',
};

export default function ConsentPage() {
  const { t } = useT();
  const { me } = useShell();
  const [query, setQuery] = useState('');
  const [info, setInfo] = useState<Info | null>(null);
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    const q = window.location.search.replace(/^\?/, '');
    setQuery(q);
    let alive = true;
    api
      .get<Info>(`/api/oauth2/consent?${q}`)
      .then((d) => alive && setInfo(d))
      .catch((e) => alive && setErr(e instanceof ApiError ? e.message : 'Хүсэлт буруу'));
    return () => {
      alive = false;
    };
  }, []);

  const decide = async (approve: boolean) => {
    setBusy(true);
    try {
      const r = await api.post<{ redirect: string }>('/api/oauth2/consent', { approve, query });
      window.location.assign(r.redirect);
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t('Алдаа гарлаа'));
      setBusy(false);
    }
  };

  return (
    <Card className="mx-auto max-w-xl">
      <div className="mb-4 flex items-center gap-3">
        <span className="bg-background-muted text-foreground-muted grid size-10 shrink-0 place-items-center rounded-md">
          <Icons.Lock className="size-5" aria-hidden />
        </span>
        <h1 className="text-foreground text-lg font-semibold">{t('Хандалт зөвшөөрөх')}</h1>
      </div>

      {err && (
        <Alert variant="danger" className="mb-4">
          {err}
        </Alert>
      )}

      {!info ? (
        !err && (
          <div className="flex justify-center py-8">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        )
      ) : (
        <>
          <p className="text-foreground-muted text-sm">
            <strong className="text-foreground">{info.client_name}</strong> ({info.redirect_host}){' '}
            {t('систем таны')} <strong className="text-foreground">{info.tenant_name}</strong>{' '}
            {t('байгууллагын бүртгэлээр нэвтэрч, дараах мэдээллийг авахыг хүсэж байна:')}
          </p>

          <ul className="text-foreground-muted my-4 space-y-1.5 text-sm">
            {info.scopes.map((s) => (
              <li key={s} className="flex flex-wrap items-center gap-2">
                <Icons.Check className="text-success-text size-4 shrink-0" aria-hidden />
                {t(SCOPE_LABEL[s] ?? s)}
                <code className="text-foreground-subtle font-mono text-xs">{s}</code>
              </li>
            ))}
          </ul>

          <p className="text-foreground-subtle text-sm">
            {t('Та')}: {me.user.name} · {me.user.email}
          </p>

          <div className="mt-6 flex justify-end gap-2">
            <Button variant="secondary" disabled={busy} onClick={() => decide(false)}>
              {t('Татгалзах')}
            </Button>
            <Button loading={busy} onClick={() => decide(true)}>
              {t('Зөвшөөрөх')}
            </Button>
          </div>
        </>
      )}
    </Card>
  );
}
