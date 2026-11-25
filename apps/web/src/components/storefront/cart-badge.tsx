"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ShoppingBag } from "lucide-react";
import { formatIDR } from "@/lib/api";
import { cartGroups, onCartChanged, readCart, removeFromCart, setCartQuantity, type CartLine } from "@/lib/cart";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";

export function CartBadge({ storeSlug, color }: { storeSlug: string; color: string }) {
  const [open, setOpen] = useState(false);
  const [lines, setLines] = useState<CartLine[]>([]);

  useEffect(() => {
    setLines(readCart(storeSlug));
    return onCartChanged(storeSlug, setLines);
  }, [storeSlug]);

  const count = lines.reduce((total, line) => total + line.quantity, 0);
  const groups = cartGroups(storeSlug);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <button type="button" className="relative inline-flex items-center gap-1 text-sm opacity-80 hover:opacity-100" style={{ color }} aria-label={`Open cart, ${count} items`}>
          <ShoppingBag className="h-5 w-5" strokeWidth={1.75} />
          {count > 0 && <span className="absolute -right-2 -top-2 flex h-4 min-w-4 items-center justify-center rounded-sm px-0.5 text-[10px] font-medium text-white" style={{ background: "#C83200" }}>{count}</span>}
        </button>
      </SheetTrigger>
      <SheetContent side="right" className="h-dvh w-[min(100%,30rem)]">
        <SheetHeader className="shrink-0"><SheetTitle>Cart {count > 0 ? `(${count})` : ""}</SheetTitle></SheetHeader>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-5">
          {lines.length === 0 ? <p className="text-sm text-foreground-secondary">Your cart is empty.</p> : groups.map((group) => (
            <section key={group.key} className="mb-8">
              <h3 className="font-medium">{group.batchName || (group.sellingModel === "preorder" ? "Preorder" : "Ready stock")}</h3>
              <ul className="mt-3 space-y-3">{group.lines.map((line) => (
                <li key={`${line.productId}:${line.variantId || ""}:${line.batchId || ""}`} className="rounded-lg border border-border p-3">
                  <p className="font-medium">{line.name}</p>
                  <p className="mt-1 text-sm text-foreground-secondary">{formatIDR(line.priceIdr)}</p>
                  <div className="mt-3 flex items-center gap-3">
                    <label className="text-xs text-foreground-secondary">Qty <input className="ml-2 w-16 rounded-md border border-border px-2 py-1 text-sm" type="number" min={1} max={999} value={line.quantity} onChange={(event) => setCartQuantity(storeSlug, line.productId, line.variantId, Number(event.target.value) || 1, line.batchId)} /></label>
                    <button type="button" className="text-xs text-brand-dark underline" onClick={() => removeFromCart(storeSlug, line.productId, line.variantId, line.batchId)}>Remove</button>
                  </div>
                </li>
              ))}</ul>
              <p className="mt-3 text-sm font-medium">Subtotal {formatIDR(group.lines.reduce((total, line) => total + line.priceIdr * line.quantity, 0))}</p>
              <Link href={`/${storeSlug}/checkout?group=${encodeURIComponent(group.key)}`} className="mt-3 inline-flex h-10 w-full items-center justify-center rounded-md bg-foreground px-4 text-sm font-medium text-background">Checkout</Link>
            </section>
          ))}
        </div>
      </SheetContent>
    </Sheet>
  );
}
