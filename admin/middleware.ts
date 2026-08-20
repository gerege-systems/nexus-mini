import { NextResponse, type NextRequest } from "next/server";

// Session cookie-гүй зочинд админ аппын ямар ч хуудас (login-ээс бусад)
// харагдахгүй. Жинхэнэ эрхийн шалгалт API талд.
export function middleware(req: NextRequest) {
  if (!req.cookies.has("nexus_session")) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    return NextResponse.redirect(url);
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/", "/tenants/:path*", "/users/:path*", "/apps/:path*", "/audit/:path*", "/profile/:path*"],
};
