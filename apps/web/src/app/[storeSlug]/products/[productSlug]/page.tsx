import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getPublicProduct, getPublicStore } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { RESERVED_STOREFRONT_SLUGS, urls } from "@/lib/urls";

type Props = {
  params: Promise<{ storeSlug: string; productSlug: string }>;
};

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { storeSlug, productSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) return {};
  const [storeData, productData] = await Promise.all([
    getPublicStore(storeSlug),
    getPublicProduct(storeSlug, productSlug),
  ]);
  if (!storeData || !productData) return { robots: { index: false, follow: false } };
  const p = productData.product;
  const sf = storeData.store.storefront;
  const title = `${p.name} · ${storeData.store.name}`;
  const description = p.description || `Buy ${p.name} from ${storeData.store.name}.`;
  const favicon = sf.favicon_file_id
    ? urls.publicFile(storeData.store.id, sf.favicon_file_id)
    : undefined;
  return {
    title,
    description,
    alternates: { canonical: urls.storefrontProductAbsolute(storeSlug, productSlug) },
    openGraph: {
      title,
      description,
      url: urls.storefrontProductAbsolute(storeSlug, productSlug),
    },
    icons: favicon ? { icon: favicon } : undefined,
  };
}

export default async function PublicProductPage({ params }: Props) {
  const { storeSlug, productSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) notFound();
  const [storeData, productData] = await Promise.all([
    getPublicStore(storeSlug),
    getPublicProduct(storeSlug, productSlug),
  ]);
  if (!storeData || !productData) notFound();
  const store = storeData.store;
  const sf = store.storefront;
  const p = productData.product;
  const image = p.images?.[0]?.url || p.primary_image_url;

  return (
    <div className="min-h-screen" style={{ background: sf.background_color, color: sf.text_color }}>
      <header className="mx-auto max-w-dashboard px-4 py-6 md:px-8">
        <Link href={urls.storefrontPath(storeSlug)} className="text-sm opacity-70 hover:opacity-100">
          ← {store.name}
        </Link>
      </header>
      <main className="mx-auto grid max-w-dashboard gap-8 px-4 pb-16 md:grid-cols-2 md:px-8">
        <div
          className="aspect-square overflow-hidden rounded-md border"
          style={{ borderColor: sf.border_color, background: sf.surface_color }}
        >
          {image ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={image} alt={p.name} className="h-full w-full object-cover" />
          ) : null}
        </div>
        <div>
          <h1 className="text-3xl font-medium">{p.name}</h1>
          {p.sku && <p className="mt-2 text-sm opacity-60">SKU {p.sku}</p>}
          <p className="mt-4 text-xl tabular-nums">{formatIDR(p.base_price_idr)}</p>
          {p.description && <p className="mt-6 text-base opacity-80 whitespace-pre-wrap">{p.description}</p>}
          <p className="mt-8 text-sm opacity-70">
            Checkout arrives in a later phase. This product is ready for flipbook hotspot linking via
            id <code className="text-xs">{p.id}</code>.
          </p>
        </div>
      </main>
    </div>
  );
}
