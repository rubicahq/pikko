"use client";

import { useCallback, useEffect, useState } from "react";
import { Heart } from "lucide-react";
import { apiFetch, formatIDR } from "@/lib/api";
import { getWishlistIdentity } from "@/lib/wishlist";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

type WishlistItem = {
  id: string;
  product_name: string;
  price_idr: number;
};

export function WishlistBadge({
  storeSlug,
  liveSlug,
  color,
}: {
  storeSlug: string;
  liveSlug: string;
  color: string;
}) {
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<WishlistItem[]>([]);
  const [hasIdentity, setHasIdentity] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    const identity = getWishlistIdentity(storeSlug);
    setHasIdentity(Boolean(identity?.phone));
    if (!identity?.phone) {
      setItems([]);
      return;
    }
    try {
      const result = await apiFetch<{ items: WishlistItem[] }>(
        `/api/v1/public/stores/${storeSlug}/live/${liveSlug}/wishlist?phone=${encodeURIComponent(identity.phone)}`,
      );
      setItems(result.items || []);
      setError(null);
    } catch {
      setError("Wishlist belum bisa dimuat. Coba lagi.");
    }
  }, [storeSlug, liveSlug]);

  useEffect(() => {
    void refresh();
    const onIdentity = () => { void refresh(); };
    window.addEventListener("pikko:wishlist-identity", onIdentity);
    return () => window.removeEventListener("pikko:wishlist-identity", onIdentity);
  }, [refresh]);

  return (
    <Sheet open={open} onOpenChange={(next) => { setOpen(next); if (next) void refresh(); }}>
      <SheetTrigger asChild>
        <button type="button" aria-label={`Open wishlist, ${items.length} items`}
          className="relative inline-flex items-center opacity-80 hover:opacity-100" style={{ color }}>
          <Heart className="h-5 w-5" strokeWidth={1.75} />
          {items.length > 0 ? <span className="absolute -right-2 -top-2 flex h-4 min-w-4 items-center justify-center rounded-sm bg-red-600 px-0.5 text-[10px] text-white">{items.length}</span> : null}
        </button>
      </SheetTrigger>
      <SheetContent side="right" className="h-dvh w-[min(100%,30rem)]">
        <SheetHeader><SheetTitle>Wishlist {items.length ? `(${items.length})` : ""}</SheetTitle></SheetHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5">
          {!hasIdentity ? <p className="text-sm text-foreground-secondary">Pilih Wishlist pada produk, lalu masukkan nama dan nomor WhatsApp untuk melihat daftar di sini.</p> : null}
          {error ? <p role="alert" className="text-sm text-error">{error}</p> : null}
          {hasIdentity && !error && items.length === 0 ? <p className="text-sm text-foreground-secondary">Belum ada produk di wishlist.</p> : null}
          <ul className="space-y-3">{items.map((item) => (
            <li key={item.id} className="rounded-lg border border-border p-3">
              <p className="font-medium">{item.product_name}</p>
              {item.price_idr > 0 ? <p className="mt-1 text-sm text-foreground-secondary">{formatIDR(item.price_idr)}</p> : null}
            </li>
          ))}</ul>
        </div>
      </SheetContent>
    </Sheet>
  );
}
