import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getPublicCatalogs, getPublicStore } from "@/lib/server-api";
import { safeCssColor } from "@/lib/safe-css-color";
import { storefrontIconsMetadata, storefrontLogoSrc } from "@/lib/storefront-branding";
import { RESERVED_STOREFRONT_SLUGS, urls } from "@/lib/urls";
import { CartBadge } from "@/components/storefront/cart-badge";
import { CatalogBannerCards } from "@/components/storefront/catalog-banner-cards";

type Props = { params: Promise<{ storeSlug: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { storeSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) return {};
  const data = await getPublicStore(storeSlug);
  if (!data) return { robots: { index: false, follow: false } };
  return {
    title: `Catalogs · ${data.store.name}`,
    description: `Browse catalogs from ${data.store.name}`,
    alternates: { canonical: urls.storefrontCatalogs(storeSlug) },
    icons: storefrontIconsMetadata(data.store.id, data.store.storefront),
  };
}

export default async function PublicCatalogsExplorePage({ params }: Props) {
  const { storeSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) notFound();
  const data = await getPublicStore(storeSlug);
  if (!data) notFound();
  const { store } = data;
  const sf = store.storefront;
  const catalogs = await getPublicCatalogs(storeSlug);
  const logoSrc = storefrontLogoSrc(store.id, sf);
  const bg = safeCssColor(sf.background_color, "#FFFFFF");
  const text = safeCssColor(sf.text_color, "#15191E");
  const primary = safeCssColor(sf.primary_color, "#FF3D00");
  const border = safeCssColor(sf.border_color, "#E1E3E8");
  const surface = safeCssColor(sf.surface_color, "#F3F3F7");

  return (
    <div className="min-h-screen" style={{ background: bg, color: text }}>
      <header className="mx-auto flex max-w-dashboard items-center justify-between gap-4 px-4 py-6 md:px-8">
        <div className="flex items-center gap-3">
          <Link href={urls.storefrontPath(storeSlug)} className="flex items-center gap-3">
            {logoSrc ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={logoSrc} alt={store.name} className="h-10 w-auto" />
            ) : (
              <span className="text-2xl font-medium">{store.name}</span>
            )}
          </Link>
        </div>
        <div className="flex items-center gap-4">
          <CartBadge storeSlug={storeSlug} color={text} />
          {sf.whatsapp ? (
            <a
              className="text-sm underline-offset-2 hover:underline"
              href={`https://wa.me/${sf.whatsapp.replace(/\D/g, "")}`}
              style={{ color: primary }}
            >
              Contact
            </a>
          ) : null}
        </div>
      </header>

      <main className="mx-auto max-w-dashboard px-4 pb-16 md:px-8">
        <p className="text-sm opacity-70">
          <Link href={urls.storefrontPath(storeSlug)} className="hover:opacity-100">
            Store
          </Link>
          <span className="mx-1.5">/</span>
          Catalogs
        </p>
        <h1 className="mt-3 text-2xl font-medium">Catalogs</h1>
        <p className="mt-2 max-w-reading text-sm opacity-70">
          Browse every published catalog from {store.name}.
        </p>

        {catalogs.length === 0 ? (
          <p className="mt-10 text-sm opacity-70">No catalogs published yet.</p>
        ) : (
          <CatalogBannerCards
            storeSlug={storeSlug}
            catalogs={catalogs}
            borderColor={border}
            surfaceColor={surface}
            textColor={text}
          />
        )}
      </main>

      <footer
        className="flex flex-col items-center gap-2 border-t px-4 py-6 text-center text-xs opacity-60 md:px-8"
        style={{ borderColor: border }}
      >
        <span>{store.name}</span>
        {!data.hide_platform_branding ? (
          <span className="inline-flex items-center gap-1.5">
            Powered by
            <Image
              src="/brand/pikko-logo.png"
              alt="Pikko"
              width={72}
              height={29}
              className="h-3.5 w-auto opacity-90"
            />
          </span>
        ) : null}
      </footer>
    </div>
  );
}
