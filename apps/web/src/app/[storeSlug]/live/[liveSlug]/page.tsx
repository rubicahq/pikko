import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getPublicLive, getPublicStore } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { storefrontIconsMetadata } from "@/lib/storefront-branding";
import { RESERVED_STOREFRONT_SLUGS, urls } from "@/lib/urls";
import { CartBadge } from "@/components/storefront/cart-badge";
import { AddToCartButton } from "@/components/storefront/add-to-cart-button";
import { WishlistButton } from "@/components/storefront/wishlist-button";

type Props = { params: Promise<{ storeSlug: string; liveSlug: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { storeSlug, liveSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) return {};
  const [storeData, liveData] = await Promise.all([
    getPublicStore(storeSlug),
    getPublicLive(storeSlug, liveSlug),
  ]);
  if (!storeData || !liveData?.live) return { robots: { index: false } };
  return {
    title: `${liveData.live.name} · ${storeData.store.name}`,
    description: liveData.live.description || `Live shopping ${liveData.live.name}`,
    alternates: { canonical: urls.storefrontLiveAbsolute(storeSlug, liveSlug) },
    icons: storefrontIconsMetadata(storeData.store.id, storeData.store.storefront),
  };
}

export default async function PublicLivePage({ params }: Props) {
  const { storeSlug, liveSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) notFound();
  const [storeData, liveData] = await Promise.all([
    getPublicStore(storeSlug),
    getPublicLive(storeSlug, liveSlug),
  ]);
  if (!storeData || !liveData?.live) notFound();
  const store = storeData.store;
  const sf = store.storefront;
  const live = liveData.live;
  const open = live.is_accepting_orders || live.operational_status === "open";
  const wishlistEnabled =
    live.wishlist_enabled === true || live.operational_status === "scheduled";
  const catalogs = live.catalogs || [];

  return (
    <div className="min-h-screen" style={{ background: sf.background_color, color: sf.text_color }}>
      <header className="mx-auto flex max-w-dashboard items-center justify-between px-4 py-6 md:px-8">
        <Link href={urls.storefrontPath(storeSlug)} className="text-sm opacity-70 hover:opacity-100">
          ← {store.name}
        </Link>
        <CartBadge storeSlug={storeSlug} color={sf.text_color} />
      </header>
      <main className="mx-auto max-w-dashboard px-4 pb-16 md:px-8">
        <p className="text-sm opacity-70 capitalize">
          Live · {live.operational_status.replaceAll("_", " ")}
        </p>
        <h1 className="mt-2 text-3xl font-medium">{live.name}</h1>
        {live.description ? (
          <p className="mt-4 max-w-reading whitespace-pre-wrap opacity-80">{live.description}</p>
        ) : null}
        <p className="mt-4 text-sm opacity-70">
          Window{" "}
          {new Intl.DateTimeFormat("id-ID", {
            timeZone: "Asia/Jakarta",
            dateStyle: "medium",
            timeStyle: "short",
          }).format(new Date(live.opens_at))}{" "}
          –{" "}
          {new Intl.DateTimeFormat("id-ID", {
            timeZone: "Asia/Jakarta",
            dateStyle: "medium",
            timeStyle: "short",
          }).format(new Date(live.closes_at))}{" "}
          (Asia/Jakarta)
        </p>
        {wishlistEnabled ? (
          <p className="mt-4 text-sm opacity-80">
            Live belum dibuka. Jelajahi katalog dan simpan item ke wishlist — checkout tersedia saat
            sesi dibuka.
          </p>
        ) : !open ? (
          <p className="mt-4 text-sm opacity-80">This live session is not accepting new orders.</p>
        ) : null}

        {catalogs.length > 0 ? (
          <section className="mt-8">
            <h2 className="text-lg font-medium">Catalogs</h2>
            <ul className="mt-3 grid gap-3 sm:grid-cols-2">
              {catalogs.map((c) => (
                <li key={c.id}>
                  <Link
                    href={`/${storeSlug}/catalog`}
                    className="block rounded-md border p-4 transition-opacity hover:opacity-90"
                    style={{ borderColor: sf.border_color || "#E1E3E8" }}
                  >
                    <p className="font-medium">{c.title}</p>
                    <p className="mt-1 text-xs opacity-70">
                      {c.page_count ? `${c.page_count} pages` : "Catalog"} · browse
                    </p>
                  </Link>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        <ul className="mt-10 grid gap-6 md:grid-cols-2">
          {(live.products || []).map((p) => (
            <li
              key={p.product_id}
              className="rounded-md border p-4"
              style={{ borderColor: sf.border_color || "#E1E3E8" }}
            >
              <div className="flex items-start justify-between gap-2">
                <Link
                  href={urls.storefrontProduct(storeSlug, p.product_slug)}
                  className="font-medium hover:underline"
                >
                  {p.product_name}
                </Link>
                {p.featured ? (
                  <span className="text-xs opacity-70" style={{ color: sf.primary_color }}>
                    Featured
                  </span>
                ) : null}
              </div>
              <p className="mt-2 text-lg tabular-nums">{formatIDR(p.effective_price_idr)}</p>
              {p.quantity_limit != null ? (
                <p className="mt-1 text-xs opacity-70">
                  Quota {p.sold_qty}/{p.quantity_limit}
                  {p.remaining_qty != null ? ` · ${p.remaining_qty} left` : ""}
                </p>
              ) : null}
              <div className="mt-4 space-y-3">
                {wishlistEnabled ? (
                  <WishlistButton
                    storeSlug={storeSlug}
                    liveSlug={liveSlug}
                    productId={p.product_id}
                    productName={p.product_name}
                    primaryColor={sf.primary_color || "#FF3D00"}
                    buttonFg={sf.background_color || "#FFFFFF"}
                    enabled
                  />
                ) : (
                  <AddToCartButton
                    storeSlug={storeSlug}
                    productId={p.product_id}
                    productSlug={p.product_slug}
                    name={p.product_name}
                    priceIdr={p.effective_price_idr}
                    primaryColor={sf.primary_color || "#FF3D00"}
                    buttonFg={sf.background_color || "#FFFFFF"}
                    disabled={!open || (p.remaining_qty != null && p.remaining_qty <= 0)}
                    readyStockEligible
                  />
                )}
              </div>
            </li>
          ))}
        </ul>
      </main>
    </div>
  );
}
