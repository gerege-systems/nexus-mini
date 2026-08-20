"use client";

import { useEffect, useState } from "react";
import { api, ApiError, type Me } from "@/lib/api";
import { toast } from "@/lib/toast";

export default function ProfilePage() {
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [nameMsg, setNameMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [pw, setPw] = useState({ current_password: "", new_password: "", confirm: "" });
  const [pwMsg, setPwMsg] = useState<{ ok: boolean; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void api.get<Me>("/api/me").then((m) => {
      setName(m.user.name);
      setEmail(m.user.email);
    });
  }, []);

  const saveName = async (e: React.FormEvent) => {
    e.preventDefault();
    setNameMsg(null);
    try {
      await api.put("/api/me", { name });
      toast("Хадгалагдлаа");
    } catch (ex) {
      setNameMsg({ ok: false, text: ex instanceof ApiError ? ex.message : "Алдаа гарлаа" });
    }
  };

  const savePassword = async (e: React.FormEvent) => {
    e.preventDefault();
    setPwMsg(null);
    if (pw.new_password !== pw.confirm) {
      setPwMsg({ ok: false, text: "Шинэ нууц үг давталттайгаа таарахгүй байна" });
      return;
    }
    setBusy(true);
    try {
      await api.post("/api/me/password", {
        current_password: pw.current_password,
        new_password: pw.new_password,
      });
      setPw({ current_password: "", new_password: "", confirm: "" });
      toast("Нууц үг солигдлоо — бусад төхөөрөмжийн нэвтрэлт хаагдсан");
    } catch (ex) {
      setPwMsg({ ok: false, text: ex instanceof ApiError ? ex.message : "Алдаа гарлаа" });
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <div className="page-head">
        <div>
          <h1>Профайл</h1>
          <div className="sub">{email}</div>
        </div>
      </div>

      <div className="card card__pad" style={{ maxWidth: 480 }}>
        <b style={{ display: "block", marginBottom: "0.9rem" }}>Ерөнхий мэдээлэл</b>
        {nameMsg && <div className="alert alert--danger">{nameMsg.text}</div>}
        <form onSubmit={saveName}>
          <div className="field">
            <label>Нэр</label>
            <input value={name} required onChange={(e) => setName(e.target.value)} />
          </div>
          <div className="field">
            <label>Имэйл</label>
            <input value={email} disabled style={{ opacity: 0.6 }} />
          </div>
          <button className="btn">Хадгалах</button>
        </form>
      </div>

      <div className="card card__pad" style={{ maxWidth: 480, marginTop: "1rem" }}>
        <b style={{ display: "block", marginBottom: "0.9rem" }}>Нууц үг солих</b>
        {pwMsg && <div className="alert alert--danger">{pwMsg.text}</div>}
        <form onSubmit={savePassword}>
          <div className="field">
            <label>Одоогийн нууц үг</label>
            <input type="password" value={pw.current_password} required
              onChange={(e) => setPw({ ...pw, current_password: e.target.value })} />
          </div>
          <div className="field">
            <label>Шинэ нууц үг (8+)</label>
            <input type="password" value={pw.new_password} required minLength={8}
              onChange={(e) => setPw({ ...pw, new_password: e.target.value })} />
          </div>
          <div className="field">
            <label>Шинэ нууц үг (давталт)</label>
            <input type="password" value={pw.confirm} required
              onChange={(e) => setPw({ ...pw, confirm: e.target.value })} />
          </div>
          <button className="btn" disabled={busy}>Солих</button>
        </form>
      </div>
    </>
  );
}
