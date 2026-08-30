'use client';

import { useCallback, useEffect, useState } from 'react';
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
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
  Textarea,
  Tooltip,
  toast,
} from '@craftzbay/ui';
import { PageHead } from '@/components/states';
import { api, ApiError } from '@/lib/api';
import { useT } from '@/lib/i18n';

// OAuth2/OIDC клиентүүд — гадны систем энэ байгууллагын хэрэглэгчээр нэвтрэх.
type Client = {
  id: string;
  client_id: string;
  name: string;
  public: boolean;
  redirect_uris: string[];
  post_logout_uris: string[];
  scopes: string;
  created_at: string;
};
type Form = {
  id?: string;
  name: string;
  public: boolean;
  redirect_uris: string;
  post_logout_uris: string;
  scopes: string;
};

const EMPTY: Form = {
  name: '',
  public: false,
  redirect_uris: '',
  post_logout_uris: '',
  scopes: 'openid profile email',
};
const ALL_SCOPES = ['openid', 'profile', 'email', 'tenant', 'roles', 'offline_access'];

const msg = (e: unknown, fallback: string) => (e instanceof ApiError ? e.message : fallback);

export default function SSOClientsPage() {
  const { t } = useT();
  const [clients, setClients] = useState<Client[] | null>(null);
  const [issuer, setIssuer] = useState('');
  const [loadErr, setLoadErr] = useState('');
  const [form, setForm] = useState<Form | null>(null);
  const [secret, setSecret] = useState<{ client_id: string; client_secret: string } | null>(null);
  const [removing, setRemoving] = useState<Client | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await api.get<{ clients: Client[]; issuer: string }>('/api/sso-clients');
      setClients(r.clients);
      setIssuer(r.issuer);
    } catch (e) {
      setLoadErr(msg(e, t('Алдаа гарлаа')));
    }
  }, [t]);

  useEffect(() => {
    void load();
  }, [load]);

  const copy = (s: string) => {
    void navigator.clipboard?.writeText(s);
    toast({ title: t('Хуулагдлаа'), variant: 'success' });
  };

  return (
    <>
      <PageHead
        title={t('SSO клиентүүд')}
        description={t('Гадны систем энэ байгууллагын бүртгэлээр нэвтрэх (OpenID Connect)')}
        actions={
          <Button leadingIcon={<Icons.Plus />} onClick={() => setForm(EMPTY)}>
            {t('Клиент нэмэх')}
          </Button>
        }
      />

      <Card className="mb-4">
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="text-foreground font-medium">{t('Issuer')}:</span>
          <code className="text-foreground-muted font-mono text-xs break-all">{issuer}</code>
          <Tooltip label={t('Хуулах')}>
            <IconButton
              aria-label={t('Хуулах')}
              icon={<Icons.Copy />}
              variant="ghost"
              size="sm"
              onClick={() => copy(issuer)}
            />
          </Tooltip>
        </div>
        <p className="text-foreground-muted mt-1 text-sm break-all">
          {t('Discovery')}:{' '}
          <code className="font-mono text-xs">{issuer}/.well-known/openid-configuration</code> ·
          PKCE S256 {t('заавал')} · RS256
        </p>
      </Card>

      <Card padding="none">
        {loadErr ? (
          <div className="p-5">
            <Alert variant="danger">{loadErr}</Alert>
          </div>
        ) : clients === null ? (
          <div className="flex justify-center py-12">
            <Spinner label={t('Уншиж байна…')} />
          </div>
        ) : clients.length === 0 ? (
          <EmptyState
            icon={<Icons.Key />}
            title={t('Клиент байхгүй')}
            description={t('Гадны системээ бүртгээд client_id авна')}
          />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Нэр')}</TableHead>
                <TableHead>client_id</TableHead>
                <TableHead>{t('Төрөл')}</TableHead>
                <TableHead>Redirect</TableHead>
                <TableHead>Scope</TableHead>
                <TableHead align="right" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {clients.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-medium">{c.name}</TableCell>
                  <TableCell>
                    <span className="flex items-center gap-1">
                      <code className="text-foreground-muted font-mono text-xs">{c.client_id}</code>
                      <Tooltip label={t('Хуулах')}>
                        <IconButton
                          aria-label={t('Хуулах')}
                          icon={<Icons.Copy />}
                          variant="ghost"
                          size="sm"
                          onClick={() => copy(c.client_id)}
                        />
                      </Tooltip>
                    </span>
                  </TableCell>
                  <TableCell>
                    {c.public ? (
                      <Badge tone="neutral">public · PKCE</Badge>
                    ) : (
                      <Badge tone="accent">confidential</Badge>
                    )}
                  </TableCell>
                  <TableCell className="text-foreground-muted text-xs">
                    {c.redirect_uris.map((u) => (
                      <span key={u} className="block break-all">
                        {u}
                      </span>
                    ))}
                  </TableCell>
                  <TableCell className="text-foreground-muted text-xs">{c.scopes}</TableCell>
                  <TableCell align="right">
                    <span className="flex justify-end gap-1">
                      <Tooltip label={t('засах')}>
                        <IconButton
                          aria-label={`${c.name} — ${t('засах')}`}
                          icon={<Icons.Pencil />}
                          variant="ghost"
                          size="sm"
                          onClick={() =>
                            setForm({
                              id: c.id,
                              name: c.name,
                              public: c.public,
                              redirect_uris: c.redirect_uris.join('\n'),
                              post_logout_uris: c.post_logout_uris.join('\n'),
                              scopes: c.scopes,
                            })
                          }
                        />
                      </Tooltip>
                      <Tooltip label={t('устгах')}>
                        <IconButton
                          aria-label={`${c.name} — ${t('устгах')}`}
                          icon={<Icons.Trash2 />}
                          variant="ghost"
                          size="sm"
                          onClick={() => setRemoving(c)}
                        />
                      </Tooltip>
                    </span>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {/* Нууц үг нэг л удаа — хаахаас нааш өөр зүйл харуулахгүй */}
      <Dialog open={secret !== null} onOpenChange={(o) => !o && setSecret(null)}>
        <DialogContent size="md" showClose={false}>
          <DialogHeader>
            <DialogTitle>{t('Клиентийн нууц үг — нэг л удаа харагдана')}</DialogTitle>
          </DialogHeader>
          {secret && (
            <div className="space-y-4">
              <Input
                label="client_id"
                readOnly
                value={secret.client_id}
                onFocus={(e) => e.target.select()}
              />
              <Input
                label="client_secret"
                readOnly
                value={secret.client_secret}
                onFocus={(e) => e.target.select()}
                helperText={t('Хадгалаад хаа — дахин харуулахгүй, алдвал клиентээ шинээр үүсгэнэ.')}
              />
            </div>
          )}
          <DialogFooter>
            <Button onClick={() => setSecret(null)}>{t('Хадгалсан')}</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {form && (
        <ClientDialog
          form={form}
          onClose={() => setForm(null)}
          onSaved={(s) => {
            setForm(null);
            if (s) setSecret(s);
            void load();
          }}
        />
      )}

      <ConfirmationDialog
        open={removing !== null}
        onOpenChange={(o) => !o && setRemoving(null)}
        title={removing ? `"${removing.name}"` : ''}
        description={t('клиентийг устгах уу? Олгосон бүх токен хүчингүй болно.')}
        confirmLabel={t('устгах')}
        confirmVariant="destructive"
        onConfirm={async () => {
          if (!removing) return;
          await api.del(`/api/sso-clients/${removing.id}`);
          toast({ title: t('Устгагдлаа'), variant: 'success' });
          await load();
        }}
      />
    </>
  );
}

function ClientDialog({
  form: initial,
  onClose,
  onSaved,
}: {
  form: Form;
  onClose: () => void;
  onSaved: (secret: { client_id: string; client_secret: string } | null) => void;
}) {
  const { t } = useT();
  const [form, setForm] = useState(initial);
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  const lines = (s: string) =>
    s
      .split(/\n|,/)
      .map((x) => x.trim())
      .filter(Boolean);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr('');
    setBusy(true);
    const body = {
      name: form.name,
      public: form.public,
      redirect_uris: lines(form.redirect_uris),
      post_logout_uris: lines(form.post_logout_uris),
      scopes: form.scopes,
    };
    try {
      if (form.id) {
        await api.put(`/api/sso-clients/${form.id}`, body);
        toast({ title: t('Хадгалагдлаа'), variant: 'success' });
        onSaved(null);
      } else {
        const r = await api.post<{ client_id: string; client_secret: string }>(
          '/api/sso-clients',
          body,
        );
        toast({ title: t('Клиент үүслээ'), variant: 'success' });
        onSaved(r.client_secret ? r : null);
      }
    } catch (ex) {
      setErr(msg(ex, t('Алдаа гарлаа')));
      setBusy(false);
    }
  };

  const scopes = form.scopes.split(' ').filter(Boolean);

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent size="md">
        <DialogHeader>
          <DialogTitle>{form.id ? t('Клиент засах') : t('Клиент нэмэх')}</DialogTitle>
        </DialogHeader>

        <form onSubmit={submit} className="space-y-4">
          {err && <Alert variant="danger">{err}</Alert>}

          <Input
            label={t('Нэр')}
            value={form.name}
            autoFocus
            required
            placeholder="Bold ERP"
            onChange={(e) => setForm({ ...form, name: e.target.value })}
          />

          {!form.id && (
            <Checkbox
              checked={form.public}
              onCheckedChange={(v) => setForm({ ...form, public: v === true })}
              label={t('Public клиент (SPA/mobile — secret-гүй, PKCE)')}
            />
          )}

          <Textarea
            label={t('Redirect URI-ууд (мөр тус бүр)')}
            rows={3}
            value={form.redirect_uris}
            placeholder="https://erp.bold.mn/auth/callback"
            onChange={(e) => setForm({ ...form, redirect_uris: e.target.value })}
          />

          <Textarea
            label={t('Logout-ын дараах URI-ууд')}
            rows={2}
            value={form.post_logout_uris}
            onChange={(e) => setForm({ ...form, post_logout_uris: e.target.value })}
          />

          <div className="space-y-1.5">
            <span className="text-foreground block text-sm font-medium">Scope</span>
            <span className="flex flex-wrap gap-1.5">
              {ALL_SCOPES.map((s) => {
                const on = scopes.includes(s);
                return (
                  <Button
                    key={s}
                    type="button"
                    size="sm"
                    variant={on ? 'primary' : 'outline'}
                    aria-pressed={on}
                    onClick={() =>
                      setForm({
                        ...form,
                        scopes: (on ? scopes.filter((x) => x !== s) : [...scopes, s]).join(' '),
                      })
                    }
                  >
                    {s}
                  </Button>
                );
              })}
            </span>
          </div>

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
