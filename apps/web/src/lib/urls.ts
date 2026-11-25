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
  // Products — master barang (SKU, price, variants, relate/merge, CSV)
  storeProducts: (storeId: string) => `/stores/${storeId}/products`,
  storeProductNew: (storeId: string) => `/stores/${storeId}/products/new`,
  storeProduct: (storeId: string, productId: string) =>
    `/stores/${storeId}/products/${productId}`,
  productsImport: (storeId: string) => `/stores/${storeId}/products/import`,
  productsImportJob: (storeId: string, jobId: string) =>
    `/stores/${storeId}/products/import/${jobId}`,
  storeAttributes: (storeId: string) => `/stores/${storeId}/attributes`,
  storeAttributeNew: (storeId: string) => `/stores/${storeId}/attributes/new`,
  storeAttribute: (storeId: string, attributeId: string) =>
    `/stores/${storeId}/attributes/${attributeId}`,
  // Catalogs — PDF flipbook documents
  storeCatalogs: (storeId: string) => `/stores/${storeId}/catalogs`,
  storeCatalogsNew: (storeId: string) => `/stores/${storeId}/catalogs/new`,
  storeCatalogDoc: (storeId: string, catalogId: string) =>
    `/stores/${storeId}/catalogs/${catalogId}`,
  catalogsImportJob: (storeId: string, jobId: string) =>
    `/stores/${storeId}/catalogs/import/${jobId}`,
  catalogsFlipbook: (storeId: string) => `/stores/${storeId}/catalogs/flipbook`,
  // Legacy aliases (old /catalog product-master paths)
  storeCatalog: (storeId: string) => `/stores/${storeId}/products`,
  storeCatalogNew: (storeId: string) => `/stores/${storeId}/products/new`,
  storeCatalogProduct: (storeId: string, productId: string) =>
    `/stores/${storeId}/products/${productId}`,
  catalogImport: (storeId: string) => `/stores/${storeId}/products/import`,
  catalogImportJob: (storeId: string, jobId: string) =>
    `/stores/${storeId}/catalogs/import/${jobId}`,
  catalogFlipbook: (storeId: string) => `/stores/${storeId}/catalogs/flipbook`,
  storeSettings: (storeId: string) => `/stores/${storeId}/settings`,
  storeTeam: (storeId: string) => `/stores/${storeId}/team`,
  inviteAccept: (token: string) => `/invites/accept?token=${encodeURIComponent(token)}`,
  storeOrders: (storeId: string) => `/stores/${storeId}/orders`,
  storeOrderNew: (storeId: string) => `/stores/${storeId}/orders/new`,
  storeOrderImport: (storeId: string) => `/stores/${storeId}/orders/import`,
  storeOrder: (storeId: string, orderId: string) => `/stores/${storeId}/orders/${orderId}`,
  storeCustomers: (storeId: string) => `/stores/${storeId}/customers`,
  storeCustomer: (storeId: string, customerId: string) =>
    `/stores/${storeId}/customers/${customerId}`,
  storeAnalytics: (storeId: string) => `/stores/${storeId}/analytics`,
  storeNotifications: (storeId: string) => `/stores/${storeId}/notifications`,
  storeMessages: (storeId: string, query?: Record<string, string>) => {
    const base = `/stores/${storeId}/messages`;
    if (!query) return base;
    const qs = new URLSearchParams(query).toString();
    return qs ? `${base}?${qs}` : base;
  },
  storePreorders: (storeId: string) => `/stores/${storeId}/preorders`,
  storePreorder: (storeId: string, batchId: string) => `/stores/${storeId}/preorders/${batchId}`,
  storeCampaigns: (storeId: string) => `/stores/${storeId}/campaigns`,
  storeCampaign: (storeId: string, campaignId: string) => `/stores/${storeId}/campaigns/${campaignId}`,
  storeCampaignNew: (storeId: string) => `/stores/${storeId}/campaigns/new`,
  storeLive: (storeId: string) => `/stores/${storeId}/live`,
  storeLiveSession: (storeId: string, campaignId: string) => `/stores/${storeId}/live/${campaignId}`,
  storePurchasing: (storeId: string) => `/stores/${storeId}/purchasing`,
  storeVendors: (storeId: string) => `/stores/${storeId}/vendors`,
  storeReceiving: (storeId: string, receivingId: string) =>
    `/stores/${storeId}/purchasing/receiving/${receivingId}`,
  storePacking: (storeId: string) => `/stores/${storeId}/packing`,
  storeShipments: (storeId: string) => `/stores/${storeId}/shipments`,
  storeFinance: (storeId: string) => `/stores/${storeId}/finance`,
  storefront: (slug: string) => `${storefront()}/${slug}`,
  storefrontPath: (slug: string) => `/${slug}`,
  storefrontCart: (slug: string) => `/${slug}/cart`,
  storefrontCheckout: (slug: string) => `/${slug}/checkout`,
  storefrontCatalog: (slug: string) => `${storefront()}/${slug}/catalog`,
  storefrontCatalogs: (slug: string) => `/${slug}/catalogs`,
  storefrontCatalogDoc: (slug: string, catalogId: string) => `/${slug}/catalog/${catalogId}`,
  storefrontProduct: (slug: string, productSlug: string) => `/${slug}/products/${productSlug}`,
  storefrontProductAbsolute: (slug: string, productSlug: string) =>
    `${storefront()}/${slug}/products/${productSlug}`,
  storefrontPreorder: (slug: string, batchSlug: string) => `/${slug}/preorders/${batchSlug}`,
  storefrontPreorderAbsolute: (slug: string, batchSlug: string) =>
    `${storefront()}/${slug}/preorders/${batchSlug}`,
  storefrontLive: (slug: string, liveSlug: string) => `/${slug}/live/${liveSlug}`,
  storefrontLiveAbsolute: (slug: string, liveSlug: string) =>
    `${storefront()}/${slug}/live/${liveSlug}`,
  tracking: (token: string) => `/track/${encodeURIComponent(token)}`,
  trackingAbsolute: (token: string) => `${storefront()}/track/${encodeURIComponent(token)}`,
  /** Portal session landing (not shareable — needs personal access URL). */
  portal: (storeSlug: string) => `/portal/${encodeURIComponent(storeSlug)}`,
  portalAbsolute: (storeSlug: string) =>
    `${storefront()}/portal/${encodeURIComponent(storeSlug)}`,
  /** Durable shareable customer portal: /{slug}/customer/{secret}-{phone} */
  portalCustomer: (storeSlug: string, secret: string, phone: string) =>
    `/${encodeURIComponent(storeSlug)}/customer/${encodeURIComponent(`${secret}-${phone}`)}`,
  portalCustomerAbsolute: (storeSlug: string, secret: string, phone: string) =>
    `${storefront()}/${encodeURIComponent(storeSlug)}/customer/${encodeURIComponent(`${secret}-${phone}`)}`,
  api: (path: string) => {
    const base = api();
    const p = path.startsWith("/") ? path : `/${path}`;
    return base ? `${base}${p}` : p;
  },
  dashboardAbsolute: (path: string) =>
    `${dashboard()}${path.startsWith("/") ? path : `/${path}`}`,
  /**
   * Browser-facing file URLs must stay same-origin relative.
   * Never prefix with API_BASE_URL — on the web container that is the Docker
   * hostname `http://api:8080`, which browsers cannot resolve (broken logo/favicon).
   */
  file: (storeId: string, fileId: string) =>
    `/api/v1/stores/${storeId}/files/${fileId}`,
  publicFile: (storeId: string, fileId: string) =>
    `/api/v1/public/files/${storeId}/${fileId}`,
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
  "catalogs",
  "flipbook",
  "products",
  "collections",
  "preorders",
  "live",
  "cart",
  "search",
  "favicon.ico",
  "brand",
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
  "portal",
  "invites",
]);
