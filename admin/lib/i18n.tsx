"use client";

// Админ аппын хэлний дэд бүтэц — porталын lib/i18n.tsx-тэй ижил зарчим:
// түлхүүр нь монгол текст, шинэ хэл = нэг толь.

import { useEffect, useState } from "react";

export type Locale = "mn" | "en";
export const locales: { code: Locale; label: string }[] = [
  { code: "mn", label: "MN" },
  { code: "en", label: "EN" },
];

export function getLocale(): Locale {
  if (typeof window === "undefined") return "mn";
  const l = localStorage.getItem("nexus_locale");
  return locales.some((x) => x.code === l) ? (l as Locale) : "mn";
}

export function setLocale(l: Locale) {
  localStorage.setItem("nexus_locale", l);
  window.location.reload();
}

const en: Record<string, string> = {
  "Платформын админ": "Platform admin",
  "nexus-mini удирдлагын систем": "nexus-mini management console",
  "Энэ систем зөвхөн платформын админд зориулагдсан": "This console is for platform admins only",
  "Имэйл": "Email",
  "Нууц үг": "Password",
  "Нэвтрэх": "Sign in",
  "Гарах": "Log out",
  "Тойм": "Overview",
  "Байгууллагууд": "Organizations",
  "Хэрэглэгчид": "Users",
  "Каталог": "Catalog",
  "Профайл": "Profile",
  "Платформын ерөнхий үзүүлэлтүүд": "Platform-wide metrics",
  "Байгууллага": "Organization",
  "Хэрэглэгч": "Users",
  "Бэлэн апп": "Available apps",
  "Суулгалт": "Installs",
  "Платформ дээрх бүх байгууллага": "All organizations on the platform",
  "Нэр": "Name",
  "Гишүүд": "Members",
  "Аппууд": "Apps",
  "Үүссэн": "Created",
  "Платформ дээрх бүх бүртгэлтэй хэрэглэгч": "All registered users on the platform",
  "Бүртгүүлсэн": "Registered",
  "платформ админ": "platform admin",
  "App store-ийн бүх апп, суулгалтын тоо": "Every app in the store with install counts",
  "Апп": "App",
  "Хувилбар": "Version",
  "Бинарид": "In binary",
  "Тийм": "Yes",
  "Үгүй": "No",
  "Бүх байгууллагын сүүлийн үйлдлүүд": "Recent actions across all organizations",
  "Үйлдэл": "Action",
  "Объект": "Object",
  "Хэн": "Who",
  "Хэзээ": "When",
  "систем": "system",
  "Ерөнхий мэдээлэл": "General",
  "Хадгалах": "Save",
  "Хадгалагдлаа": "Saved",
  "Нууц үг солих": "Change password",
  "Одоогийн нууц үг": "Current password",
  "Шинэ нууц үг (8+)": "New password (8+)",
  "Шинэ нууц үг (давталт)": "New password (repeat)",
  "Солих": "Change",
  "Нууц үг солигдлоо — бусад төхөөрөмжийн нэвтрэлт хаагдсан": "Password changed — sessions on other devices were closed",
  "Шинэ нууц үг давталттайгаа таарахгүй байна": "New password doesn't match its repeat",
  "Алдаа гарлаа": "Something went wrong",
  "Нэр хоосон байж болохгүй": "Name can't be empty",
};

const dicts: Record<Locale, Record<string, string>> = { mn: {}, en };

export function useT() {
  const [locale, setLoc] = useState<Locale>("mn");
  useEffect(() => setLoc(getLocale()), []);
  const t = (s: string) => (locale === "mn" ? s : dicts[locale][s] ?? s);
  return { t, locale };
}
