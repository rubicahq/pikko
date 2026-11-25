/** Client-side mirror of server rbac for nav gating (NAV-03). Server remains authoritative. */

export type Permission =
  | "stores.read"
  | "stores.write"
  | "stores.archive"
  | "stores.settings"
  | "team.read"
  | "team.invite"
  | "team.manage"
  | "catalog.read"
  | "catalog.write"
  | "campaign.read"
  | "campaign.write"
  | "storefront.read"
  | "storefront.write"
  | "orders.read"
  | "orders.write"
  | "orders.payments"
  | "customers.read"
  | "customers.write"
  | "purchasing.read"
  | "purchasing.write"
  | "packing.read"
  | "packing.write"
  | "finance.read"
  | "finance.write"
  | "business_rules.read"
  | "business_rules.write"
  | "analytics.read"
  | "notifications.read"
  | "notifications.write";

const NAV_PERMISSION: Record<string, Permission> = {
  overview: "stores.read",
  notifications: "notifications.read",
  messages: "notifications.read",
  analytics: "analytics.read",
  orders: "orders.read",
  customers: "customers.read",
  purchasing: "purchasing.read",
  packing: "packing.read",
  shipments: "packing.read",
  finance: "finance.read",
  products: "catalog.read",
  catalogs: "catalog.read",
  catalog: "catalog.read",
  preorders: "campaign.read",
  campaigns: "campaign.read",
  live: "campaign.read",
  storefront: "storefront.read",
  "business-rules": "business_rules.read",
  settings: "stores.settings",
  vendors: "purchasing.read",
  team: "team.read",
  // Hybrid workspace ids
  fulfillment: "purchasing.read",
};

export function hasPermission(
  permissions: string[] | undefined | null,
  perm: Permission,
): boolean {
  if (!permissions || permissions.length === 0) return false;
  return permissions.includes(perm);
}

export function canSeeNavItem(
  href: string,
  permissions: string[] | undefined | null,
): boolean {
  const required = NAV_PERMISSION[href];
  if (!required) return true;
  // Owners historically had no permissions array — treat missing as full access for SSR safety.
  if (!permissions) return true;
  return hasPermission(permissions, required);
}

export const ROLE_LABELS: Record<string, string> = {
  owner: "Owner",
  admin: "Store Admin",
  cs: "Customer Service",
  catalog: "Catalog",
  purchasing: "Purchasing",
  packing: "Packing",
  finance: "Finance",
};
