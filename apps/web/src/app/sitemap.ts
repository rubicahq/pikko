import type { MetadataRoute } from "next";
import { marketingURL } from "@/lib/marketing-seo";

export default function sitemap(): MetadataRoute.Sitemap {
  const paths = [
    "/",
    "/pricing",
    "/faq",
    "/privacy",
    "/terms",
    "/solutions",
    "/solutions/jastip",
    "/solutions/social-sellers",
  ];
  const now = new Date();
  return paths.map((path) => ({
    url: marketingURL(path),
    lastModified: now,
    changeFrequency: path === "/" ? "weekly" : "monthly",
    priority: path === "/" ? 1 : 0.7,
  }));
}
