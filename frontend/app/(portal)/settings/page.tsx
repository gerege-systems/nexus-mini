"use client";

import { useEffect, useState } from "react";
import { api, ApiError } from "@/lib/api";
import { useShell } from "@/components/shell";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

type Profile = {
  name: string; slug: string; legal_name: string; registration_number: string; tax_number: string;
  address: string; phone: string; email: string; website: string;
};

const fields: { key: keyof Profile; label: string; max: number }[] = [
  { key: "name", label: "Байгууллагын нэр", max: 120 },
  { key: "legal_name", label: "Хуулийн нэр", max: 200 },
  { key: "registration_number", label: "Регистрийн дугаар", max: 32 },
  { key: "tax_number", label: "ТТД", max: 32 },
  { key: "address", label: "Хаяг", max: 500 },
  { key: "phone", label: "Утас", max: 32 },
  { key: "email", label: "Имэйл", max: 255 },
  { key: "website", label: "Вэб сайт", max: 255 },
];

export default function SettingsPage() {
  const { t } = useT();
  const { me, refresh } = useShell();
  const canEdit = !!me.permissions["core.settings.manage"];
  const [p, setP] = useState<Profile | null>(null);
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => { void api.get<Profile>("/api/tenant/profile").then(setP); }, []);

  const save = async () => {
    if (!p) return;
    setErr(""); setBusy(true);
    try {
      await api.put("/api/tenant/profile", p);
      toast(t("Хадгалагдлаа"));
      refresh();
    } catch (e) {
      setErr(e instanceof ApiError ? e.message : t("Алдаа гарлаа"));
    } finally { setBusy(false); }
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Байгууллагын тохиргоо")}</h1>
          <div className="sub">{t("Нэр, хуулийн мэдээлэл, холбоо барих")}</div>
        </div>
      </div>
      <div className="card card__pad" style={{ maxWidth: 640 }}>
        {!p ? (
          <div style={{ color: "var(--text-3)" }}>{t("Уншиж байна…")}</div>
        ) : (
          <>
            {err && <div className="alert alert--danger">{t(err)}</div>}
            <div className="field">
              <label>Slug</label>
              <input value={p.slug} disabled readOnly />
              <div className="hint">{t("Slug өөрчлөгдөхгүй")}</div>
            </div>
            {fields.map((f) => (
              <div className="field" key={f.key}>
                <label>{t(f.label)}</label>
                <input value={p[f.key]} maxLength={f.max} disabled={!canEdit}
                  onChange={(e) => setP({ ...p, [f.key]: e.target.value })} />
              </div>
            ))}
            {canEdit ? (
              <button className="btn" onClick={save} disabled={busy}>{t("Хадгалах")}</button>
            ) : (
              <div className="hint">{t("Засах эрхгүй — зөвхөн харах")}</div>
            )}
          </>
        )}
      </div>
    </>
  );
}
