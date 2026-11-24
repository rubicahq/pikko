import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

function applySecurityHeaders(res: NextResponse) {
  res.headers.set("X-Content-Type-Options", "nosniff");
  res.headers.set("X-Frame-Options", "DENY");
  res.headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  res.headers.set(
    "Permissions-Policy",
    "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
  );
  res.headers.set("X-DNS-Prefetch-Control", "on");
  const appEnv = process.env.NEXT_PUBLIC_APP_ENV || process.env.NODE_ENV;
  if (appEnv === "production") {
    res.headers.set(
      "Strict-Transport-Security",
      "max-age=31536000; includeSubDomains",
    );
  }
  return res;
}

export function middleware(request: NextRequest) {
  // Never expose the component preview in production (PRD §14).
  if (
    request.nextUrl.pathname.startsWith("/dev/") &&
    (process.env.NODE_ENV === "production" || process.env.NEXT_PUBLIC_APP_ENV === "production")
  ) {
    return applySecurityHeaders(NextResponse.rewrite(new URL("/404", request.url)));
  }
  return applySecurityHeaders(NextResponse.next());
}

export const config = {
  matcher: [
    /*
     * Apply to all paths except Next static assets and image optimizer.
     */
    "/((?!_next/static|_next/image|favicon.ico).*)",
  ],
};
