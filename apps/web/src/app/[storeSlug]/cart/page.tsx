"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { formatIDR } from "@/lib/api";
import {
  cartGroups,
  clearCart,
  onCartChanged,
  readCart,
  removeFromCart,
  setCartQuantity,
  type CartGroup,
  type CartLine,
} from "@/lib/cart";
import { Button } from "@/components/ui/button";

export default function CartPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const [lines, setLines] = useState<CartLine[]>([]);
  const [groups, setGroups] = useState<CartGroup[]>([]);

  useEffect(() => {
    const refresh = () => {
      setLines(readCart(storeSlug));
      setGroups(cartGroups(storeSlug));
    };
    refresh();
    return onCartChanged(storeSlug, refresh);
  }, [storeSlug]);

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
        <div className="space-y-8">
          {groups.map((group) => {
            const subtotal = group.lines.reduce((n, l) => n + l.priceIdr * l.quantity, 0);
            return (
              <section key={group.key} className="grid gap-6 lg:grid-cols-[1fr_280px]">
                <div>
                  <h2 className="text-lg font-medium text-foreground">
                    {group.sellingModel === "preorder"
                      ? `Preorder · ${group.batchName || "Batch"}`
                      : "Ready stock"}
                  </h2>
                  <p className="mt-1 text-sm text-foreground-secondary">
                    Checkout this group separately from other fulfillment schedules.
                  </p>
                  <ul className="mt-4 space-y-4">
                    {group.lines.map((line) => (
                      <li
                        key={`${line.productId}:${line.variantId || ""}:${line.batchId || ""}`}
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
                          <input
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
                                line.batchId,
                              )
                            }
                          />
                          <Button
                            type="button"
                            variant="ghost"
                            onClick={() =>
                              removeFromCart(storeSlug, line.productId, line.variantId, line.batchId)
                            }
                          >
                            Remove
                          </Button>
                        </div>
                      </li>
                    ))}
                  </ul>
                </div>
                <aside className="h-fit rounded-[10px] border border-border p-5">
                  <p className="text-sm text-foreground-secondary">Group subtotal</p>
                  <p className="mt-1 text-2xl font-medium tabular-nums">{formatIDR(subtotal)}</p>
                  <Link
                    href={`/${storeSlug}/checkout?group=${encodeURIComponent(group.key)}`}
                    className="mt-6 block"
                  >
                    <Button className="w-full" type="button">
                      Checkout group
                    </Button>
                  </Link>
                </aside>
              </section>
            );
          })}
        </div>
      )}
    </div>
  );
}
