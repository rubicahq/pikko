import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { getPublicBatch, getPublicStore } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { RESERVED_STOREFRONT_SLUGS, urls } from "@/lib/urls";
import { CartBadge } from "@/components/storefront/cart-badge";
import { AddToCartButton } from "@/components/storefront/add-to-cart-button";

type Props = { params: Promise<{ storeSlug: string; batchSlug: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { storeSlug, batchSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) return {};
  const [storeData, batchData] = await Promise.all([
    getPublicStore(storeSlug),
    getPublicBatch(storeSlug, batchSlug),
  ]);
  if (!storeData || !batchData?.batch) return { robots: { index: false } };
  return {
    title: `${batchData.batch.name} · ${storeData.store.name}`,
    description: batchData.batch.description || `Preorder ${batchData.batch.name}`,
    alternates: { canonical: urls.storefrontPreorderAbsolute(storeSlug, batchSlug) },
  };
}

export default async function PublicPreorderPage({ params }: Props) {
  const { storeSlug, batchSlug } = await params;
  if (RESERVED_STOREFRONT_SLUGS.has(storeSlug)) notFound();
  const [storeData, batchData] = await Promise.all([
    getPublicStore(storeSlug),
    getPublicBatch(storeSlug, batchSlug),
  ]);
  if (!storeData || !batchData?.batch) notFound();
  const store = storeData.store;
  const sf = store.storefront;
  const b = batchData.batch;
  const open = b.operational_status === "open";

  return (
    <div className="min-h-screen" style={{ background: sf.background_color, color: sf.text_color }}>
      <header className="mx-auto flex max-w-dashboard items-center justify-between px-4 py-6 md:px-8">
        <Link href={urls.storefrontPath(storeSlug)} className="text-sm opacity-70 hover:opacity-100">
          ← {store.name}
        </Link>
        <CartBadge storeSlug={storeSlug} color={sf.text_color} />
      </header>
      <main className="mx-auto max-w-dashboard px-4 pb-16 md:px-8">
        <p className="text-sm opacity-70 capitalize">{b.operational_status.replaceAll("_", " ")}</p>
        <h1 className="mt-2 text-3xl font-medium">{b.name}</h1>
        {b.description ? <p className="mt-4 max-w-reading opacity-80 whitespace-pre-wrap">{b.description}</p> : null}
        <p className="mt-4 text-sm opacity-70">
          Closes{" "}
          {new Intl.DateTimeFormat("id-ID", {
            timeZone: "Asia/Jakarta",
            dateStyle: "medium",
            timeStyle: "short",
          }).format(new Date(b.closes_at))}{" "}
          (Asia/Jakarta)
          {b.eta_start ? ` · ETA from ${String(b.eta_start).slice(0, 10)}` : ""}
        </p>
        {!open ? (
          <p className="mt-4 text-sm opacity-80">This batch is not accepting new orders.</p>
        ) : null}
        <ul className="mt-10 grid gap-6 md:grid-cols-2">
          {(b.products || []).map((p) => (
            <li key={p.product_id} className="rounded-md border p-4" style={{ borderColor: sf.border_color }}>
              <Link href={urls.storefrontProduct(storeSlug, p.product_slug)} className="font-medium hover:underline">
                {p.product_name}
              </Link>
              <p className="mt-2 text-lg tabular-nums">{formatIDR(p.effective_price_idr)}</p>
              {p.quantity_limit != null ? (
                <p className="mt-1 text-xs opacity-70">
                  Quota {p.reserved_qty || 0}/{p.quantity_limit}
                </p>
              ) : null}
              <div className="mt-4">
                <AddToCartButton
                  storeSlug={storeSlug}
                  productId={p.product_id}
                  productSlug={p.product_slug}
                  name={p.product_name}
                  priceIdr={p.effective_price_idr}
                  primaryColor={sf.primary_color}
                  buttonFg={sf.button_foreground || "#FCFCFD"}
                  disabled={!open || !store.checkout_enabled}
                  batches={[{ id: b.id, name: b.name, slug: b.slug, price_idr: p.effective_price_idr }]}
                  preorderEligible
                  readyStockEligible={false}
                />
              </div>
            </li>
          ))}
        </ul>
      </main>
    </div>
  );
}
