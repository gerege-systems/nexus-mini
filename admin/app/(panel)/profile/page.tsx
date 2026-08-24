"use client";

import { useEffect, useState } from "react";
import { api, ApiError, type Me } from "@/lib/api";
import { toast } from "@/lib/toast";
import { useT } from "@/lib/i18n";

export default function ProfilePage() {
  const { t } = useT();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [pw, setPw] = useState({ current_password: "", new_password: "", confirm: "" });
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void api.get<Me>("/api/me").then((m) => {
      setName(m.user.name);
      setEmail(m.user.email);
    });
  }, []);

  const saveName = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await api.put("/api/me", { name });
      toast(t("Хадгалагдлаа"));
    } catch (ex) {
      toast(ex instanceof ApiError ? t(ex.message) : t("Алдаа гарлаа"), "err");
    }
  };

  const savePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    if (pw.new_password !== pw.confirm) {
      toast(t("Шинэ нууц үг давталттайгаа таарахгүй байна"), "err");
      return;
    }
    setBusy(true);
    try {
      await api.post("/api/me/password", {
        current_password: pw.current_password,
        new_password: pw.new_password,
      });
      setPw({ current_password: "", new_password: "", confirm: "" });
      toast(t("Нууц үг солигдлоо — бусад төхөөрөмжийн нэвтрэлт хаагдсан"));
    } catch (ex) {
      toast(ex instanceof ApiError ? t(ex.message) : t("Алдаа гарлаа"), "err");
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>{t("Профайл")}</h1>
          <div className="sub">{email}</div>
        </div>
      </div>

      <div className="card card__pad" style={{ maxWidth: 480 }}>
        <b style={{ display: "block", marginBottom: "0.9rem" }}>{t("Ерөнхий мэдээлэл")}</b>
        <form onSubmit={saveName}>
          <div className="field">
            <label>{t("Нэр")}</label>
            <input value={name} required onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="field">
            <label>{t("Имэйл")}</label>
            <input value={email} disabled style={{ opacity: 0.6 }} />
          </div>
          <button className="btn">{t("Хадгалах")}</button>
        </form>
      </div>

      <div className="card card__pad" style={{ maxWidth: 480, marginTop: "1rem" }}>
        <b style={{ display: "block", marginBottom: "0.9rem" }}>{t("Нууц үг солих")}</b>
        <form onSubmit={savePassword}>
          <div className="field">
            <label>{t("Одоогийн нууц үг")}</label>
            <input type="password" value={pw.current_password} required
              onChange={(e) => setPw({ ...pw, current_password: e.target.value })} />
          </div>
          <div className="field">
            <label>{t("Шинэ нууц үг")}</label>
            <input type="password" value={pw.new_password} required minLength={8}
              onChange={(e) => setPw({ ...pw, new_password: e.target.value })} />
            <div className="hint">{t("8+ тэмдэгт: латин үсэг, тоо, тусгай тэмдэгт (кирилл хориотой)")}</div>
          </div>
          <div className="field">
            <label>{t("Шинэ нууц үг (давталт)")}</label>
            <input type="password" value={pw.confirm} required
              onChange={(e) => setPw({ ...pw, confirm: e.target.value })} />
          </div>
          <button className="btn" disabled={busy}>{t("Солих")}</button>
        </form>
      </div>
    </>
  );
}
