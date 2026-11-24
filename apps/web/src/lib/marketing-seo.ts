import type { Metadata } from "next";
import { marketingOrigin } from "@/lib/site";

const siteName = "Pikko";
const defaultDescription =
  "Sell your way. Create a beautiful online store, give customers a better shopping experience, and manage orders the way your business works.";

export function marketingURL(path = "/"): string {
  const base = marketingOrigin();
  const p = path.startsWith("/") ? path : `/${path}`;
  return `${base}${p === "/" ? "/" : p}`;
}

export function marketingMetadata({
  title,
  description = defaultDescription,
  path = "/",
  noIndex = false,
}: {
  title: string;
  description?: string;
  path?: string;
  noIndex?: boolean;
}): Metadata {
  const url = marketingURL(path);
  const fullTitle = title === siteName ? siteName : `${title} · ${siteName}`;
  return {
    title,
    description,
    alternates: { canonical: url },
    openGraph: {
      type: "website",
      url,
      siteName,
      title: fullTitle,
      description,
      locale: "id_ID",
    },
    twitter: {
      card: "summary_large_image",
      title: fullTitle,
      description,
    },
    robots: noIndex
      ? { index: false, follow: false }
      : { index: true, follow: true },
  };
}

export const MARKETING_DEFAULT_DESCRIPTION = defaultDescription;
