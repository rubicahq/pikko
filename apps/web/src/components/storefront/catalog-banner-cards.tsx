import Link from "next/link";
import { urls } from "@/lib/urls";

export type StorefrontCatalogCard = {
  id: string;
  title: string;
  cover_url?: string;
  page_count?: number;
};

const HOMEPAGE_LIMIT = 8;

export function CatalogBannerCards({
  storeSlug,
  catalogs,
  limit,
  showExploreMore,
  borderColor,
  surfaceColor,
  textColor,
}: {
  storeSlug: string;
  catalogs: StorefrontCatalogCard[];
  /** Cap shown cards; omit to show all. */
  limit?: number;
  showExploreMore?: boolean;
  borderColor: string;
  surfaceColor: string;
  textColor: string;
}) {
  if (catalogs.length === 0) return null;
  const cap = limit ?? catalogs.length;
  const visible = catalogs.slice(0, cap);
  const hasMore = showExploreMore && catalogs.length > (limit ?? HOMEPAGE_LIMIT);

  return (
    <div>
      <ul className="mt-4 flex flex-col gap-4">
        {visible.map((c) => (
          <li key={c.id}>
            <Link
              href={urls.storefrontCatalogDoc(storeSlug, c.id)}
              className="group relative block overflow-hidden rounded-[14px] border transition-opacity hover:opacity-95"
              style={{ borderColor }}
            >
              <div className="aspect-video w-full" style={{ background: surfaceColor }}>
                {c.cover_url ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={c.cover_url.startsWith("http") ? c.cover_url : urls.api(c.cover_url)}
                    alt=""
                    className="h-full w-full object-cover transition-transform duration-300 group-hover:scale-[1.02]"
                  />
                ) : null}
                <div
                  className="absolute inset-0 flex items-end bg-gradient-to-t from-black/55 via-black/15 to-transparent p-4 md:p-6"
                  style={{ color: "#FFFFFF" }}
                >
                  <div>
                    <p className="text-lg font-semibold tracking-tight md:text-xl">{c.title}</p>
                    {c.page_count != null && c.page_count > 0 ? (
                      <p className="mt-1 text-sm opacity-80">{c.page_count} pages</p>
                    ) : null}
                  </div>
                </div>
              </div>
            </Link>
          </li>
        ))}
      </ul>
      {hasMore ? (
        <div className="mt-6 flex justify-center">
          <Link
            href={urls.storefrontCatalogs(storeSlug)}
            className="inline-flex h-11 items-center justify-center rounded-[10px] border px-5 text-sm font-medium transition-opacity hover:opacity-90"
            style={{ borderColor, color: textColor }}
          >
            Explore more
          </Link>
        </div>
      ) : null}
    </div>
  );
}

export { HOMEPAGE_LIMIT as CATALOG_HOMEPAGE_LIMIT };
