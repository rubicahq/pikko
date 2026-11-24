import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  // Never expose the component preview in production (PRD §14).
  if (
    request.nextUrl.pathname.startsWith("/dev/") &&
    (process.env.NODE_ENV === "production" || process.env.NEXT_PUBLIC_APP_ENV === "production")
  ) {
    return NextResponse.rewrite(new URL("/404", request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: ["/dev/:path*"],
};
