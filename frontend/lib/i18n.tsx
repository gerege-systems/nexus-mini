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
  "Байгууллагын үйлчилгээ, үйл ажиллагаа, систем, өгөгдлийг нэг дор холбодог модульт платформ — цөм нь суурийг, апп дэлгүүр нь боломжуудыг өгнө.":
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
  "Имэйлээр хайна: бүртгэлтэй бол нэр нь гарна, үгүй бол шинээр үүсгэнэ": "Looked up by email: registered users show their name, otherwise a new account is created",
  "Хайж байна…": "Looking up…",
  "Аль хэдийн энэ байгууллагын гишүүн": "Already a member of this organization",
  "Бүртгэлтэй хэрэглэгч — role өгөөд нэмнэ": "Registered user — pick a role and add",
  "Бүртгэлгүй — нэр, түр нууц үг өгч шинээр үүсгэнэ": "Not registered — give a name and a temporary password to create the account",
  "Түр нууц үг (8+)": "Temporary password (8+)",
  "Платформын админ энэ хэрэглэгчийн нэрийн өмнөөс нэвтэрсэн байна — бүх үйлдэл audit-д тэмдэглэгдэнэ (30 минутын session).":
    "A platform admin is signed in on behalf of this user — every action is recorded in the audit log (30-minute session).",
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
  "миграц: цөм ok · devices ok": "migrations: core ok · devices ok",
  "платформын админ үүслээ": "platform admin created",
  "модуль": "module",
  "нэг бинари": "one binary",
  "Тусгаарлалт DB давхаргад": "Isolation at the database layer",
  "Байгууллага бүрийн өгөгдөл PostgreSQL Row-Level Security-ээр тусгаарлагдана — кодын алдаа ч хана даван харагдуулахгүй.":
    "Each organization's data is isolated with PostgreSQL Row-Level Security — even a code bug can't leak across the wall.",
  "Эрх тунхаглалаар, бүртгэл гинжээр": "Permissions by declaration, records by chain",
  "Permission суулгах үед role-уудад автоматаар оноогдоно; бүх чухал үйлдэл hash chain-тэй audit бүртгэлд үлдэнэ.":
    "Permissions are granted to roles automatically on install; every important action lands in a hash-chained audit log.",
  "7 метод = таны модуль": "7 methods = your module",
  "Go interface хэрэгжүүлээд каталогт PR илгээхэд л таны модуль store-д — нэг бинари, микросервисийн төвөггүй.":
    "Implement the Go interface and send a catalog PR — your module is in the store. One binary, no microservice overhead.",

  // ─── Серверийн алдааны мессежүүд (клиент талд орчуулна) ───
  "имэйл эсвэл нууц үг буруу": "Wrong email or password",
  "бүх талбарыг зөв бөглөнө үү (нууц үг 8+)": "Please fill in all fields correctly (password 8+)",
  "имэйл эсвэл байгууллагын slug бүртгэлтэй байна": "This email or organization slug is already registered",
  "бүртгэл амжилтгүй боллоо": "Sign-up failed",
  "slug давхардаж байна": "This slug is already taken",
  "нэр ба slug шаардлагатай": "Name and slug are required",
  "энэ байгууллагын гишүүн биш": "You are not a member of this organization",
  "апп олдсонгүй": "App not found",
  "энэ апп бинарид ороогүй байна — `nexus-mini add` коммандаар нэмээд дахин build хийнэ": "This app is not in the binary — add it with `nexus-mini add` and rebuild",
  "суулгалт амжилтгүй боллоо": "Installation failed",
  "code давхардаж байна (эсвэл формат буруу)": "Code already exists (or has invalid format)",
  "өвлөх role олдсонгүй": "The role to inherit from was not found",
  "оноолт хадгалагдсангүй (permission код зөв үү?)": "Grants were not saved (is the permission code valid?)",
  "role олдсонгүй": "Role not found",
  "зөв имэйл шаардлагатай": "A valid email is required",
  "шинэ хэрэглэгчид нэр ба 8+ тэмдэгт түр нууц үг өгнө": "New users need a name and a temporary password (8+)",
  "хэрэглэгч үүсгэж чадсангүй": "Could not create the user",
  "үл мэдэх role код байна": "Unknown role code",
  "байгууллагад дор хаяж нэг админ үлдэх ёстой": "The organization must keep at least one admin",
  "гишүүн олдсонгүй": "Member not found",
  "гишүүн олдсонгүй (өөрийгөө хасаж болохгүй)": "Member not found (you can't remove yourself)",
  "нэр хоосон байж болохгүй": "Name can't be empty",
  "шинэ нууц үг 8+ тэмдэгт байх ёстой": "New password must be 8+ characters",
  "одоогийн нууц үг буруу": "Current password is wrong",
  "солиход алдаа гарлаа": "Failed to change password",
  "нэр, сериал шаардлагатай; статус буруу": "Name and serial are required; invalid status",
  "сериал давхардаж байна": "This serial already exists",
  "олдсонгүй эсвэл таны бүртгэл биш": "Not found, or not your record",
  "хадгалж чадсангүй": "Could not save",

  // ─── UI жижиг орхигдлууд ───
  "Цайвар": "Light",
  "Бараан": "Dark",
  "Систем": "System",
  "Агуулахын ажилтан": "Warehouse staff",

  // ─── Хөгжүүлэгчийн гарын авлага (бүрэн) ───
  "модулийн ГЭРЭЭ: ID, permission, цэс, миграц, route↔permission холболт": "the module CONTRACT: ID, permissions, menu, migrations, route↔permission wiring",
  "хүсэлт/хариултын struct + validation": "request/response structs + validation",
  "HTTP handler-ууд (нэг resource = нэг файл)": "HTTP handlers (one resource = one file)",
  "модулийн goose миграцууд": "the module's goose migrations",
  "// reverse-DNS, глобал давтагдашгүй": "// reverse-DNS, globally unique",
  "// permission prefix + URL зам": "// permission prefix + URL path",
  "Хүний нэр": "Human name",
  "Код заавал": "The code must start with",
  "-ээр эхэлнэ — өөр модулийн эрхийг булааж чадахгүй": " — a module can't claim another module's permissions",
  "нь суулгах үед хэн авахыг тунхагладаг:": "declares who receives it on install:",
  "үргэлж бүгдийг авна, жагсаалтад бичсэн нь нэмж авна,": "always gets everything, listed roles get it in addition,",
  "нь зөвхөн өөрийн мөрийн эрх": "grants it with own-records-only scope",
  "хоосон = зөвхөн admin (аюулгүй default)": "empty = admin only (safe default)",
  "зэрэг нэрс нөөцлөгдсөн": "and similar names are reserved",
  "RLS policy": "RLS policy",
  "жишээг devices-ээс хуул": "copy the example from devices",
  "ашиглах бол": "requires a",
  "багана заавал": "column",
  "Бүх string баганад урттай хязгаар (varchar(n)) — задгай text хориотой": "Every string column needs a length cap (varchar(n)) — bare text is forbidden",
  "Төгсгөлд нь": "Finish with",
  "Модуль бүр өөрийн goose хүснэгттэй": "Each module has its own goose table",
  "тул цөм болон бусад модультай мөргөлдөхгүй.": "so it never collides with the core or other modules.",
  "Танд өгөгдөх": "The router",
  "нь аль хэдийн хамгаалагдсан:": "you receive is already protected: it lives under",
  "дор байрладаг, нэвтрээгүй хүн 401, апп суулгаагүй tenant 403 авчихсан байдаг. Handler дотор:": "— unauthenticated callers already got 401 and tenants without the app got 403. Inside a handler:",
  "хүсэлтийн identity": "the request identity",
  "бол query-дээ": "means adding a",
  "шүүлт нэм": "filter to your query",
  "RLS context автоматаар тохирдог холболт; SQL-даа": "a connection with RLS context set automatically; still write",
  "гэж бас бич": "in your SQL",
  "чухал үйлдлээ audit гинжид бич": "record important actions to the audit chain",
  "вэб туслахууд": "web helpers",
  "Монгол нэр": "Mongolian name",
  "6. UI хуудас (portal)": "6. UI page (portal)",
  "Цэсэндээ зарласан Path-тайгаа ижил замд Next.js хуудас үүсгэнэ — devices-ийн хуудас": "Create a Next.js page at the same path you declared in your menu — the devices page",
  "бэлэн загвар нь.": "is a ready-made template.",
  "// frontend/app/(portal)/name/page.tsx — Path: \"/name\"-тэй ижил": "// frontend/app/(portal)/name/page.tsx — same as Path: \"/name\"",
  "// хэрэглэгч + permissions": "// user + permissions",
  "// хэл (mn/en)": "// language (mn/en)",
  "// undefined | \"all\" | \"own\"": "// undefined | \"all\" | \"own\"",
  "// api.get(`/api/apps/name/`) — cookie автоматаар, 401 бол login руу": "// api.get(`/api/apps/name/`) — cookie attached automatically, 401 redirects to login",
  "Эрхээр UI-гаа нуу:": "Hide UI by permission:",
  "байхгүй бол товчоо бүү харуул (энэ нь UX — жинхэнэ хамгаалалт серверт)": "missing → don't render the button (this is UX — real enforcement is server-side)",
  "«Өөрийн» scope-той хэрэглэгчид засах/устгах товчийг": "For own-scope users, show edit/delete buttons only when",
  "үед л харуулна": "",
  "Цэсний icon нэрээ": "Add your menu icon name to the map in",
  "ийн map-д нэм (lucide icon)": "(a lucide icon)",
  "Бэлэн загварууд:": "Ready-made styles:",
  "амжилтад": "use",
  "текстэд": "for success and",
  "7. Бүртгэх ба асаах": "7. Register and run",
  "// backend/apps/apps.go — нэг мөр:": "// backend/apps/apps.go — one line:",
  "# модуль store-д гарч ирнэ": "# the module appears in the store",
  "8. Store-д нийтлэх": "8. Publish to the store",
  "-д бүртгэлээ нэмээд PR илгээнэ. Үе 2-т төв registry +": " — add your entry and send a PR. In phase 2 a central registry and the",
  "CLI ирэхэд go_module замаар тань шууд татдаг болно.": "CLI will fetch it directly via your go_module path.",
  "SQL parse/encode бүх логикт unit тест бич.": "Write unit tests for all SQL/parse/encode logic.",
  "нь linux cross-build + vet + test + SDK-ийн хилийн шалгалт (модуль internal/* импортолбол унадаг) — push бүрийн өмнө заавал.": "runs a linux cross-build + vet + tests + the SDK boundary check (importing internal/* fails) — required before every push.",

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
