import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getPublicProducts, getPublicStore } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { RESERVED_STOREFRONT_SLUGS, urls } from "@/lib/urls";

type Props = { params: Promise<{ storeSlug: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { storeSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) return {};
  const data = await getPublicStore(storeSlug);
  if (!data) return { robots: { index: false, follow: false } };
  const sf = data.store.storefront;
  const title = data.store.name;
  const description =
    sf.description || `Shop ${data.store.name} on Pikko.`;
  const favicon = sf.favicon_file_id
    ? urls.publicFile(data.store.id, sf.favicon_file_id)
    : sf.logo_file_id
      ? urls.publicFile(data.store.id, sf.logo_file_id)
      : undefined;
  return {
    title,
    description,
    alternates: { canonical: urls.storefront(storeSlug) },
    openGraph: {
      title,
      description,
      url: urls.storefront(storeSlug),
      siteName: data.store.name,
    },
    icons: favicon ? { icon: favicon } : undefined,
    robots: { index: true, follow: true },
  };
}

export default async function PublicStorePage({ params }: Props) {
  const { storeSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) notFound();
  const data = await getPublicStore(storeSlug);
  if (!data) notFound();
  const { store } = data;
  const sf = store.storefront;
  const { products } = await getPublicProducts(storeSlug);
  const logoSrc = sf.logo_file_id
    ? urls.publicFile(store.id, sf.logo_file_id)
    : sf.logo_url;

  return (
    <div
      className="min-h-screen"
      style={{
        background: sf.background_color,
        color: sf.text_color,
      }}
    >
      <header className="mx-auto flex max-w-dashboard items-center justify-between gap-4 px-4 py-6 md:px-8">
        <div className="flex items-center gap-3">
          {logoSrc ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={logoSrc} alt={store.name} className="h-10 w-auto" />
          ) : (
            <h1 className="text-2xl font-medium">{store.name}</h1>
          )}
        </div>
        {sf.whatsapp && (
          <a
            className="text-sm underline-offset-2 hover:underline"
            href={`https://wa.me/${sf.whatsapp.replace(/\D/g, "")}`}
            style={{ color: sf.primary_color }}
          >
            Contact
          </a>
        )}
      </header>

      <main className="mx-auto max-w-dashboard px-4 pb-16 md:px-8">
        {logoSrc && <h1 className="sr-only">{store.name}</h1>}
        {sf.description && (
          <p className="max-w-reading text-base opacity-80">{sf.description}</p>
        )}

        <section className="mt-10">
          <h2 className="text-xl font-medium">Products</h2>
          {products.length === 0 ? (
            <p className="mt-4 text-sm opacity-70">No products available yet.</p>
          ) : (
            <ul className="mt-6 grid grid-cols-2 gap-4 md:grid-cols-3 lg:grid-cols-4">
              {products.map((p) => (
                <li key={p.id}>
                  <Link
                    href={urls.storefrontProduct(storeSlug, p.slug)}
                    className="block rounded-md border p-3 transition-opacity hover:opacity-90"
                    style={{ borderColor: sf.border_color || "#E1E3E8" }}
                  >
                    <div
                      className="aspect-square overflow-hidden rounded-md"
                      style={{ background: sf.surface_color || "#F3F3F7" }}
                    >
                      {p.primary_image_url ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img
                          src={p.primary_image_url}
                          alt={p.name}
                          className="h-full w-full object-cover"
                        />
                      ) : null}
                    </div>
                    <p className="mt-3 text-sm font-medium">{p.name}</p>
                    <p className="mt-1 text-sm tabular-nums opacity-80">
                      {formatIDR(p.base_price_idr)}
                    </p>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      </main>

      <footer className="border-t px-4 py-6 text-center text-xs opacity-60 md:px-8"
        style={{ borderColor: sf.border_color || "#E1E3E8" }}
      >
        {store.name} · Powered by Pikko
      </footer>
    </div>
  );
}
