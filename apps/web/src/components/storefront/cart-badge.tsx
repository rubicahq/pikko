"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { ShoppingBag } from "lucide-react";
import { cartCount, onCartChanged, readCart } from "@/lib/cart";

export function CartBadge({ storeSlug, color }: { storeSlug: string; color: string }) {
  const [count, setCount] = useState(0);

  useEffect(() => {
    setCount(cartCount(storeSlug));
    return onCartChanged(storeSlug, () => setCount(cartCount(storeSlug)));
  }, [storeSlug]);

  // hydrate once
  useEffect(() => {
    setCount(readCart(storeSlug).reduce((n, l) => n + l.quantity, 0));
  }, [storeSlug]);

  return (
    <Link
      href={`/${storeSlug}/cart`}
      className="relative inline-flex items-center gap-1 text-sm opacity-80 hover:opacity-100"
      style={{ color }}
      aria-label={`Cart, ${count} items`}
    >
      <ShoppingBag className="h-5 w-5" strokeWidth={1.75} />
      {count > 0 ? (
        <span
          className="absolute -right-2 -top-2 flex h-4 min-w-4 items-center justify-center rounded-sm px-0.5 text-[10px] font-medium text-white"
          style={{ background: "#C83200" }}
        >
          {count}
        </span>
      ) : null}
    </Link>
  );
}
