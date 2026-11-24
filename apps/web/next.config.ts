import type { NextConfig } from "next";
import path from "path";

const apiUpstream =
  process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

const nextConfig: NextConfig = {
  output: "standalone",
  // Allow importing shared packages from the monorepo root.
  turbopack: {
    root: path.join(__dirname, "../.."),
  },
  transpilePackages: ["@pikko/design-tokens"],
  async rewrites() {
    // Auth/API traffic uses the App Router proxy at src/app/api/v1/[...path]
    // so Set-Cookie is forwarded correctly. Health stays a simple rewrite.
    return [
      {
        source: "/healthz",
        destination: `${apiUpstream}/healthz`,
      },
      {
        source: "/readyz",
        destination: `${apiUpstream}/readyz`,
      },
      {
        source: "/livez",
        destination: `${apiUpstream}/livez`,
      },
    ];
  },
  async headers() {
    return [
      {
        source: "/:path*",
        headers: [
          { key: "X-Content-Type-Options", value: "nosniff" },
          { key: "X-Frame-Options", value: "DENY" },
          { key: "Referrer-Policy", value: "strict-origin-when-cross-origin" },
          {
            key: "Permissions-Policy",
            value: "camera=(), microphone=(), geolocation=(), payment=(), usb=()",
          },
        ],
      },
    ];
  },
};

export default nextConfig;
