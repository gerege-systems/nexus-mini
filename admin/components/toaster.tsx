"use client";

import { useEffect, useState } from "react";
import { CheckCircle2, XCircle } from "lucide-react";
import type { ToastKind } from "@/lib/toast";

type Item = { id: number; message: string; kind: ToastKind };
let seq = 0;

export function Toaster() {
  const [items, setItems] = useState<Item[]>([]);

  useEffect(() => {
    const onToast = (e: Event) => {
      const { message, kind } = (e as CustomEvent).detail as Omit<Item, "id">;
      const id = ++seq;
      setItems((xs) => [...xs, { id, message, kind }]);
      setTimeout(() => setItems((xs) => xs.filter((x) => x.id !== id)), 3500);
    };
    window.addEventListener("nexus:toast", onToast);
    return () => window.removeEventListener("nexus:toast", onToast);
  }, []);

  if (items.length === 0) return null;
  return (
    <div className="toaster">
      {items.map((t) => (
        <div key={t.id} className={`toast${t.kind === "err" ? " toast--err" : ""}`}>
          {t.kind === "err"
            ? <XCircle size={17} style={{ color: "var(--danger)" }} />
            : <CheckCircle2 size={17} style={{ color: "var(--ok)" }} />}
          {t.message}
        </div>
      ))}
    </div>
  );
}
