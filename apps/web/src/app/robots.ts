import type { MetadataRoute } from "next";
import { marketingURL } from "@/lib/marketing-seo";

export default function robots(): MetadataRoute.Robots {
  const appEnv = process.env.NEXT_PUBLIC_APP_ENV || process.env.NODE_ENV || "development";
  if (appEnv !== "production") {
    return {
      rules: { userAgent: "*", disallow: "/" },
    };
  }

  return {
    rules: [
      {
        userAgent: "*",
        allow: ["/", "/pricing", "/faq", "/privacy", "/terms", "/solutions"],
        disallow: [
          "/auth",
          "/account",
          "/stores",
          "/onboarding",
          "/dev",
          "/track",
          "/api",
        ],
      },
    ],
    sitemap: marketingURL("/sitemap.xml"),
    host: marketingURL("/").replace(/\/$/, ""),
  };
}
