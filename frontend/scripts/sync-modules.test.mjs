// scripts/sync-modules.mjs-ийн хамгаалалтууд (node --test, хамааралгүй):
// нөөцөлсөн short_id цөмийн хуудсыг устгахгүй, зам репогоос гарахгүй,
// symlink/сервер код хуулагдахгүй, давхардал барина, толь нэгдэнэ.
import { test } from "node:test";
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import { mkdirSync, mkdtempSync, readFileSync, symlinkSync, writeFileSync, existsSync, cpSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const frontendRoot = resolve(here, "..");

// sandbox — frontend-ийн хуулбар дээр (жинхэнэ репод хүрэхгүй) sync ажиллуулна.
function sandbox(modulesJSON, files = {}) {
  // Тест бүр өөрийн үндэстэй: модулийн файлууд root-ийн ХАЖУУД (../mod) —
  // /tmp-д хуваалцвал өмнөх тестийн үлдэгдэл (symlink г.м.) хальдана.
  const base = mkdtempSync(join(tmpdir(), "syncmod-"));
  const root = join(base, "frontend");
  mkdirSync(join(root, "scripts"), { recursive: true });
  mkdirSync(join(root, "app", "(portal)", "dashboard"), { recursive: true });
  mkdirSync(join(root, "lib"), { recursive: true });
  cpSync(join(frontendRoot, "scripts", "sync-modules.mjs"), join(root, "scripts", "sync-modules.mjs"));
  writeFileSync(join(root, "app", "(portal)", "dashboard", "page.tsx"), "// цөмийн хуудас\n");
  writeFileSync(join(root, "modules.json"), JSON.stringify(modulesJSON, null, 2));
  for (const [p, body] of Object.entries(files)) {
    const full = join(root, p);
    mkdirSync(dirname(full), { recursive: true });
    writeFileSync(full, body);
  }
  return root;
}

// run — stdout + stderr хоёуланг буцаана (console.warn нь stderr).
function run(root) {
  try {
    return execFileSync(process.execPath, [join(root, "scripts", "sync-modules.mjs")],
      { cwd: root, encoding: "utf8", stdio: ["ignore", "pipe", "pipe"] });
  } catch (e) {
    e.message += "\n" + (e.stderr || "");
    throw e;
  }
}

// runAll — stdout+stderr нийлүүлж авах (алгассан файлын мэдэгдэл шалгахад).
function runAll(root) {
  const r = execFileSync(process.execPath, ["-e", `
    const { execFileSync } = require("node:child_process");
    try { process.stdout.write(execFileSync(process.execPath, [${JSON.stringify(join(root, "scripts", "sync-modules.mjs"))}],
      { cwd: ${JSON.stringify(root)}, encoding: "utf8", stdio: ["ignore","pipe","pipe"] })); } catch (e) { process.stdout.write(String(e.stderr||"")); }
  `], { encoding: "utf8" });
  return r;
}

const moduleFiles = (name = "mod") => ({
  [`../${name}/ui/pages/page.tsx`]: "export default function P(){return null}\n",
  [`../${name}/ui/pages/sub/page.tsx`]: "export default function S(){return null}\n",
  [`../${name}/ui/i18n.ts`]: 'const i18n = { en: { "Сайн": "Hi" } };\nexport default i18n;\n',
});

test("модулийн UI + толь хуулагдана, хавтас нь gitignore-той", () => {
  const root = sandbox([{ short_id: "devices", ui: "../mod/ui" }], moduleFiles());
  const out = run(root);
  assert.match(out, /module ui: devices/);
  assert.ok(existsSync(join(root, "app", "(portal)", "devices", "page.tsx")));
  assert.ok(existsSync(join(root, "app", "(portal)", "devices", "sub", "page.tsx")));
  assert.equal(readFileSync(join(root, "app", "(portal)", "devices", ".gitignore"), "utf8").trim(), "*");
  const gen = readFileSync(join(root, "lib", "i18n.modules.ts"), "utf8");
  assert.match(gen, /import m0 from ".\/modules\/devices.i18n"/);
  assert.match(gen, /export default merged/);
  // Цөмийн хуудас хэвээр.
  assert.ok(existsSync(join(root, "app", "(portal)", "dashboard", "page.tsx")));
});

test("нөөцөлсөн short_id цөмийн хуудсыг устгахгүй", () => {
  for (const id of ["dashboard", "members", "settings", "api", "core"]) {
    const root = sandbox([{ short_id: id, ui: "../mod/ui" }], moduleFiles());
    assert.throws(() => run(root), /нөөцөлсөн|буруу/, `${id} нэвтрэв`);
    assert.ok(existsSync(join(root, "app", "(portal)", "dashboard", "page.tsx")), `${id}: цөмийн хуудас устсан`);
  }
});

test("буруу short_id ба ui зам татгалзана", () => {
  assert.throws(() => run(sandbox([{ short_id: "Буруу", ui: "../mod/ui" }], moduleFiles())), /буруу/);
  assert.throws(() => run(sandbox([{ short_id: "a", ui: "../mod/ui" }], moduleFiles())), /буруу/); // хэт богино
  assert.throws(() => run(sandbox([{ short_id: "devices" }], moduleFiles())), /ui зам/);
  // Репогоос гадуур зам.
  assert.throws(() => run(sandbox([{ short_id: "devices", ui: "/etc" }], moduleFiles())), /репогоос гадуур/);
});

test("давхардсан short_id татгалзана", () => {
  const root = sandbox([{ short_id: "devices", ui: "../mod/ui" }, { short_id: "devices", ui: "../mod/ui" }], moduleFiles());
  assert.throws(() => run(root), /давхардсан/);
});

test("зөвхөн page/layout/loading/error .tsx хуулагдана", () => {
  const files = moduleFiles();
  files["../mod/ui/pages/route.ts"] = "export async function GET(){}\n"; // сервер код
  files["../mod/ui/pages/secret.txt"] = "нууц\n";
  files["../mod/ui/pages/layout.tsx"] = "export default function L(){return null}\n";
  const root = sandbox([{ short_id: "devices", ui: "../mod/ui" }], files);
  run(root);
  // Алгассан тухай мэдэгдэл stderr-т очдог — файлын үр дүнгээр шалгана.
  assert.ok(existsSync(join(root, "app", "(portal)", "devices", "layout.tsx")));
  assert.ok(!existsSync(join(root, "app", "(portal)", "devices", "route.ts")), "route.ts хуулагдав");
  assert.ok(!existsSync(join(root, "app", "(portal)", "devices", "secret.txt")));
});

test("symlink хуулагдахгүй", () => {
  const root = sandbox([{ short_id: "devices", ui: "../mod/ui" }], moduleFiles());
  try {
    symlinkSync("/etc/passwd", join(root, "..", "mod", "ui", "pages", "link.tsx"));
  } catch {
    return; // symlink дэмжигдэхгүй орчин
  }
  assert.throws(() => run(root), /symlink/);
});

test("хоёр модулийн толь тус тусдаа alias-тай", () => {
  const files = { ...moduleFiles("mod"), ...moduleFiles("mod2") };
  const root = sandbox([{ short_id: "devices", ui: "../mod/ui" }, { short_id: "organisation", ui: "../mod2/ui" }], files);
  run(root);
  const gen = readFileSync(join(root, "lib", "i18n.modules.ts"), "utf8");
  assert.match(gen, /import m0 from ".\/modules\/devices.i18n"/);
  assert.match(gen, /import m1 from ".\/modules\/organisation.i18n"/);
  rmSync(root, { recursive: true, force: true });
});
