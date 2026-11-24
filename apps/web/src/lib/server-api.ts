import { cookies } from "next/headers";
import type {
  Account,
  Category,
  OnboardingDraft,
  Product,
  Session,
  Store,
  StorefrontConfig,
  Theme,
  User,
} from "./api";

const API = process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";
const COOKIE = process.env.AUTH_COOKIE_NAME || "pikko_session";

async function serverFetch<T>(path: string, auth = true): Promise<T | null> {
  const headers: Record<string, string> = { Accept: "application/json" };
  if (auth) {
    const jar = await cookies();
    const token = jar.get(COOKIE)?.value;
    if (!token) return null;
    headers.Authorization = `Bearer ${token}`;
    headers.Cookie = `${COOKIE}=${token}`;
  }
  const res = await fetch(`${API}${path}`, {
    headers,
    cache: "no-store",
  });
  if (!res.ok) return null;
  return (await res.json()) as T;
}

export async function getSession(): Promise<{ user: User; session: Session } | null> {
  return serverFetch("/api/v1/auth/session");
}

export async function getStores(includeArchived = false): Promise<Store[]> {
  const data = await serverFetch<{ stores: Store[] }>(
    `/api/v1/stores${includeArchived ? "?include_archived=true" : ""}`,
  );
  return data?.stores || [];
}

export async function getStore(storeId: string): Promise<Store | null> {
  const data = await serverFetch<{ store: Store }>(`/api/v1/stores/${storeId}`);
  return data?.store || null;
}

export async function getAccount(): Promise<Account | null> {
  const data = await serverFetch<{ account: Account }>("/api/v1/account");
  return data?.account || null;
}

export async function getOnboardingDraft(): Promise<OnboardingDraft | null> {
  const data = await serverFetch<{ draft: OnboardingDraft }>("/api/v1/onboarding/draft");
  return data?.draft || null;
}

export async function getCategories(): Promise<{ value: string; label: string }[]> {
  const data = await serverFetch<{ categories: { value: string; label: string }[] }>(
    "/api/v1/meta/categories",
  );
  return data?.categories || [];
}

export async function getThemes(): Promise<Theme[]> {
  const data = await serverFetch<{ themes: Theme[] }>("/api/v1/meta/themes");
  return data?.themes || [];
}

export async function getStorefront(
  storeId: string,
  channel: "draft" | "published" = "draft",
): Promise<StorefrontConfig | null> {
  const data = await serverFetch<{ storefront: StorefrontConfig }>(
    `/api/v1/stores/${storeId}/storefront?channel=${channel}`,
  );
  return data?.storefront || null;
}

export async function getProducts(
  storeId: string,
  query = "",
): Promise<{ products: Product[]; total: number }> {
  const q = query ? `?q=${encodeURIComponent(query)}` : "";
  const data = await serverFetch<{ products: Product[]; total: number }>(
    `/api/v1/stores/${storeId}/products${q}`,
  );
  return data || { products: [], total: 0 };
}

export async function getProduct(storeId: string, productId: string): Promise<Product | null> {
  const data = await serverFetch<{ product: Product }>(
    `/api/v1/stores/${storeId}/products/${productId}`,
  );
  return data?.product || null;
}

export async function getStoreCategories(storeId: string): Promise<Category[]> {
  const data = await serverFetch<{ categories: Category[] }>(
    `/api/v1/stores/${storeId}/categories`,
  );
  return data?.categories || [];
}

export async function getSetupChecklist(
  storeId: string,
): Promise<Record<string, boolean>> {
  const data = await serverFetch<{ checklist: Record<string, boolean> }>(
    `/api/v1/stores/${storeId}/setup-checklist`,
  );
  return data?.checklist || {};
}

export async function getPublicStore(slug: string) {
  return serverFetch<{
    store: {
      id: string;
      name: string;
      slug: string;
      currency: string;
      storefront: StorefrontConfig;
    };
    storefront_url: string;
  }>(`/api/v1/public/stores/${slug}`, false);
}

export async function getPublicProducts(slug: string) {
  const data = await serverFetch<{ products: Product[]; total: number }>(
    `/api/v1/public/stores/${slug}/products`,
    false,
  );
  return data || { products: [], total: 0 };
}

export async function getPublicProduct(slug: string, productSlug: string) {
  return serverFetch<{
    product: Product;
    product_url: string;
    link_ref: Record<string, unknown>;
  }>(`/api/v1/public/stores/${slug}/products/${productSlug}`, false);
}
