import type { Metadata } from "next";
import type { StorefrontConfig } from "@/lib/api";
import { urls } from "@/lib/urls";

/** Public logo URL for storefront HTML (same-origin relative). */
export function storefrontLogoSrc(
  storeId: string,
  sf: Pick<StorefrontConfig, "logo_file_id" | "logo_url">,
): string | undefined {
  if (sf.logo_file_id) return urls.publicFile(storeId, sf.logo_file_id);
  if (sf.logo_url?.startsWith("/")) return sf.logo_url;
  return undefined;
}

/**
 * Favicon + site (apple) icon metadata for a published storefront.
 * Uses relative public file paths so SSR never embeds Docker-internal API hosts.
 */
export function storefrontIconsMetadata(
  storeId: string,
  sf: Pick<
    StorefrontConfig,
    "favicon_file_id" | "logo_file_id" | "store_icon_file_id"
  >,
): Metadata["icons"] | undefined {
  const faviconId = sf.favicon_file_id || sf.logo_file_id;
  const appleId = sf.store_icon_file_id || sf.favicon_file_id || sf.logo_file_id;
  if (!faviconId && !appleId) return undefined;

  const icons: NonNullable<Metadata["icons"]> = {};
  if (faviconId) {
    icons.icon = [{ url: urls.publicFile(storeId, faviconId) }];
  }
  if (appleId) {
    icons.apple = [{ url: urls.publicFile(storeId, appleId) }];
  }
  return icons;
}
