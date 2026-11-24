/**
 * Lightweight cart hook for storefront hotspots.
 * Full checkout (PRD §20) is not in this release — items persist in localStorage
 * and CTAs can deep-link to the product page.
 */

export type CartLine = {
  storeSlug: string;
  productId: string;
  productSlug: string;
  name: string;
  priceIdr: number;
  variantId?: string | null;
  quantity: number;
};

const KEY = "pikko_cart_v1";

export function readCart(storeSlug: string): CartLine[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    const all = JSON.parse(raw) as CartLine[];
    return all.filter((l) => l.storeSlug === storeSlug);
  } catch {
    return [];
  }
}

export function addToCart(line: CartLine): CartLine[] {
  if (typeof window === "undefined") return [];
  let all: CartLine[] = [];
  try {
    all = JSON.parse(localStorage.getItem(KEY) || "[]") as CartLine[];
  } catch {
    all = [];
  }
  const idx = all.findIndex(
    (l) =>
      l.storeSlug === line.storeSlug &&
      l.productId === line.productId &&
      (l.variantId || null) === (line.variantId || null),
  );
  if (idx >= 0) {
    all[idx] = { ...all[idx], quantity: all[idx].quantity + (line.quantity || 1) };
  } else {
    all.push({ ...line, quantity: line.quantity || 1 });
  }
  localStorage.setItem(KEY, JSON.stringify(all));
  window.dispatchEvent(new CustomEvent("pikko:cart-updated", { detail: { storeSlug: line.storeSlug } }));
  return all.filter((l) => l.storeSlug === line.storeSlug);
}

/** Hook point for future checkout — call after addToCart when order APIs exist. */
export function onCartChanged(storeSlug: string, cb: (lines: CartLine[]) => void) {
  const handler = (e: Event) => {
    const detail = (e as CustomEvent).detail;
    if (detail?.storeSlug === storeSlug) cb(readCart(storeSlug));
  };
  window.addEventListener("pikko:cart-updated", handler);
  return () => window.removeEventListener("pikko:cart-updated", handler);
}
