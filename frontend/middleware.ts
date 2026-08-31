import { NextResponse, type NextRequest } from "next/server";

// Нийтийн хуудас (landing /, /apps, /developers, /login, /signup) — бусад
// бүх зам хамгаалалттай, модулийн шинэ route нэмэгдэхэд энд гар хүрэхгүй.
// (Модулийн ShortID нь эдгээр нийтийн нэртэй давхцахыг Register хориглоно.)
const PUBLIC = /^\/(?:login|signup|apps|developers)(?:\/|$)/;

// Хатуу CSP. script-src нь nonce + strict-dynamic — Next өөрийн inline
// (flight) script-үүддээ nonce-ыг ХҮСЭЛТИЙН Content-Security-Policy
// толгойгоос уншиж тавина; layout дахь theme script нь x-nonce-оос авна.
// style-src-т 'unsafe-inline' үлдэнэ: React-ийн style атрибутыг nonce-оор
// хамгаалах боломжгүй. Гадаад эх сурвалж огт байхгүй тул бусад нь 'self'.
function policy(nonce: string): string {
  const dev = process.env.NODE_ENV !== "production";
  return [
    "default-src 'self'",
    `script-src 'self' 'nonce-${nonce}' 'strict-dynamic'${dev ? " 'unsafe-eval'" : ""}`,
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data:",
    "font-src 'self'",
    "connect-src 'self'",
    "form-action 'self'",
    "frame-ancestors 'none'",
    "frame-src 'none'",
    "base-uri 'self'",
    "object-src 'none'",
    ...(dev ? [] : ["upgrade-insecure-requests"]),
  ].join("; ");
}

function newNonce(): string {
  return btoa(String.fromCharCode(...crypto.getRandomValues(new Uint8Array(16))));
}

export function middleware(req: NextRequest) {
  const path = req.nextUrl.pathname;

  // Session cookie огт байхгүй зочныг хамгаалалттай хуудаснаас сервер талд
  // шууд /login руу — client гацахаас өмнө blank flash гарахгүй, portal-ийн
  // bundle нэвтрээгүй хүнд татагдахгүй. Cookie-гийн ХҮЧИНТЭЙГ энд шалгахгүй
  // (httpOnly token-ийг edge дээр батлах боломжгүй) — жинхэнэ шалгалт API-д.
  if (path !== "/" && !PUBLIC.test(path) && !req.cookies.has("nexus_session")) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.search = "?next=" + encodeURIComponent(path + req.nextUrl.search);
    return NextResponse.redirect(url);
  }

  // CSP зөвхөн баримт (document) хүсэлтэд. RSC/prefetch хариу нь баримт биш
  // бөгөөд түүнд nonce тавибал client-д кэшлэгдэж хуучирна.
  if (req.headers.get("rsc")) return NextResponse.next();

  const nonce = newNonce();
  const csp = policy(nonce);
  const headers = new Headers(req.headers);
  headers.set("x-nonce", nonce);
  headers.set("content-security-policy", csp);

  const res = NextResponse.next({ request: { headers } });
  res.headers.set("content-security-policy", csp);
  return res;
}

// Статик файл (цэгтэй зам), Next-ийн дотоод зам, API-аас БУСАД бүгд —
// нийтийн хуудсууд ч CSP авах ёстой тул matcher нь тэдгээрийг оруулна.
export const config = {
  matcher: ["/((?!api|_next/static|_next/image|.*\\.).*)"],
};
