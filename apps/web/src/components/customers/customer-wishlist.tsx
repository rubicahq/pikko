"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ApiError, apiFetch, formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";

type WishlistItem = {
  id: string;
  product_id: string;
  product_name?: string;
  product_slug?: string;
  price_idr?: number;
  campaign_id: string;
  campaign_name?: string;
  campaign_slug?: string;
  created_at: string;
};

export function CustomerWishlist({
  storeId,
  customerId,
}: {
  storeId: string;
  customerId: string;
}) {
  const [items, setItems] = useState<WishlistItem[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await apiFetch<{ items: WishlistItem[] }>(
          `/api/v1/stores/${storeId}/customers/${customerId}/wishlist`,
        );
        if (!cancelled) setItems(res.items || []);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.message : "Gagal memuat wishlist.");
          setItems([]);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [storeId, customerId]);

  return (
    <section className="mt-8">
      <h2 className="text-xl font-medium text-foreground">Wishlist</h2>
      <p className="mt-1 text-sm text-foreground-secondary">
        Item yang disimpan pelanggan saat Live Shopping masih scheduled.
      </p>
      {error ? <p className="mt-4 text-sm text-error">{error}</p> : null}
      {items === null ? (
        <p className="mt-4 text-sm text-foreground-secondary">Memuat…</p>
      ) : items.length === 0 ? (
        <p className="mt-4 text-sm text-foreground-secondary">Belum ada wishlist.</p>
      ) : (
        <div className="mt-4 overflow-hidden rounded-[10px] border border-border">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface text-xs font-medium uppercase tracking-wide text-foreground-secondary">
              <tr>
                <th className="px-4 py-3">Product</th>
                <th className="px-4 py-3">Live</th>
                <th className="px-4 py-3 text-right">Price</th>
                <th className="px-4 py-3 text-right">Saved</th>
              </tr>
            </thead>
            <tbody>
              {items.map((item) => (
                <tr key={item.id} className="border-t border-border">
                  <td className="px-4 py-3 font-medium">{item.product_name || item.product_id}</td>
                  <td className="px-4 py-3">
                    {item.campaign_id ? (
                      <Link
                        href={urls.storeLiveSession(storeId, item.campaign_id)}
                        className="hover:underline"
                      >
                        {item.campaign_name || "Live"}
                      </Link>
                    ) : (
                      "—"
                    )}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums">
                    {item.price_idr != null ? formatIDR(item.price_idr) : "—"}
                  </td>
                  <td className="px-4 py-3 text-right text-xs text-foreground-muted">
                    {new Date(item.created_at).toLocaleString("id-ID", {
                      timeZone: "Asia/Jakarta",
                    })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
