"use client";

import { useState } from "react";
import { ShoppingBag } from "lucide-react";
import { addToCart } from "@/lib/cart";
import { Button } from "@/components/ui/button";

export function AddToCartButton({
  storeSlug,
  productId,
  productSlug,
  name,
  priceIdr,
  primaryColor,
  buttonFg,
  disabled,
}: {
  storeSlug: string;
  productId: string;
  productSlug: string;
  name: string;
  priceIdr: number;
  primaryColor: string;
  buttonFg: string;
  disabled?: boolean;
}) {
  const [added, setAdded] = useState(false);

  function onAdd() {
    addToCart({
      storeSlug,
      productId,
      productSlug,
      name,
      priceIdr,
      quantity: 1,
    });
    setAdded(true);
    window.setTimeout(() => setAdded(false), 1600);
  }

  return (
    <Button
      type="button"
      disabled={disabled}
      onClick={onAdd}
      className="gap-2"
      style={{ background: primaryColor, color: buttonFg }}
    >
      <ShoppingBag className="h-4 w-4" strokeWidth={1.75} />
      {added ? "Added to cart" : "Add to cart"}
    </Button>
  );
}
