"use client";

// Анхны тохиргооны шидтэн (open-gerege-nexus-ийн /setup-ийн загвар). Эрх нь
// хаягийн мөрөн дэх токен: зөвхөн React state-д, хөтчийн хадгалалтад хэзээ ч
// биш (хуудас хаагдмагц үгүй болох ёстой). Сервер 401 биш 404 өгдөг тул 404 =
// токен хуучирсан гэж үзэж токен асуух дэлгэц рүү буцна.

import { useEffect, useState } from "react";
import Link from "next/link";
import { Alert, Button, Input, Separator } from "@gerege-systems/ui";
import { AuthCard } from "@/components/auth-card";
import { api, ApiError } from "@/lib/api";
import { setupStatus, type SetupStatus } from "@/lib/setup";
import { useT } from "@/lib/i18n";

type Done = { user_id: string; tenant_id?: string; tenant_error?: string };

export default function SetupPage() {
  const { t } = useT();
  const [status, setStatus] = useState<SetupStatus | null>(null);
  const [token, setToken] = useState("");
  const [typed, setTyped] = useState("");
  const [addressRead, setAddressRead] = useState(false);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [again, setAgain] = useState("");
  const [orgName, setOrgName] = useState("");
  const [orgSlug, setOrgSlug] = useState("");
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState<Done | null>(null);

  useEffect(() => {
    setToken(new URLSearchParams(window.location.search).get("token") || "");
    setAddressRead(true);
    void setupStatus().then(setStatus);
  }, []);

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setErr("");
    if (password !== again) {
      setErr(t("Шинэ нууц үг хоёр талд таарахгүй байна"));
      return;
    }
    setBusy(true);
    try {
      const out = await api.post<Done>(
        "/api/setup/complete",
        { admin: { email, name, password }, organisation: { name: orgName, slug: orgSlug } },
        { "X-Setup-Token": token },
      );
      setDone(out);
    } catch (ex) {
      if (ex instanceof ApiError && ex.status === 404) {
        setToken("");
        setErr(t("Энэ холбоос хүчингүй болсон — API-ийн логоос шинэ холбоосыг ав."));
      } else {
        setErr(ex instanceof ApiError ? ex.message : t("Алдаа гарлаа"));
      }
      setBusy(false);
    }
  };

  const footer = (
    <Link href="/login" className="text-accent hover:underline">
      {t("Нэвтрэх")}
    </Link>
  );

  if (!status || !addressRead) {
    return <AuthCard title={t("Анхны тохиргоо")}>{null}</AuthCard>;
  }
  if (done) {
    return (
      <AuthCard title={t("Бэлэн боллоо")} subtitle={email} footer={footer}>
        <p className="text-foreground-muted text-sm">
          {t("Платформын админ үүслээ. Одоо нэвтэрч аппаа store-оос суулга.")}
        </p>
        {done.tenant_error && (
          <Alert variant="warning" className="mt-4">
            {t("Байгууллага үүсгэж чадсангүй:")} {done.tenant_error} — {t("нэвтэрсний дараа үүсгэнэ.")}
          </Alert>
        )}
      </AuthCard>
    );
  }
  if (!status.required) {
    return (
      <AuthCard title={t("Анхны тохиргоо")} footer={footer}>
        <p className="text-foreground-muted text-sm">{t("Энэ суулгац аль хэдийн тохируулагдсан.")}</p>
      </AuthCard>
    );
  }
  if (!status.armed) {
    return (
      <AuthCard title={t("Анхны тохиргоо")} footer={footer}>
        <Alert variant="warning">
          {t("Шидтэн зэвсэглээгүй: API-г дахин асаагаад логоос setup холбоосыг ав, эсвэл nexus-mini.env-д ADMIN_* бичээд make migrate ажиллуул.")}
        </Alert>
      </AuthCard>
    );
  }
  if (!token) {
    return (
      <AuthCard title={t("Анхны тохиргоо")} subtitle={t("API-ийн логт хэвлэгдсэн токен")} footer={footer}>
        {err && (
          <Alert variant="danger" className="mb-4">
            {err}
          </Alert>
        )}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            setToken(typed.trim());
          }}
          className="space-y-4"
        >
          <Input label={t("Setup токен")} value={typed} autoFocus required onChange={(e) => setTyped(e.target.value)} />
          <Button type="submit" className="w-full">
            {t("Үргэлжлүүлэх")}
          </Button>
        </form>
      </AuthCard>
    );
  }
  return (
    <AuthCard
      title={t("Анхны тохиргоо")}
      subtitle={t("Платформын анхны админ ба (заавал биш) анхны байгууллага")}
      footer={footer}
    >
      {err && (
        <Alert variant="danger" className="mb-4">
          {err}
        </Alert>
      )}
      <form onSubmit={submit} className="space-y-4">
        <Input type="email" label={t("Имэйл")} autoComplete="username" value={email} autoFocus required onChange={(e) => setEmail(e.target.value)} />
        <Input label={t("Нэр")} value={name} required maxLength={120} onChange={(e) => setName(e.target.value)} />
        <Input type="password" label={t("Нууц үг")} autoComplete="new-password" value={password} required minLength={8} onChange={(e) => setPassword(e.target.value)} />
        <Input type="password" label={t("Шинэ нууц үг (давтах)")} autoComplete="new-password" value={again} required onChange={(e) => setAgain(e.target.value)} />
        <Separator />
        <Input label={t("Байгууллагын нэр")} value={orgName} maxLength={160} onChange={(e) => setOrgName(e.target.value)} />
        <Input
          label={t("Байгууллагын slug")}
          value={orgSlug}
          pattern="[a-z0-9][a-z0-9\\-]{1,62}"
          placeholder="my-org"
          onChange={(e) => setOrgSlug(e.target.value.toLowerCase())}
        />
        <Button type="submit" className="w-full" loading={busy}>
          {t("Тохируулах")}
        </Button>
      </form>
    </AuthCard>
  );
}
