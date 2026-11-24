"use client";

import { useState } from "react";
import { ShoppingBag } from "lucide-react";
import { addToCart } from "@/lib/cart";
import { Button } from "@/components/ui/button";

type BatchOption = {
  id: string;
  name: string;
  slug: string;
  closes_at?: string;
  price_idr?: number;
};

export function AddToCartButton({
  storeSlug,
  productId,
  productSlug,
  name,
  priceIdr,
  primaryColor,
  buttonFg,
  disabled,
  batches = [],
  preorderEligible = false,
  readyStockEligible = true,
}: {
  storeSlug: string;
  productId: string;
  productSlug: string;
  name: string;
  priceIdr: number;
  primaryColor: string;
  buttonFg: string;
  disabled?: boolean;
  batches?: BatchOption[];
  preorderEligible?: boolean;
  readyStockEligible?: boolean;
}) {
  const [added, setAdded] = useState(false);
  const openBatches = preorderEligible ? batches : [];
  const [batchId, setBatchId] = useState(openBatches[0]?.id || "");
  const needsBatch = openBatches.length > 0 && !readyStockEligible;
  const canReady = readyStockEligible;
  const [mode, setMode] = useState<"ready_stock" | "preorder">(
    canReady ? "ready_stock" : "preorder",
  );

  function onAdd() {
    const selected = openBatches.find((b) => b.id === batchId);
    const usePreorder = mode === "preorder" || needsBatch;
    if (usePreorder && openBatches.length > 0 && !selected) return;
    addToCart({
      storeSlug,
      productId,
      productSlug,
      name,
      priceIdr: usePreorder && selected?.price_idr != null ? selected.price_idr : priceIdr,
      quantity: 1,
      batchId: usePreorder ? selected?.id || null : null,
      batchName: usePreorder ? selected?.name || null : null,
      sellingModel: usePreorder ? "preorder" : "ready_stock",
    });
    setAdded(true);
    window.setTimeout(() => setAdded(false), 1600);
  }

  return (
    <div className="space-y-3">
      {openBatches.length > 0 && canReady ? (
        <div className="flex flex-wrap gap-2 text-sm">
          <button
            type="button"
            className={`rounded-md border px-3 py-2 ${mode === "ready_stock" ? "border-current font-medium" : "opacity-70"}`}
            onClick={() => setMode("ready_stock")}
          >
            Ready stock
          </button>
          <button
            type="button"
            className={`rounded-md border px-3 py-2 ${mode === "preorder" ? "border-current font-medium" : "opacity-70"}`}
            onClick={() => setMode("preorder")}
          >
            Preorder
          </button>
        </div>
      ) : null}
      {(mode === "preorder" || needsBatch) && openBatches.length > 0 ? (
        <label className="block text-sm opacity-80">
          Preorder batch
          <select
            className="mt-1 h-11 w-full rounded-md border bg-transparent px-3"
            value={batchId}
            onChange={(e) => setBatchId(e.target.value)}
          >
            {openBatches.map((b) => (
              <option key={b.id} value={b.id}>
                {b.name}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      {preorderEligible && openBatches.length === 0 && !canReady ? (
        <p className="text-sm opacity-70">No open preorder batches for this product right now.</p>
      ) : null}
      <Button
        type="button"
        disabled={
          disabled ||
          ((mode === "preorder" || needsBatch) && openBatches.length === 0) ||
          (!canReady && openBatches.length === 0)
        }
        onClick={onAdd}
        className="gap-2"
        style={{ background: primaryColor, color: buttonFg }}
      >
        <ShoppingBag className="h-4 w-4" strokeWidth={1.75} />
        {added ? "Added to cart" : "Add to cart"}
      </Button>
    </div>
  );
}
