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

// Нийтийн хуудас (landing /, /apps, /developers, /login, /signup), API,
// Next-ийн дотоод зам, статик файлаас БУСАД бүгд хамгаалалттай — модулийн
// шинэ route нэмэгдэхэд энд гар хүрэх шаардлагагүй. (Модулийн ShortID нь
// эдгээр нийтийн нэртэй давхцахыг Register хориглоно.)
export const config = {
  matcher: ["/((?!api|_next|login|signup|apps|developers|favicon\\.ico|$).*)"],
};
