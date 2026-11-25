/**
 * Storefront wishlist identity for scheduled Live Shopping.
 * Phone is required on first save; portal URL is returned by the API.
 */

const KEY = "pikko_wishlist_identity_v1";

export type WishlistIdentity = {
  storeSlug: string;
  phone: string;
  portalUrl?: string;
  customerId?: string;
};

function readAll(): WishlistIdentity[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return [];
    return JSON.parse(raw) as WishlistIdentity[];
  } catch {
    return [];
  }
}

function writeAll(all: WishlistIdentity[]) {
  localStorage.setItem(KEY, JSON.stringify(all));
}

export function getWishlistIdentity(storeSlug: string): WishlistIdentity | null {
  return readAll().find((i) => i.storeSlug === storeSlug) || null;
}

export function setWishlistIdentity(identity: WishlistIdentity): void {
  if (typeof window === "undefined") return;
  const all = readAll().filter((i) => i.storeSlug !== identity.storeSlug);
  all.push(identity);
  writeAll(all);
  window.dispatchEvent(
    new CustomEvent("pikko:wishlist-identity", { detail: { storeSlug: identity.storeSlug } }),
  );
}

export function clearWishlistIdentity(storeSlug: string): void {
  if (typeof window === "undefined") return;
  writeAll(readAll().filter((i) => i.storeSlug !== storeSlug));
}
