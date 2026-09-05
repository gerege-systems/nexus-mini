// Админ панелийн статик шалгалт: i18n бүрэн байдал, давхардсан түлхүүр,
// hydration эрсдэл. (Portal-ийнхтэй ижил дүрэм, тусдаа апп тул тусдаа.)
import { readFileSync, readdirSync, statSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const root = resolve(dirname(fileURLToPath(import.meta.url)), "..");
let failures = 0;
const fail = (m) => { console.error("✗ " + m); failures++; };
const ok = (m) => console.log("✓ " + m);

function walk(dir, out = []) {
  for (const name of readdirSync(dir)) {
    if (name === "node_modules" || name.startsWith(".next")) continue;
    const p = join(dir, name);
    if (statSync(p).isDirectory()) walk(p, out);
    else if (/\.(tsx|ts)$/.test(p)) out.push(p);
  }
  return out;
}

const dict = readFileSync(join(root, "lib/i18n.tsx"), "utf8");
const keys = new Set();
const dupes = [];
for (const m of dict.matchAll(/^\s{2,4}"((?:[^"\\]|\\.)+)":/gm)) {
  if (keys.has(m[1])) dupes.push(m[1]);
  keys.add(m[1]);
}
if (dupes.length) fail(`i18n давхардсан түлхүүр: ${dupes.slice(0, 5).join(" | ")}`);

const sources = [...walk(join(root, "app")), ...walk(join(root, "components"))];
const used = new Set();
for (const f of sources) {
  for (const m of readFileSync(f, "utf8").matchAll(/(^|[^.\w])t\("((?:[^"\\]|\\.)+)"\)/g)) used.add(m[2]);
}
const missing = [...used].filter((k) => !keys.has(k));
if (missing.length) fail(`i18n дутуу орчуулга (${missing.length}): ${missing.slice(0, 5).map((s) => s.slice(0, 40)).join(" | ")}`);
else ok(`i18n: ${used.size} түлхүүр ашиглагдсан, бүгд ${keys.size} толинд бий`);

const stripStrings = (line) => line.replace(/"(?:[^"\\]|\\.)*"/g, '""').replace(/'(?:[^'\\]|\\.)*'/g, "''");
const risky = [];
for (const f of sources) {
  const lines = readFileSync(f, "utf8").split("\n");
  let inEffect = 0, inTemplate = false;
  lines.forEach((line, i) => {
    const ticks = (line.match(/`/g) || []).length;
    const was = inTemplate;
    if (ticks % 2 === 1) inTemplate = !inTemplate;
    if (was || inTemplate) return;
    const code = stripStrings(line);
    if (/useEffect\(|on[A-Z]\w*=|=>\s*{|function /.test(code)) inEffect = 8; // onX= = JSX handler prop
    if (/(matchMedia|localStorage|sessionStorage|window\.(location|innerWidth|navigator))/.test(code) &&
        inEffect === 0 && !/typeof window/.test(code)) risky.push(`admin/${f.replace(root + "/", "")}:${i + 1}`);
    if (inEffect > 0) inEffect--;
  });
}
if (risky.length) fail(`render дотор браузерын API: ${risky.slice(0, 5).join(" | ")}`);
else ok("hydration: render дотор браузерын API уншилт олдсонгүй");

process.exit(failures ? 1 : 0);
