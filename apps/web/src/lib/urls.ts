/**
 * Central URL generation for the web app (PRD §7).
 * Absolute hosts come from env; business logic uses relative paths.
 */

const marketing = () =>
  process.env.NEXT_PUBLIC_MARKETING_URL || "http://localhost:3000";
const dashboard = () =>
  process.env.NEXT_PUBLIC_DASHBOARD_URL || "http://localhost:3000";
const storefront = () =>
  process.env.NEXT_PUBLIC_STOREFRONT_URL || "http://localhost:3000";
// Browser calls same-origin `/api/v1/*` (Next rewrite → Go). Absolute API URL is for SSR.
const api = () => {
  if (typeof window !== "undefined") {
    return "";
  }
  return process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";
};

export const urls = {
  marketingHome: () => `${marketing()}/`,
  auth: () => `/auth`,
  authCheckInbox: (email?: string) =>
    email ? `/auth/check-inbox?email=${encodeURIComponent(email)}` : `/auth/check-inbox`,
  authConfirm: (token: string) => `/auth/confirm?token=${encodeURIComponent(token)}`,
  authInvalid: () => `/auth/invalid`,
  onboarding: () => `/onboarding`,
  stores: () => `/stores`,
  newStore: () => `/stores/new`,
  accountSettings: () => `/account/settings`,
  storeOverview: (storeId: string) => `/stores/${storeId}/overview`,
  storeStorefront: (storeId: string) => `/stores/${storeId}/storefront`,
  storeCatalog: (storeId: string) => `/stores/${storeId}/catalog`,
  storeCatalogNew: (storeId: string) => `/stores/${storeId}/catalog/new`,
  storeCatalogProduct: (storeId: string, productId: string) =>
    `/stores/${storeId}/catalog/${productId}`,
  catalogImport: (storeId: string) => `/stores/${storeId}/catalog/import`,
  catalogImportJob: (storeId: string, jobId: string) =>
    `/stores/${storeId}/catalog/import/${jobId}`,
  catalogFlipbook: (storeId: string) => `/stores/${storeId}/catalog/flipbook`,
  storeSettings: (storeId: string) => `/stores/${storeId}/settings`,
  storeTeam: (storeId: string) => `/stores/${storeId}/team`,
  inviteAccept: (token: string) => `/invites/accept?token=${encodeURIComponent(token)}`,
  storeOrders: (storeId: string) => `/stores/${storeId}/orders`,
  storeOrder: (storeId: string, orderId: string) => `/stores/${storeId}/orders/${orderId}`,
  storeCustomers: (storeId: string) => `/stores/${storeId}/customers`,
  storeCustomer: (storeId: string, customerId: string) =>
    `/stores/${storeId}/customers/${customerId}`,
  storeAnalytics: (storeId: string) => `/stores/${storeId}/analytics`,
  storeNotifications: (storeId: string) => `/stores/${storeId}/notifications`,
  storePreorders: (storeId: string) => `/stores/${storeId}/preorders`,
  storePreorder: (storeId: string, batchId: string) => `/stores/${storeId}/preorders/${batchId}`,
  storefront: (slug: string) => `${storefront()}/${slug}`,
  storefrontPath: (slug: string) => `/${slug}`,
  storefrontCart: (slug: string) => `/${slug}/cart`,
  storefrontCheckout: (slug: string) => `/${slug}/checkout`,
  storefrontCatalog: (slug: string) => `${storefront()}/${slug}/catalog`,
  storefrontProduct: (slug: string, productSlug: string) => `/${slug}/products/${productSlug}`,
  storefrontProductAbsolute: (slug: string, productSlug: string) =>
    `${storefront()}/${slug}/products/${productSlug}`,
  storefrontPreorder: (slug: string, batchSlug: string) => `/${slug}/preorders/${batchSlug}`,
  storefrontPreorderAbsolute: (slug: string, batchSlug: string) =>
    `${storefront()}/${slug}/preorders/${batchSlug}`,
  tracking: (token: string) => `/track/${encodeURIComponent(token)}`,
  trackingAbsolute: (token: string) => `${storefront()}/track/${encodeURIComponent(token)}`,
  api: (path: string) => {
    const base = api();
    const p = path.startsWith("/") ? path : `/${path}`;
    return base ? `${base}${p}` : p;
  },
  dashboardAbsolute: (path: string) =>
    `${dashboard()}${path.startsWith("/") ? path : `/${path}`}`,
  file: (storeId: string, fileId: string) =>
    urls.api(`/api/v1/stores/${storeId}/files/${fileId}`),
  publicFile: (storeId: string, fileId: string) =>
    urls.api(`/api/v1/public/files/${storeId}/${fileId}`),
};

export const RESERVED_STOREFRONT_SLUGS = new Set([
  "auth",
  "account",
  "stores",
  "onboarding",
  "api",
  "dev",
  "track",
  "catalog",
  "flipbook",
  "products",
  "collections",
  "preorders",
  "cart",
  "search",
  "favicon.ico",
  "pricing",
  "faq",
  "privacy",
  "terms",
  "legal",
  "solutions",
  "sitemap",
  "robots",
  "about",
  "contact",
  "security",
]);
