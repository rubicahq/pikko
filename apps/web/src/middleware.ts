import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { isHQHost, mapLegacyHost } from "@/lib/site";

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
  const host = request.headers.get("host") || "";
  const mapped = mapLegacyHost(host);
  if (mapped) {
    const url = request.nextUrl.clone();
    url.host = mapped;
    url.protocol = "https:";
    url.port = "";
    if (url.searchParams.has("token")) {
      url.searchParams.delete("token");
    }
    return applySecurityHeaders(NextResponse.redirect(url, 308));
  }

  const appEnv =
    process.env.NEXT_PUBLIC_APP_ENV || process.env.APP_ENV || process.env.NODE_ENV;
  if (request.nextUrl.pathname.startsWith("/dev/") && appEnv === "production") {
    return applySecurityHeaders(NextResponse.rewrite(new URL("/404", request.url)));
  }

  // hq.pikko.site → rewrite into /hq/* app surface (not merchant dashboard).
  if (isHQHost(host)) {
    const path = request.nextUrl.pathname;
    if (
      path.startsWith("/api/") ||
      path.startsWith("/_next/") ||
      path === "/favicon.ico" ||
      path.startsWith("/hq/") ||
      path === "/hq"
    ) {
      return applySecurityHeaders(NextResponse.next());
    }
    const url = request.nextUrl.clone();
    url.pathname = path === "/" ? "/hq" : `/hq${path}`;
    return applySecurityHeaders(NextResponse.rewrite(url));
  }

  // /hq/* is reachable by path for local/dev. Production uses hq.* host rewrite above.
  // Authorization is enforced by HQ API (allowlist) — merchant nav never links here.
  return applySecurityHeaders(NextResponse.next());
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
