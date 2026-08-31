import { NextResponse, type NextRequest } from "next/server";

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
  // Session cookie-гүй зочинд админ аппын ямар ч хуудас (login-ээс бусад)
  // харагдахгүй. Жинхэнэ эрхийн шалгалт API талд.
  if (req.nextUrl.pathname !== "/login" && !req.cookies.has("nexus_session")) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
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

// Статик файл (цэгтэй зам), Next-ийн дотоод зам, API-аас БУСАД бүгд.
export const config = {
  matcher: ["/((?!api|_next/static|_next/image|.*\\.).*)"],
};
