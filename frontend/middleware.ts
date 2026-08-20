import { NextResponse, type NextRequest } from "next/server";

// Session cookie огт байхгүй зочныг хамгаалалттай хуудаснаас сервер талд
// шууд /login руу — client гacahaas өмнө blank flash гарахгүй, portal-ийн
// bundle нэвтрээгүй хүнд татагдахгүй. Cookie-гийн ХҮЧИНТЭЙГ энд шалгахгүй
// (httpOnly token-ийг edge дээр батлах боломжгүй) — жинхэнэ шалгалт API-д.
export function middleware(req: NextRequest) {
  if (!req.cookies.has("nexus_session")) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/store/:path*",
    "/devices/:path*",
    "/members/:path*",
    "/roles/:path*",
    "/audit/:path*",
    "/org/:path*",
  ],
};
