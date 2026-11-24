/**
 * Store-scoped cart for storefront + checkout (PRD §20).
 * Line items persist in localStorage; prices/stock are always revalidated
 * server-side on checkout preview/submit.
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
};

const KEY = "pikko_cart_v2";

function readAll(): CartLine[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) {
      // Migrate v1 if present
      const v1 = localStorage.getItem("pikko_cart_v1");
      if (v1) {
        const parsed = JSON.parse(v1) as CartLine[];
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

export function readCart(storeSlug: string): CartLine[] {
  return readAll().filter((l) => l.storeSlug === storeSlug);
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
  const idx = all.findIndex(
    (l) =>
      l.storeSlug === line.storeSlug &&
      l.productId === line.productId &&
      (l.variantId || null) === (line.variantId || null),
  );
  if (idx >= 0) {
    all[idx] = {
      ...all[idx],
      ...line,
      quantity: all[idx].quantity + qty,
      priceIdr: line.priceIdr,
      name: line.name,
    };
  } else {
    all.push({ ...line, quantity: qty });
  }
  writeAll(all);
  emit(line.storeSlug);
  return all.filter((l) => l.storeSlug === line.storeSlug);
}

export function setCartQuantity(
  storeSlug: string,
  productId: string,
  variantId: string | null | undefined,
  quantity: number,
): CartLine[] {
  if (typeof window === "undefined") return [];
  let all = readAll();
  const idx = all.findIndex(
    (l) =>
      l.storeSlug === storeSlug &&
      l.productId === productId &&
      (l.variantId || null) === (variantId || null),
  );
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
): CartLine[] {
  return setCartQuantity(storeSlug, productId, variantId, 0);
}

export function clearCart(storeSlug: string): void {
  if (typeof window === "undefined") return;
  writeAll(readAll().filter((l) => l.storeSlug !== storeSlug));
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
