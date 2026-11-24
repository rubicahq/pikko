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
  storefront: (slug: string) => `${storefront()}/${slug}`,
  storefrontPath: (slug: string) => `/${slug}`,
  api: (path: string) => {
    const base = api();
    const p = path.startsWith("/") ? path : `/${path}`;
    return base ? `${base}${p}` : p;
  },
  dashboardAbsolute: (path: string) =>
    `${dashboard()}${path.startsWith("/") ? path : `/${path}`}`,
};
