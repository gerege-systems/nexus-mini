"use client";

// Глобал toast — provider plumbing-гүй, CustomEvent-ээр. toast("Хадгалагдлаа")
// гэж дуудахад root layout-д суусан <Toaster/> харуулна.

export type ToastKind = "ok" | "err";

export function toast(message: string, kind: ToastKind = "ok") {
  window.dispatchEvent(new CustomEvent("nexus:toast", { detail: { message, kind } }));
}
