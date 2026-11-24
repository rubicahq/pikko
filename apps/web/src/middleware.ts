import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { mapLegacyHost } from "@/lib/site";

function applySecurityHeaders(res: NextResponse) {
  res.headers.set("X-Content-Type-Options", "nosniff");
  res.headers.set("X-Frame-Options", "DENY");
  res.headers.set("Referrer-Policy", "strict-origin-when-cross-origin");
  res.headers.set(
    "Permissions-Policy",
    "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
  );
  res.headers.set("Cross-Origin-Resource-Policy", "same-site");
  res.headers.set("X-DNS-Prefetch-Control", "on");
  // Conservative CSP for Next App Router (SEC-08). Inline styles/scripts needed for Next runtime.
  res.headers.set(
    "Content-Security-Policy",
    [
      "default-src 'self'",
      "base-uri 'self'",
      "frame-ancestors 'none'",
      "object-src 'none'",
      "img-src 'self' data: blob: https:",
      "font-src 'self' data:",
      "style-src 'self' 'unsafe-inline'",
      "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
      "connect-src 'self' https: wss:",
      "form-action 'self'",
    ].join("; "),
  );
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
  // Permanent redirect from legacy apex/subdomains (e.g. pikko.site → pikko.id).
  // Primary cutover redirect lives in nginx; this is defense-in-depth for the web app.
  const host = request.headers.get("host") || "";
  const mapped = mapLegacyHost(host);
  if (mapped) {
    const url = request.nextUrl.clone();
    url.host = mapped;
    url.protocol = "https:";
    url.port = "";
    // Never forward magic-link tokens through uncontrolled redirects (PRD §7).
    if (url.searchParams.has("token")) {
      url.searchParams.delete("token");
    }
    return applySecurityHeaders(NextResponse.redirect(url, 308));
  }

  // Never expose the component preview in production (PRD §14).
  // Prefer APP_ENV / NEXT_PUBLIC_APP_ENV so local `next start` for e2e can keep /dev open.
  const appEnv =
    process.env.NEXT_PUBLIC_APP_ENV || process.env.APP_ENV || process.env.NODE_ENV;
  if (request.nextUrl.pathname.startsWith("/dev/") && appEnv === "production") {
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
