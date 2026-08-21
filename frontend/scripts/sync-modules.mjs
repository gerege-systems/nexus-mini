// Модулиудын UI-г portal-д нэгтгэнэ (prebuild/predev). Цөмийн файлд гар
// хүрэхгүй тул дистрибуц цөмийн frontend-ийг шинэчлэхэд мөргөлдөөн гарахгүй.
//
// modules.json: [{ "short_id": "devices", "ui": "../backend/apps/devices/ui" }]
// ui/pages/**   → app/(portal)/<short_id>/**   (зөвхөн page/layout/loading/error .tsx)
// ui/i18n.ts    → lib/modules/<short_id>.i18n.ts, lib/i18n.modules.ts (нэгтгэсэн)
//
// Хамгаалалт: short_id нь цөмийн route нэртэй давхцахгүй (цөмийн хуудас
// устгахгүй), ui зам репогоос гадагш гарахгүй, symlink/route.ts/сервер код
// хуулахгүй, үүсгэсэн хавтас бүрт .gitignore (git-д орохгүй).
import { cpSync, existsSync, lstatSync, mkdirSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, relative, resolve, sep } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repo = resolve(root, "..");
const RESERVED = new Set(["core", "api", "admin", "platform", "store", "apps", "developers", "login",
  "signup", "dashboard", "members", "roles", "audit", "settings", "org", "_next"]);
const PAGE_FILE = /^(page|layout|loading|error|not-found)\.tsx$/;

const list = JSON.parse(readFileSync(join(root, "modules.json"), "utf8"));
if (!Array.isArray(list)) throw new Error("modules.json: массив байх ёстой");
const modDir = join(root, "lib", "modules");
rmSync(modDir, { recursive: true, force: true });
mkdirSync(modDir, { recursive: true });

function copyPages(src, dst) {
  for (const name of readdirSync(src)) {
    const s = join(src, name), d = join(dst, name);
    const st = lstatSync(s);
    if (st.isSymbolicLink()) throw new Error(`symlink хуулахгүй: ${s}`);
    if (st.isDirectory()) {
      if (!/^[a-z0-9_\[\]-]+$/.test(name)) throw new Error(`буруу хавтасны нэр: ${s}`);
      mkdirSync(d, { recursive: true });
      copyPages(s, d);
    } else if (PAGE_FILE.test(name)) {
      cpSync(s, d);
    } else {
      console.warn(`  алгасав (зөвхөн page/layout/loading/error .tsx): ${relative(root, s)}`);
    }
  }
}

const seen = new Set();
const imports = [];
list.forEach((m, idx) => {
  if (typeof m?.short_id !== "string" || !/^[a-z][a-z0-9_]{1,31}$/.test(m.short_id) || RESERVED.has(m.short_id))
    throw new Error(`modules.json: буруу/нөөцөлсөн short_id ${JSON.stringify(m?.short_id)}`);
  if (seen.has(m.short_id)) throw new Error(`modules.json: short_id давхардсан ${m.short_id}`);
  seen.add(m.short_id);
  if (typeof m.ui !== "string") throw new Error(`modules.json: ${m.short_id} ui зам байхгүй`);
  const ui = resolve(root, m.ui);
  if (!(ui === repo || ui.startsWith(repo + sep))) throw new Error(`modules.json: ${m.short_id} ui зам репогоос гадуур: ${ui}`);
  const pages = join(ui, "pages");
  const target = join(root, "app", "(portal)", m.short_id);
  rmSync(target, { recursive: true, force: true });
  if (existsSync(pages)) {
    mkdirSync(target, { recursive: true });
    copyPages(pages, target);
    writeFileSync(join(target, ".gitignore"), "*\n");
  }
  const i18n = join(ui, "i18n.ts");
  if (existsSync(i18n)) {
    cpSync(i18n, join(modDir, `${m.short_id}.i18n.ts`));
    imports.push({ id: m.short_id, alias: `m${idx}` });
  }
  console.log(`module ui: ${m.short_id} ← ${m.ui}${existsSync(pages) ? "" : " (pages байхгүй)"}`);
});

const gen = [
  "// Үүсгэсэн файл — scripts/sync-modules.mjs. Гараар засахгүй.",
  ...imports.map((m) => `import ${m.alias} from "./modules/${m.id}.i18n";`),
  "",
  "const merged: Record<string, Record<string, string>> = {};",
  ...imports.map((m) => `for (const [l, d] of Object.entries(${m.alias})) merged[l] = { ...(merged[l] ?? {}), ...d };`),
  "",
  "export default merged;",
  "",
].join("\n");
writeFileSync(join(root, "lib", "i18n.modules.ts"), gen);
