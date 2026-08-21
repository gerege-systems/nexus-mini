// Модулиудын UI-г portal-д нэгтгэнэ (prebuild/predev). Цөмийн файлд гар
// хүрэхгүй тул дистрибуц цөмийн frontend-ийг шинэчлэхэд мөргөлдөөн гарахгүй.
//
// modules.json: [{ "short_id": "devices", "ui": "../backend/apps/devices/ui" }]
// ui/pages/**   → app/(portal)/<short_id>/**   (Next route-ууд)
// ui/i18n.ts    → lib/modules/<short_id>.i18n.ts, lib/i18n.modules.ts (нэгтгэсэн)
import { cpSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const list = JSON.parse(readFileSync(join(root, "modules.json"), "utf8"));
const modDir = join(root, "lib", "modules");
rmSync(modDir, { recursive: true, force: true });
mkdirSync(modDir, { recursive: true });

const imports = [];
for (const m of list) {
  if (!/^[a-z][a-z0-9_]*$/.test(m.short_id)) throw new Error(`modules.json: буруу short_id ${m.short_id}`);
  const ui = resolve(root, m.ui);
  const pages = join(ui, "pages");
  const target = join(root, "app", "(portal)", m.short_id);
  rmSync(target, { recursive: true, force: true });
  if (existsSync(pages)) cpSync(pages, target, { recursive: true });
  const i18n = join(ui, "i18n.ts");
  if (existsSync(i18n)) {
    cpSync(i18n, join(modDir, `${m.short_id}.i18n.ts`));
    imports.push(m.short_id);
  }
  console.log(`module ui: ${m.short_id} ← ${m.ui}${existsSync(pages) ? "" : " (pages байхгүй)"}`);
}

const gen = [
  "// Үүсгэсэн файл — scripts/sync-modules.mjs. Гараар засахгүй.",
  ...imports.map((id) => `import ${id} from "./modules/${id}.i18n";`),
  "",
  "const merged: Record<string, Record<string, string>> = {};",
  ...imports.map((id) => `for (const [l, d] of Object.entries(${id})) merged[l] = { ...(merged[l] ?? {}), ...d };`),
  "",
  "export default merged;",
  "",
].join("\n");
writeFileSync(join(root, "lib", "i18n.modules.ts"), gen);
