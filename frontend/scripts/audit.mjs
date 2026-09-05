// Frontend-ийн статик шалгалт (build-ээс тусдаа, хамааралгүй):
//   1. i18n бүрэн байдал — t("…") бүр EN толинд байгаа эсэх, давхардсан түлхүүр
//   2. middleware-ийн matcher — хамгаалалттай/нийтийн замууд зөв ялгагдаж байгаа
//   3. hydration эрсдэл — render дотор window/localStorage/matchMedia уншилт
// Ажиллуулах: node scripts/audit.mjs   (make check-web)
import { readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repo = resolve(root, "..");
let failures = 0;
const fail = (msg) => { console.error("✗ " + msg); failures++; };
const okMsg = (msg) => console.log("✓ " + msg);

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (name === "node_modules" || name === ".next" || name.startsWith(".next")) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (/\.(tsx|ts)$/.test(p)) out.push(p);
  }
  return out;
}

// ─── 1. i18n ───────────────────────────────────────────────────────────
const dictFiles = [join(root, "lib/i18n.tsx")];
for (const m of JSON.parse(readFileSync(join(root, "modules.json"), "utf8"))) {
  const p = resolve(root, m.ui, "i18n.ts");
  try { statSync(p); dictFiles.push(p); } catch {}
}
const keyRe = /^\s{2,4}"((?:[^"\\]|\\.)+)":/gm;
const keys = new Set();
const dupes = [];
for (const f of dictFiles) {
  const src = readFileSync(f, "utf8");
  const seen = new Set();
  for (const m of src.matchAll(keyRe)) {
    if (seen.has(m[1])) dupes.push(`${f.replace(repo + "/", "")}: ${m[1]}`);
    seen.add(m[1]);
    keys.add(m[1]);
  }
}
if (dupes.length) fail(`i18n давхардсан түлхүүр: ${dupes.slice(0, 5).join(" | ")}`);
const sources = [...walk(join(root, "app")), ...walk(join(root, "components")),
  ...JSON.parse(readFileSync(join(root, "modules.json"), "utf8")).flatMap((m) => {
    try { return walk(resolve(root, m.ui)); } catch { return []; }
  })];
const used = new Set();
for (const f of sources) {
  const src = readFileSync(f, "utf8");
  // t("...") — .get("...") зэрэг худал эерэгийг хасна
  for (const m of src.matchAll(/(^|[^.\w])t\("((?:[^"\\]|\\.)+)"\)/g)) used.add(m[2]);
}
const missing = [...used].filter((k) => !keys.has(k));
if (missing.length) fail(`i18n дутуу орчуулга (${missing.length}): ${missing.slice(0, 5).map((m) => m.slice(0, 40)).join(" | ")}`);
else okMsg(`i18n: ${used.size} түлхүүр ашиглагдсан, бүгд ${keys.size} толинд бий`);

// ─── 2. middleware: matcher + нийтийн зам ───────────────────────────
// Matcher нь CSP-ийн тулд бүх хуудсыг (нийтийн ч) хамрах ёстой; статик
// файл, _next, /api-г алгасна. Нэвтрээгүй зочныг /login руу шилжүүлэх
// шалгалт нь PUBLIC regex + "/" — хамгаалалттай зам түүнд таарах ёсгүй.
const mw = readFileSync(join(root, "middleware.ts"), "utf8");
const matcher = mw.match(/matcher:\s*\[\s*"([^"]+)"/)?.[1];
const publicSrc = mw.match(/const PUBLIC = \/(.+)\/;/)?.[1];
if (!matcher || !publicSrc) fail("middleware matcher / PUBLIC олдсонгүй");
else {
  const re = new RegExp("^" + matcher.replace(/\\\\/g, "\\") + "$");
  const pub = new RegExp(publicSrc);
  const isPublic = (p) => p === "/" || pub.test(p);
  const guarded = ["/dashboard", "/dashboard/x", "/store", "/members", "/roles", "/audit",
    "/settings", "/org/new", "/sso-clients", "/oauth/consent", "/devices", "/organisation/people",
    "/loginx", "/appsx"];
  const publicPaths = ["/", "/apps", "/apps/x", "/developers", "/login", "/signup"];
  const skipped = ["/api/me", "/_next/static/x.js", "/favicon.ico", "/robots.txt", "/icon.png"];
  const bad = [];
  for (const p of guarded) {
    if (!re.test(p)) bad.push(`matcher-т ороогүй: ${p}`);
    if (isPublic(p)) bad.push(`хамгаалагдаагүй (PUBLIC-т таарав): ${p}`);
  }
  for (const p of publicPaths) {
    if (!re.test(p)) bad.push(`CSP-гүй (matcher-т ороогүй): ${p}`);
    if (!isPublic(p)) bad.push(`нийтийн зам хаагдсан: ${p}`);
  }
  for (const p of skipped) if (re.test(p)) bad.push(`статик/API matcher-т орсон: ${p}`);
  if (bad.length) fail(`middleware: ${bad.join(" | ")}`);
  else okMsg(`middleware: ${guarded.length} хамгаалалттай, ${publicPaths.length} нийтийн, ${skipped.length} алгассан зам зөв`);
}

// ─── 3. hydration эрсдэл ───────────────────────────────────────────────
const risky = [];
// Мөрөөс тэмдэгт мөрийн литералуудыг хасна — тайлбар/текст доторх дурдалт
// (жишээ нь гарын авлагын өгүүлбэр) худал эерэг өгөхгүй. Template literal
// (`...`) доторх блокийг бүхэлд нь алгасна: layout-ийн theme script нь
// hydration-аас ӨМНӨ ажилладаг зөв загвар.
const stripStrings = (line) => line.replace(/"(?:[^"\\]|\\.)*"/g, '""').replace(/'(?:[^'\\]|\\.)*'/g, "''");
for (const f of sources) {
  const src = readFileSync(f, "utf8");
  const lines = src.split("\n");
  let inEffect = 0;
  let inTemplate = false;
  lines.forEach((line, i) => {
    const ticks = (line.match(/`/g) || []).length;
    const wasTemplate = inTemplate;
    if (ticks % 2 === 1) inTemplate = !inTemplate;
    if (wasTemplate || inTemplate) return;
    const code = stripStrings(line);
    if (/useEffect\(|useLayoutEffect\(|on[A-Z]\w*=|=>\s*{|function /.test(code)) inEffect = 8; // onX= = JSX handler prop
    const risk = /(matchMedia|localStorage|sessionStorage|window\.(location|innerWidth|navigator))/.test(code);
    if (risk && inEffect === 0 && !/typeof window/.test(code)) {
      risky.push(`${f.replace(repo + "/", "")}:${i + 1}`);
    }
    if (inEffect > 0) inEffect--;
  });
}
if (risky.length) fail(`render дотор браузерын API (hydration эрсдэл): ${risky.slice(0, 5).join(" | ")}`);
else okMsg("hydration: render дотор браузерын API уншилт олдсонгүй");

process.exit(failures ? 1 : 0);
