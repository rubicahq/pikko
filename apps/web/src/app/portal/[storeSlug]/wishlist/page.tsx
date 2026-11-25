"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Heart } from "lucide-react";
import { apiFetch, formatIDR } from "@/lib/api";
import { PortalShell } from "@/components/portal/portal-shell";
import { usePortalSession } from "@/components/portal/use-portal-session";

type Item = {
  id: string;
  product_name: string;
  price_idr?: number;
  campaign_name?: string;
  campaign_slug?: string;
  catalog_id?: string;
};

export default function PortalWishlistPage() {
  const storeSlug = String(useParams().storeSlug || "");
  const { session, loading } = usePortalSession(storeSlug);
  const [items, setItems] = useState<Item[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const load = useCallback(async () => {
    try {
      const result = await apiFetch<{ items: Item[] }>(`/api/v1/portal/${storeSlug}/wishlist`);
      setItems(result.items || []);
      setError(null);
    } catch (err) {
      setItems([]);
      setError(err instanceof Error ? err.message : "Wishlist belum bisa dimuat.");
    }
  }, [storeSlug]);
  useEffect(() => { if (session) void load(); }, [session, load]);

  if (loading || !session) return <div className="flex min-h-screen items-center justify-center bg-workspace">Loading…</div>;
  return <PortalShell storeSlug={storeSlug} storeName={session.store.name}
    customerEmail={session.customer.email} primaryColor={session.store.primary_color}>
    <div className="flex items-center gap-3">
      <Heart className="h-6 w-6 text-brand" strokeWidth={1.75} />
      <h1 className="text-page font-bold tracking-tight text-foreground">Wishlist</h1>
    </div>
    <p className="mt-2 text-sm text-foreground-secondary">Produk yang kamu simpan dari Live Shopping.</p>
    {error ? <p role="alert" className="mt-6 text-sm text-error">{error}</p> : null}
    {items === null ? <p className="mt-8 text-sm text-foreground-secondary">Memuat wishlist…</p>
      : items.length === 0 ? <p className="mt-8 rounded-xl border border-border bg-background p-6 text-sm text-foreground-secondary">Belum ada produk di wishlist.</p>
      : <ul className="mt-6 divide-y divide-border-subtle overflow-hidden rounded-xl border border-border bg-background">
        {items.map((item) => <li key={item.id} className="p-4">
          <p className="font-semibold text-foreground">{item.product_name}</p>
          <p className="mt-1 text-sm text-foreground-secondary">{item.campaign_name || "Live Shopping"}
            {item.price_idr ? ` · ${formatIDR(item.price_idr)}` : ""}</p>
          {item.catalog_id && item.campaign_slug ? <Link className="mt-3 inline-block text-sm font-medium text-brand-accessible-dark hover:underline"
            href={`/${storeSlug}/catalog/${item.catalog_id}?live=${encodeURIComponent(item.campaign_slug)}`}>Buka katalog</Link>
            : item.campaign_slug ? <Link className="mt-3 inline-block text-sm font-medium text-brand-accessible-dark hover:underline"
              href={`/${storeSlug}/live/${item.campaign_slug}`}>Buka live</Link> : null}
        </li>)}
      </ul>}
  </PortalShell>;
}
