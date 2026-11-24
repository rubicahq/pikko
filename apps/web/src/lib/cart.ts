/**
 * Store-scoped cart for storefront + checkout (PRD §20).
 * Line items persist in localStorage; prices/stock are always revalidated
 * server-side on checkout preview/submit.
 * Ready-stock and preorder batches are separate fulfillment groups.
 */

export type CartLine = {
  storeSlug: string;
  productId: string;
  productSlug: string;
  name: string;
  priceIdr: number;
  variantId?: string | null;
  quantity: number;
  imageUrl?: string | null;
  batchId?: string | null;
  batchName?: string | null;
  sellingModel?: "ready_stock" | "preorder";
};

const KEY = "pikko_cart_v3";

function readAll(): CartLine[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) {
      const v2 = localStorage.getItem("pikko_cart_v2") || localStorage.getItem("pikko_cart_v1");
      if (v2) {
        const parsed = (JSON.parse(v2) as CartLine[]).map((l) => ({
          ...l,
          sellingModel: l.batchId ? ("preorder" as const) : ("ready_stock" as const),
        }));
        localStorage.setItem(KEY, JSON.stringify(parsed));
        return parsed;
      }
      return [];
    }
    return JSON.parse(raw) as CartLine[];
  } catch {
    return [];
  }
}

function writeAll(all: CartLine[]) {
  localStorage.setItem(KEY, JSON.stringify(all));
}

function emit(storeSlug: string) {
  window.dispatchEvent(new CustomEvent("pikko:cart-updated", { detail: { storeSlug } }));
}

function lineKey(l: Pick<CartLine, "productId" | "variantId" | "batchId">) {
  return `${l.productId}:${l.variantId || ""}:${l.batchId || ""}`;
}

export function readCart(storeSlug: string): CartLine[] {
  return readAll().filter((l) => l.storeSlug === storeSlug);
}

export type CartGroup = {
  key: string;
  sellingModel: "ready_stock" | "preorder";
  batchId: string | null;
  batchName: string | null;
  lines: CartLine[];
};

export function cartGroups(storeSlug: string): CartGroup[] {
  const lines = readCart(storeSlug);
  const map = new Map<string, CartGroup>();
  for (const l of lines) {
    const batchId = l.batchId || null;
    const key = batchId ? `preorder:${batchId}` : "ready_stock";
    const existing = map.get(key);
    if (existing) existing.lines.push(l);
    else {
      map.set(key, {
        key,
        sellingModel: batchId ? "preorder" : "ready_stock",
        batchId,
        batchName: l.batchName || null,
        lines: [l],
      });
    }
  }
  return Array.from(map.values());
}

export function cartCount(storeSlug: string): number {
  return readCart(storeSlug).reduce((n, l) => n + l.quantity, 0);
}

export function cartSubtotal(storeSlug: string): number {
  return readCart(storeSlug).reduce((n, l) => n + l.priceIdr * l.quantity, 0);
}

export function addToCart(line: CartLine): CartLine[] {
  if (typeof window === "undefined") return [];
  const all = readAll();
  const qty = Math.max(1, line.quantity || 1);
  const normalized: CartLine = {
    ...line,
    batchId: line.batchId || null,
    sellingModel: line.batchId ? "preorder" : "ready_stock",
  };
  const idx = all.findIndex(
    (l) => l.storeSlug === normalized.storeSlug && lineKey(l) === lineKey(normalized),
  );
  if (idx >= 0) {
    all[idx] = {
      ...all[idx],
      ...normalized,
      quantity: all[idx].quantity + qty,
      priceIdr: normalized.priceIdr,
      name: normalized.name,
    };
  } else {
    all.push({ ...normalized, quantity: qty });
  }
  writeAll(all);
  emit(normalized.storeSlug);
  return all.filter((l) => l.storeSlug === normalized.storeSlug);
}

export function setCartQuantity(
  storeSlug: string,
  productId: string,
  variantId: string | null | undefined,
  quantity: number,
  batchId?: string | null,
): CartLine[] {
  if (typeof window === "undefined") return [];
  let all = readAll();
  const target = lineKey({ productId, variantId, batchId: batchId || null });
  const idx = all.findIndex((l) => l.storeSlug === storeSlug && lineKey(l) === target);
  if (idx < 0) return all.filter((l) => l.storeSlug === storeSlug);
  if (quantity <= 0) {
    all = all.filter((_, i) => i !== idx);
  } else {
    all[idx] = { ...all[idx], quantity: Math.min(999, quantity) };
  }
  writeAll(all);
  emit(storeSlug);
  return all.filter((l) => l.storeSlug === storeSlug);
}

export function removeFromCart(
  storeSlug: string,
  productId: string,
  variantId?: string | null,
  batchId?: string | null,
): CartLine[] {
  return setCartQuantity(storeSlug, productId, variantId, 0, batchId);
}

export function clearCart(storeSlug: string, groupKey?: string): void {
  if (typeof window === "undefined") return;
  if (!groupKey) {
    writeAll(readAll().filter((l) => l.storeSlug !== storeSlug));
  } else {
    writeAll(
      readAll().filter((l) => {
        if (l.storeSlug !== storeSlug) return true;
        const key = l.batchId ? `preorder:${l.batchId}` : "ready_stock";
        return key !== groupKey;
      }),
    );
  }
  emit(storeSlug);
}

export function onCartChanged(storeSlug: string, cb: (lines: CartLine[]) => void) {
  const handler = (e: Event) => {
    const detail = (e as CustomEvent).detail;
    if (detail?.storeSlug === storeSlug) cb(readCart(storeSlug));
  };
  window.addEventListener("pikko:cart-updated", handler);
  return () => window.removeEventListener("pikko:cart-updated", handler);
}
