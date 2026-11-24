"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { formatIDR } from "@/lib/api";
import {
  cartSubtotal,
  clearCart,
  onCartChanged,
  readCart,
  removeFromCart,
  setCartQuantity,
  type CartLine,
} from "@/lib/cart";
import { Button } from "@/components/ui/button";

export default function CartPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const [lines, setLines] = useState<CartLine[]>([]);

  useEffect(() => {
    setLines(readCart(storeSlug));
    return onCartChanged(storeSlug, setLines);
  }, [storeSlug]);

  const subtotal = cartSubtotal(storeSlug);

  return (
    <div className="mx-auto min-h-screen max-w-[1120px] px-4 py-8 md:px-8">
      <div className="mb-6 flex items-center justify-between gap-4">
        <div>
          <Link href={`/${storeSlug}`} className="text-sm text-foreground-secondary hover:text-foreground">
            ← Back to store
          </Link>
          <h1 className="mt-2 text-[30px] font-medium text-foreground">Cart</h1>
        </div>
        {lines.length > 0 ? (
          <Button type="button" variant="ghost" onClick={() => clearCart(storeSlug)}>
            Clear
          </Button>
        ) : null}
      </div>

      {lines.length === 0 ? (
        <div className="rounded-[10px] border border-border bg-background p-8">
          <p className="text-sm text-foreground-secondary">Your cart is empty.</p>
          <Link href={`/${storeSlug}`} className="mt-4 inline-block text-sm font-medium text-brand-accessible-dark">
            Continue shopping
          </Link>
        </div>
      ) : (
        <div className="grid gap-8 lg:grid-cols-[1fr_320px]">
          <ul className="space-y-4">
            {lines.map((line) => (
              <li
                key={`${line.productId}:${line.variantId || ""}`}
                className="flex flex-col gap-3 rounded-[10px] border border-border p-4 sm:flex-row sm:items-center sm:justify-between"
              >
                <div>
                  <Link
                    href={`/${storeSlug}/products/${line.productSlug}`}
                    className="font-medium text-foreground hover:underline"
                  >
                    {line.name}
                  </Link>
                  <p className="mt-1 text-sm tabular-nums text-foreground-secondary">
                    {formatIDR(line.priceIdr)}
                  </p>
                </div>
                <div className="flex items-center gap-3">
                  <label className="sr-only" htmlFor={`qty-${line.productId}`}>
                    Quantity
                  </label>
                  <input
                    id={`qty-${line.productId}`}
                    type="number"
                    min={1}
                    max={999}
                    className="h-10 w-20 rounded-md border border-border px-2 text-sm tabular-nums"
                    value={line.quantity}
                    onChange={(e) =>
                      setCartQuantity(
                        storeSlug,
                        line.productId,
                        line.variantId,
                        Number(e.target.value) || 1,
                      )
                    }
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => removeFromCart(storeSlug, line.productId, line.variantId)}
                  >
                    Remove
                  </Button>
                </div>
              </li>
            ))}
          </ul>
          <aside className="h-fit rounded-[10px] border border-border p-5">
            <p className="text-sm text-foreground-secondary">Subtotal</p>
            <p className="mt-1 text-2xl font-medium tabular-nums">{formatIDR(subtotal)}</p>
            <p className="mt-2 text-xs text-foreground-muted">
              Shipping and final total are confirmed at checkout.
            </p>
            <Link href={`/${storeSlug}/checkout`} className="mt-6 block">
              <Button className="w-full" type="button">
                Checkout
              </Button>
            </Link>
          </aside>
        </div>
      )}
    </div>
  );
}
