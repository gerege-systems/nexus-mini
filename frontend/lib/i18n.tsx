"use client";

// Хэлний дэд бүтэц. Түлхүүр нь МОНГОЛ текст өөрөө — кодод t("Дашбоард")
// гэж бичихэд л болно, орчуулга нь энд нэг дор төвлөрнө. Шинэ хэл нэмэхэд:
// locales-д код нэмээд dicts-д толь нэмнэ — өөр юу ч өөрчлөхгүй.

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
  // ─── Нийтлэг ───
  "Дашбоард": "Dashboard",
  "Апп дэлгүүр": "App store",
  "Гишүүд": "Members",
  "Эрхийн тохиргоо": "Access control",
  "Audit лог": "Audit log",
  "Нэвтрэх": "Sign in",
  "Бүртгүүлэх": "Sign up",
  "Гарах": "Log out",
  "Хадгалах": "Save",
  "Болих": "Cancel",
  "Нэр": "Name",
  "Имэйл": "Email",
  "Нууц үг": "Password",
  "Хайх…": "Search…",
  "Алдаа гарлаа": "Something went wrong",
  "Цэс": "Menu",
  "Удирдлага": "Administration",
  "Байгууллага": "Organization",
  "Тохиргоо": "Preferences",
  "Загвар": "Theme",
  "Хэл": "Language",
  "Байгууллага нэмэх": "Add organization",
  "Хадгалагдлаа": "Saved",
  "Устгагдлаа": "Deleted",
  "Нүүр": "Home",
  "Модуль хөгжүүлэх": "Build a module",

  // ─── Landing: нүүр ───
  "НЭЭЛТТЭЙ ЭХ": "OPEN SOURCE",
  "Үйл ажиллагааны нэгдсэн дижитал платформ": "Integrated digital operations platform",
  "Байгууллагын үйлчилгээ, үйл ажиллагаа, систем, өгөгдлийг нэг дор холбодог модульт платформ — цөм нь суурийг, апп дэлгүүр нь боломжуудыг нь өгнө.":
    "A modular platform that connects an organization's services, operations, systems and data — the core provides the foundation, the app store delivers the capabilities.",
  "Байгууллагаа бүртгүүлэх": "Register your organization",
  "ЦӨМ": "CORE",
  "Платформ юу хариуцдаг вэ": "What the platform takes care of",
  "Модуль бүр дахин бичдэг байсан зүйлс нэг л удаа, цөмд:": "The things every module used to re-implement, done once, in the core:",
  "Tenant тусгаарлалт": "Tenant isolation",
  "Байгууллага бүрийн өгөгдөл PostgreSQL Row-Level Security-ээр DB давхаргад тусгаарлагдана — кодын алдаа ч хана даван харагдуулахгүй.":
    "Each organization's data is isolated at the database layer with PostgreSQL Row-Level Security — even a code bug can't leak across the wall.",
  "Модуль permission-оо тунхаглаад л болоо: суулгахад role-уудад автоматаар оноогдоно. «Зөвхөн өөрийн бүртгэл» scope, role-ийн өвлөлт дэмжинэ.":
    "A module simply declares its permissions: on install they are granted to roles automatically. \"Own records only\" scope and role inheritance are supported.",
  "Audit гинж": "Audit chain",
  "Бүх чухал үйлдэл append-only, hash chain-тэй бүртгэлд ордог — гар хүрвэл гинж тасарч илэрнэ. Нэг товчоор шалгана.":
    "Every important action lands in an append-only, hash-chained log — tampering breaks the chain and shows up. Verified with one click.",
  "Нэвтрэлт ба SSO": "Auth & SSO",
  "Session auth өнөөдөр; OIDC provider + өөр nexus-mini-тэй federation дараагийн үед ирнэ.":
    "Session auth today; OIDC provider and federation with other nexus-mini instances arrive in the next phase.",
  "Circuit breaker, load shedding, retry — гадаад системтэй холбогддог модулиудад бэлэн хэрэгсэл (үе 4).":
    "Circuit breaker, load shedding, retry — ready-made tools for modules that talk to external systems (phase 4).",
  "Нэг бинари": "One binary",
  "Модулиуд Go кодоор нэг бинарид компиллогдоно — микросервисийн төвөгггүй, сүлжээний нэмэлт дуудлагагүй.":
    "Modules compile into a single Go binary — no microservice overhead, no extra network hops.",
  "Байгууллага бүр өөрт хэрэгтэй модулиа сонгож суулгана — суусан апп эрх, цэсээ өөрөө авчирна. Одоо байгаа аппуудыг тайлбартай нь үзэх.":
    "Each organization installs only what it needs — an installed app brings its own permissions and menu. Browse the current apps with descriptions.",
  "Модуль бол долоон метод хэрэгжүүлсэн Go package. Файлын бүтэц, permission, миграц, route — бүрэн гарын авлага.":
    "A module is a Go package implementing seven methods. File layout, permissions, migrations, routes — the full guide.",
  "Өөрөө ажиллуулж үзэх үү?": "Want to run it yourself?",
  "env-ээ бөглөөд": "fill in your env, then",
  "Эсвэл": "Or",
  "Эсвэл эндээ бүртгүүлэх": "Or sign up right here",

  // ─── Landing: апп дэлгүүр ───
  "Байгууллага бүр өөрт хэрэгтэй модулиа л суулгана. Суусан апп нь permission-уудаа role-уудад тунхагласан ёсоор оноож, цэсээ эрхтэй хүнд л харуулна; унтраавал бүгд эргэж алга болно.":
    "Each organization installs only the modules it needs. An installed app grants its permissions to roles as declared and shows its menu only to entitled users; disable it and everything disappears again.",
  "Дэлгүүрээс сонгоно": "Pick from the store",
  "Каталогоос аппаа сонгоод «Суулгах» — хамаарлуудыг нь платформ өөрөө цэгцэлнэ.":
    "Pick an app from the catalog and hit Install — the platform resolves its dependencies.",
  "Эрх автоматаар": "Permissions, automatically",
  "Аппын permission-ууд role-уудад тунхагласан ёсоороо оноогдоно; админ дараа нь чөлөөтэй өөрчилнө.":
    "The app's permissions are granted to roles as declared; the admin can adjust them freely afterwards.",
  "Цэс гарч ирнэ": "The menu appears",
  "Эрхтэй хэрэглэгчид л аппын цэсийг харна. Rail дээр аппын icon нэмэгдэж, өөрийн цэстэйгээ ирнэ.":
    "Only entitled users see the app's menu. Its icon appears on the rail with its own submenu.",
  "Одоо байгаа аппууд": "Available apps",
  "Тайлбар оруулаагүй.": "No description yet.",
  "Бэлэн": "Ready",
  "Каталогт бүртгэлтэй": "Listed in catalog",
  "Каталог хоосон байна.": "The catalog is empty.",
  "Өөрийн модулиа энд гаргамаар байна уу?": "Want your module listed here?",
  "Гарын авлагыг дагаад модулиа бичээд каталогт PR илгээгээрэй.":
    "Follow the guide, build your module and send a PR to the catalog.",
  "Модуль хөгжүүлэх заавар": "Module development guide",

  // ─── Auth хуудсууд ───
  "nexus-mini ажлын талбар": "nexus-mini workspace",
  "Имэйл эсвэл нууц үг буруу": "Wrong email or password",
  "Бүртгэлгүй юу?": "No account yet?",
  "Байгууллагаа бүртгүүлэх ": "Register your organization",
  "Бүртгэлтэй юу?": "Already have an account?",
  "Бүртгүүлмэгц app store-оос модулиа сонгоно": "Right after sign-up you pick your modules from the app store",
  "Таны нэр": "Your name",
  "Нууц үг (8+)": "Password (8+ chars)",
  "Байгууллагын нэр": "Organization name",
  "Богино нэр (slug)": "Short name (slug)",
  "Жижиг латин үсэг, тоо, зураас": "Lowercase latin letters, digits, dashes",
  "Байгууллага үүсгэх": "Create organization",
  "Ажлын талбараа үүсгээд store-оос модулиа сонгоно": "Create your workspace, then pick modules from the store",
  "Үүсгэх": "Create",

  // ─── Portal ───
  "Сайн байна уу,": "Welcome,",
  "Эхлэхэд туслах": "Getting started",
  "Апп дэлгүүрээс модуль суулгах": "Install a module from the app store",
  "Байгууллагад тань хэрэгтэй модулиудыг сонгож суулгана": "Pick and install the modules your organization needs",
  "Гишүүдээ урих": "Invite your members",
  "Ажилтнуудаа нэмээд role оноогоорой": "Add your staff and assign roles",
  "Role бүрийн permission-ийг өөрийн бүтцэд тааруулна": "Tune each role's permissions to your structure",
  "Идэвхтэй апп": "Active apps",
  "Таны эрх": "Your permissions",
  "Суусан аппууд": "Installed apps",
  "Байгууллагадаа хэрэгтэй модулиудыг суулгана": "Install the modules your organization needs",
  "Суусан": "Installed",
  "Унтраах": "Disable",
  "Унтраасан": "Disabled",
  "Асаах": "Enable",
  "Суулгах": "Install",
  "Суулгаагүй": "Not installed",
  "Бинарид ороогүй": "Not in this binary",
  "суулгагдлаа": "installed",
  "Асаалаа": "Enabled",
  "Унтраалаа": "Disabled",
  "Байгууллагын гишүүд ба role оноолт": "Organization members and role assignment",
  "Гишүүн нэмэх": "Add member",
  "Гишүүн алга": "No members",
  "Бүртгэлтэй имэйл бол шууд нэгдэнэ, нэр/нууц үг хэрэггүй": "If the email is already registered they join directly — no name/password needed",
  "Нэр (шинэ хэрэглэгчид)": "Name (for new users)",
  "Түр нууц үг (шинэ хэрэглэгчид, 8+)": "Temporary password (for new users, 8+)",
  "Нэмэх": "Add",
  "Гишүүн нэмэгдлээ": "Member added",
  "Role шинэчлэгдлээ": "Roles updated",
  "Гишүүн хасагдлаа": "Member removed",
  "Өөрийн role-г эндээс өөрчлөхгүй": "You can't change your own roles here",
  "хасах уу?": "— remove?",
  "Нүд дарж — → бүгд → өөрийн гэж эргэлдэнэ. Role нь implies-ээрээ доод role-ийн эрхийг өвлөнө.":
    "Click a cell to cycle — → all → own. A role inherits the lower role's permissions via implies.",
  "Role нэмэх": "Add role",
  "Бүгд": "All",
  "Өөрийн": "Own",
  "Админ үргэлж бүх эрхтэй": "Admin always has every permission",
  "Оноолт хадгалагдлаа": "Grants saved",
  "Role үүслээ": "Role created",
  "Код": "Code",
  "Жижиг үсэг, тоо, _": "Lowercase letters, digits, _",
  "Өвлөх role (сонголттой)": "Inherit from role (optional)",
  "— өвлөхгүй —": "— no inheritance —",
  "«Өөрийн» = зөвхөн өөрийн үүсгэсэн бүртгэл дээр үйлдэл хийнэ (модуль нь дэмждэг бол)":
    "\"Own\" = act only on records you created (when the module supports it)",
  "Append-only, hash гинжтэй үйлдлийн бүртгэл": "Append-only, hash-chained action log",
  "Гинж шалгах": "Verify chain",
  "Гинж бүрэн — бүртгэлд гар хүрээгүй": "Chain intact — the log has not been tampered with",
  "дээр тасарсан!": "is broken!",
  "Үйлдэл": "Action",
  "Объект": "Object",
  "Хэн": "Who",
  "Хэзээ": "When",
  "систем": "system",

  // ─── Devices модуль (portal хуудас) ───
  "Төхөөрөмжүүд": "Devices",
  "Байгууллагын төхөөрөмжийн бүртгэл": "Your organization's device registry",
  "Бүртгэх": "Register",
  "Бүртгэл хоосон": "No records yet",
  "Эхний төхөөрөмжөө бүртгээрэй": "Register your first device",
  "Төрөл": "Type",
  "Сериал": "Serial",
  "Статус": "Status",
  "Бүртгэсэн": "Registered by",
  "Ашиглагдаж байгаа": "In use",
  "Засварт": "In repair",
  "Алдагдсан": "Lost",
  "Хассан": "Retired",
  "Төхөөрөмж засах": "Edit device",
  "Төхөөрөмж бүртгэх": "Register device",
  "Тэмдэглэл": "Note",
  "Бүртгэгдлээ": "Registered",
  "засах": "edit",
  "устгах": "delete",
  "төхөөрөмжийг устгах уу?": "— delete this device?",

  // ─── Roles/нийтлэг нэмэлт ───
  "Платформ": "Platform",

  // ─── Landing: хөгжүүлэгч ───
  "Модуль бол": "A module is a Go package implementing the",
  "interface-ийг хэрэгжүүлсэн Go package.": "interface.",
  "Tenant тусгаарлалт, нэвтрэлт, суулгалт, RBAC оноолт, audit — платформ хийнэ; та бизнес логикоо л бичнэ. Хамгийн сайн заавар бол ажиллаж байгаа жишээ —":
    "Tenant isolation, auth, installation, RBAC grants, audit — the platform handles them; you write only business logic. The best guide is the working example —",
  "Хэн юу хариуцдаг вэ": "Who is responsible for what",
  "МОДУЛЬ": "MODULE",
  "ПЛАТФОРМ": "PLATFORM",
  "Permission-оо тунхаглана": "Declares its permissions",
  "Tenant тусгаарлалт (RLS)": "Tenant isolation (RLS)",
  "Цэсээ зарлана": "Declares its menu",
  "Нэвтрэлт, session": "Auth, sessions",
  "Route-уудаа бүртгэнэ": "Registers its routes",
  "Суулгалт, хамаарлын шийдэл": "Installation, dependency resolution",
  "Өөрийн хүснэгт, миграц": "Its own tables and migrations",
  "RBAC default оноолт, шалгалт": "RBAC default grants, enforcement",
  "Бизнес логик": "Business logic",
  "Audit гинж, app store": "Audit chain, app store",
  "Файлын бүтэц": "File layout",
  "Жижиг модуль нэг файлаас эхэлж болно; өсөхөөрөө ингэж хуваана:": "A small module can start as one file; as it grows, split it like this:",
  "Алхамууд": "Steps",
  "1. Package үүсгэх": "1. Create the package",
  "2. Permission тунхаглах": "2. Declare permissions",
  "Дүрмүүд (зөрчвөл бинари асахгүй):": "Rules (the binary refuses to boot on violation):",
  "3. Миграц": "3. Migrations",
  "4. Route-ууд": "4. Routes",
  "5. Цэс": "5. Menu",
  "6. Бүртгэх ба асаах": "6. Register and run",
  "7. Store-д нийтлэх": "7. Publish to the store",
  "Тест": "Testing",
  "Бэлэн үү?": "Ready?",
  "devices-ийг хуулж эхлээд, дуусаад каталогт PR илгээгээрэй.": "Start by copying devices; when done, send a PR to the catalog.",
  "Markdown хувилбар": "Markdown version",
  "Апп дэлгүүр үзэх": "Browse the app store",
};

const dicts: Record<Locale, Record<string, string>> = { mn: {}, en };

export function useT() {
  // SSR/эхний render үргэлж mn — hydration зөрөхөөс сэргийлж mount-ын
  // дараа локалоо уншина.
  const [locale, setLoc] = useState<Locale>("mn");
  useEffect(() => setLoc(getLocale()), []);
  const t = (s: string) => (locale === "mn" ? s : dicts[locale][s] ?? s);
  return { t, locale };
}
