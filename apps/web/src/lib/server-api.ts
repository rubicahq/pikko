import { cookies } from "next/headers";
import type {
  Account,
  AttributeDefinition,
  Category,
  OnboardingDraft,
  Product,
  ProductAttributeValue,
  Session,
  Store,
  StorefrontConfig,
  Theme,
  User,
} from "./api";

const API = process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";
const COOKIE = process.env.AUTH_COOKIE_NAME || "pikko_session";

export async function serverFetch<T>(path: string, auth = true): Promise<T | null> {
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
  const data = await serverFetch<{ store: Store; permissions?: string[] }>(
    `/api/v1/stores/${storeId}`,
  );
  if (!data?.store) return null;
  return {
    ...data.store,
    permissions: data.permissions ?? data.store.permissions,
  };
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

export async function getStoreAttributes(storeId: string): Promise<AttributeDefinition[]> {
  const data = await serverFetch<{ attributes: AttributeDefinition[] }>(
    `/api/v1/stores/${storeId}/attributes`,
  );
  return data?.attributes || [];
}

export async function getProductAttributeValues(
  storeId: string,
  productId: string,
): Promise<ProductAttributeValue[]> {
  const data = await serverFetch<{ values: ProductAttributeValue[] }>(
    `/api/v1/stores/${storeId}/products/${productId}/attributes`,
  );
  return data?.values || [];
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
      checkout_enabled: boolean;
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

export async function getPublicBatches(slug: string) {
  const data = await serverFetch<{
    batches: {
      id: string;
      name: string;
      slug: string;
      operational_status: string;
      closes_at: string;
      eta_start?: string | null;
      eta_end?: string | null;
    }[];
  }>(`/api/v1/public/stores/${slug}/preorders`, false);
  return data?.batches || [];
}

export async function getPublicBatch(slug: string, batchSlug: string) {
  return serverFetch<{
    batch: {
      id: string;
      name: string;
      slug: string;
      description?: string | null;
      operational_status: string;
      closes_at: string;
      eta_start?: string | null;
      eta_end?: string | null;
      products?: {
        product_id: string;
        product_name: string;
        product_slug: string;
        effective_price_idr: number;
        quantity_limit?: number | null;
        reserved_qty?: number;
      }[];
    };
  }>(`/api/v1/public/stores/${slug}/preorders/${batchSlug}`, false);
}

export async function getPublicProductBatches(slug: string, productSlug: string) {
  const data = await serverFetch<{
    batches: { id: string; name: string; slug: string; closes_at: string }[];
  }>(`/api/v1/public/stores/${slug}/products/${productSlug}/preorders`, false);
  return data?.batches || [];
}

export async function getPublicLives(slug: string) {
  const data = await serverFetch<{
    lives: {
      id: string;
      name: string;
      slug: string;
      operational_status: string;
      closes_at: string;
      opens_at: string;
      product_count: number;
      description?: string | null;
    }[];
  }>(`/api/v1/public/stores/${slug}/live`, false);
  return data?.lives || [];
}

export async function getPublicLive(slug: string, liveSlug: string) {
  return serverFetch<{
    live: {
      id: string;
      name: string;
      slug: string;
      description?: string | null;
      opens_at: string;
      closes_at: string;
      operational_status: string;
      is_accepting_orders: boolean;
      products?: {
        product_id: string;
        product_name: string;
        product_slug: string;
        effective_price_idr: number;
        quantity_limit?: number | null;
        sold_qty: number;
        remaining_qty?: number | null;
        featured: boolean;
        availability: string;
      }[];
    };
  }>(`/api/v1/public/stores/${slug}/live/${liveSlug}`, false);
}
