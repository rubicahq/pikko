import type { LucideIcon } from "lucide-react";
import {
  LayoutDashboard,
  ShoppingBag,
  PackageCheck,
  Package,
  Megaphone,
  Users,
  Store,
  Sparkles,
  Settings,
  CircleUser,
} from "lucide-react";

/** Path after `/stores/:storeId/` (may include query). */
export type StorePath = string;

export type NavTab = {
  id: string;
  label: string;
  /** Relative path after `/stores/:storeId/` (query allowed). */
  href: StorePath;
  /** Extra path prefixes that keep this tab active (detail pages). */
  matchPrefixes?: string[];
  /** Query key/value that must match when present on the href. */
  search?: { key: string; value?: string | null };
  implemented: boolean;
  permission?: string;
};

export type WorkspaceDef = {
  id: string;
  label: string;
  icon: LucideIcon;
  /** Default landing path when the sidebar item is clicked. */
  defaultHref: StorePath;
  /** Path prefixes that keep this workspace active in the sidebar. */
  matchPrefixes: string[];
  tabs: NavTab[];
  implemented: boolean;
  permission?: string;
  kind: "primary" | "utility";
};

/**
 * Hybrid navigation SSOT — primary workspaces + contextual tabs.
 * Unimplemented / unauthorized items are filtered at render time.
 */
export const PRIMARY_WORKSPACES: WorkspaceDef[] = [
  {
    id: "overview",
    label: "Overview",
    icon: LayoutDashboard,
    defaultHref: "overview",
    matchPrefixes: ["overview", "analytics"],
    implemented: true,
    permission: "stores.read",
    kind: "primary",
    tabs: [
      {
        id: "summary",
        label: "Summary",
        href: "overview",
        matchPrefixes: ["overview"],
        implemented: true,
        permission: "stores.read",
      },
      {
        id: "analytics",
        label: "Analytics",
        href: "analytics",
        matchPrefixes: ["analytics"],
        implemented: true,
        permission: "analytics.read",
      },
    ],
  },
  {
    id: "orders",
    label: "Orders",
    icon: ShoppingBag,
    defaultHref: "orders",
    matchPrefixes: ["orders", "preorders", "live", "finance"],
    implemented: true,
    permission: "orders.read",
    kind: "primary",
    tabs: [
      {
        id: "all-orders",
        label: "All Orders",
        href: "orders",
        matchPrefixes: ["orders"],
        implemented: true,
        permission: "orders.read",
      },
      {
        id: "preorders",
        label: "Preorders",
        href: "preorders",
        matchPrefixes: ["preorders"],
        implemented: true,
        permission: "campaign.read",
      },
      {
        id: "live",
        label: "Live Shopping",
        href: "live",
        matchPrefixes: ["live"],
        implemented: true,
        permission: "campaign.read",
      },
      {
        id: "payments",
        label: "Payments",
        href: "finance",
        matchPrefixes: ["finance"],
        implemented: true,
        permission: "finance.read",
      },
    ],
  },
  {
    id: "fulfillment",
    label: "Fulfillment",
    icon: PackageCheck,
    defaultHref: "purchasing",
    matchPrefixes: ["purchasing", "packing", "shipments", "vendors", "receiving"],
    implemented: true,
    permission: "purchasing.read",
    kind: "primary",
    tabs: [
      {
        id: "purchasing",
        label: "Purchasing",
        href: "purchasing",
        matchPrefixes: ["purchasing"],
        // Active only when not on receiving tab/detail (handled in matchTab).
        search: { key: "tab", value: null },
        implemented: true,
        permission: "purchasing.read",
      },
      {
        id: "receiving",
        label: "Receiving",
        href: "purchasing?tab=receiving",
        matchPrefixes: ["purchasing/receiving", "receiving"],
        search: { key: "tab", value: "receiving" },
        implemented: true,
        permission: "purchasing.read",
      },
      {
        id: "packing",
        label: "Packing",
        href: "packing",
        matchPrefixes: ["packing"],
        implemented: true,
        permission: "packing.read",
      },
      {
        id: "shipments",
        label: "Shipments",
        href: "shipments",
        matchPrefixes: ["shipments"],
        implemented: true,
        permission: "packing.read",
      },
      {
        id: "vendors",
        label: "Vendors",
        href: "vendors",
        matchPrefixes: ["vendors"],
        implemented: true,
        permission: "purchasing.read",
      },
    ],
  },
  {
    id: "products",
    label: "Products",
    icon: Package,
    defaultHref: "products",
    matchPrefixes: ["products", "catalogs", "catalog", "attributes"],
    implemented: true,
    permission: "catalog.read",
    kind: "primary",
    tabs: [
      {
        id: "all-products",
        label: "All Products",
        href: "products",
        matchPrefixes: ["products", "catalog"],
        implemented: true,
        permission: "catalog.read",
      },
      {
        id: "catalogs",
        label: "Catalogs",
        href: "catalogs",
        matchPrefixes: ["catalogs"],
        implemented: true,
        permission: "catalog.read",
      },
      {
        id: "attributes",
        label: "Attributes",
        href: "attributes",
        matchPrefixes: ["attributes"],
        implemented: true,
        permission: "catalog.read",
      },
    ],
  },
  {
    id: "campaigns",
    label: "Campaigns",
    icon: Megaphone,
    defaultHref: "campaigns",
    matchPrefixes: ["campaigns"],
    implemented: true,
    permission: "campaign.read",
    kind: "primary",
    tabs: [
      {
        id: "all-campaigns",
        label: "All Campaigns",
        href: "campaigns",
        matchPrefixes: ["campaigns"],
        search: { key: "status", value: null },
        implemented: true,
        permission: "campaign.read",
      },
      {
        id: "active",
        label: "Active",
        href: "campaigns?status=active",
        matchPrefixes: ["campaigns"],
        search: { key: "status", value: "active" },
        implemented: true,
        permission: "campaign.read",
      },
      {
        id: "completed",
        label: "Completed",
        href: "campaigns?status=completed",
        matchPrefixes: ["campaigns"],
        search: { key: "status", value: "completed" },
        implemented: true,
        permission: "campaign.read",
      },
    ],
  },
  {
    id: "customers",
    label: "Customers",
    icon: Users,
    defaultHref: "customers",
    matchPrefixes: ["customers"],
    implemented: true,
    permission: "customers.read",
    kind: "primary",
    tabs: [
      {
        id: "all-customers",
        label: "All Customers",
        href: "customers",
        matchPrefixes: ["customers"],
        implemented: true,
        permission: "customers.read",
      },
      {
        id: "store-credit",
        label: "Store Credit",
        href: "customers/credit",
        matchPrefixes: ["customers/credit"],
        implemented: false,
        permission: "customers.read",
      },
    ],
  },
  {
    id: "storefront",
    label: "Storefront",
    icon: Store,
    defaultHref: "storefront",
    matchPrefixes: ["storefront"],
    implemented: true,
    permission: "storefront.read",
    kind: "primary",
    tabs: [
      {
        id: "appearance",
        label: "Appearance",
        href: "storefront",
        matchPrefixes: ["storefront"],
        search: { key: "tab", value: null },
        implemented: true,
        permission: "storefront.read",
      },
      {
        id: "content",
        label: "Content",
        href: "storefront?tab=content",
        matchPrefixes: ["storefront"],
        search: { key: "tab", value: "content" },
        implemented: true,
        permission: "storefront.read",
      },
      {
        id: "preview",
        label: "Preview",
        href: "storefront?tab=preview",
        matchPrefixes: ["storefront"],
        search: { key: "tab", value: "preview" },
        implemented: true,
        permission: "storefront.read",
      },
    ],
  },
];

/** Settings utility — horizontal tabs when on configuration routes. */
export const SETTINGS_WORKSPACE: WorkspaceDef = {
  id: "settings",
  label: "Settings",
  icon: Settings,
  defaultHref: "settings",
  matchPrefixes: ["settings", "team", "business-rules", "messages"],
  implemented: true,
  permission: "stores.settings",
  kind: "utility",
  tabs: [
    {
      id: "general",
      label: "General",
      href: "settings",
      matchPrefixes: ["settings"],
      search: { key: "tab", value: null },
      implemented: true,
      permission: "stores.settings",
    },
    {
      id: "team",
      label: "Team",
      href: "team",
      matchPrefixes: ["team"],
      implemented: true,
      permission: "team.read",
    },
    {
      id: "business-rules",
      label: "Business Rules",
      href: "business-rules",
      matchPrefixes: ["business-rules"],
      implemented: true,
      permission: "business_rules.read",
    },
    {
      id: "payment",
      label: "Payment Settings",
      href: "settings?tab=payment",
      matchPrefixes: ["settings"],
      search: { key: "tab", value: "payment" },
      implemented: true,
      permission: "stores.settings",
    },
    {
      id: "subscription",
      label: "Subscription & Billing",
      href: "settings?tab=subscription",
      matchPrefixes: ["settings"],
      search: { key: "tab", value: "subscription" },
      implemented: true,
      permission: "stores.settings",
    },
    {
      id: "whatsapp",
      label: "WhatsApp Templates",
      href: "messages",
      matchPrefixes: ["messages"],
      implemented: true,
      permission: "notifications.read",
    },
  ],
};

/** Ask Connie lives in the global header (not sidebar). Visual only until AI ships. */
export const CONNIE_NAV = {
  id: "connie",
  label: "Ask Connie",
  icon: Sparkles,
  implemented: false as const,
  comingSoon: true as const,
};

export const ACCOUNT_NAV = {
  id: "account",
  label: "My Account",
  icon: CircleUser,
};

/** Routes that live only in the header (not primary sidebar). */
export const HEADER_UTILITIES = {
  askConnie: CONNIE_NAV,
  messages: {
    href: "messages",
    permission: "notifications.read",
    implemented: true,
  },
  notifications: {
    href: "notifications",
    permission: "notifications.read",
    implemented: true,
  },
  /** Global search is not implemented — do not render a dead control. */
  search: {
    implemented: false,
  },
} as const;

function pathAndQuery(relative: string): { path: string; query: URLSearchParams } {
  const [pathPart, queryPart = ""] = relative.split("?");
  return { path: pathPart.replace(/^\//, ""), query: new URLSearchParams(queryPart) };
}

export function storeHref(storeId: string, relative: StorePath): string {
  const { path, query } = pathAndQuery(relative);
  const qs = query.toString();
  const base = `/stores/${storeId}/${path}`;
  return qs ? `${base}?${qs}` : base;
}

function stripStorePrefix(pathname: string, storeId: string): string {
  const prefix = `/stores/${storeId}/`;
  if (!pathname.startsWith(prefix)) return "";
  return pathname.slice(prefix.length);
}

function allowed(
  permission: string | undefined,
  permissions: string[] | null | undefined,
): boolean {
  if (!permission) return true;
  if (!permissions) return true;
  return permissions.includes(permission);
}

export function filterTabs(
  tabs: NavTab[],
  permissions?: string[] | null,
): NavTab[] {
  return tabs.filter(
    (t) => t.implemented && allowed(t.permission, permissions),
  );
}

export function livePrimaryWorkspaces(
  permissions?: string[] | null,
): WorkspaceDef[] {
  return PRIMARY_WORKSPACES.filter((w) => w.implemented)
    .map((w) => {
      const tabs = filterTabs(w.tabs, permissions);
      return {
        ...w,
        tabs,
        // Land on the first authorized tab when the workspace default is gated.
        defaultHref: tabs[0]?.href ?? w.defaultHref,
      };
    })
    .filter((w) => w.tabs.length > 0);
}

export function liveSettingsWorkspace(
  permissions?: string[] | null,
): WorkspaceDef | null {
  if (!SETTINGS_WORKSPACE.implemented) return null;
  const tabs = filterTabs(SETTINGS_WORKSPACE.tabs, permissions);
  if (tabs.length === 0) return null;
  return {
    ...SETTINGS_WORKSPACE,
    tabs,
    defaultHref: tabs[0]?.href ?? SETTINGS_WORKSPACE.defaultHref,
  };
}

function onPathPrefix(rest: string, prefix: string): boolean {
  return rest === prefix || rest.startsWith(prefix + "/");
}

export function matchTab(
  tab: NavTab,
  pathname: string,
  storeId: string,
  search: string,
): boolean {
  const rest = stripStorePrefix(pathname, storeId);
  if (!rest) return false;
  const params = new URLSearchParams(
    search.startsWith("?") ? search.slice(1) : search,
  );
  const hrefPath = pathAndQuery(tab.href).path;
  const prefixes = tab.matchPrefixes || [hrefPath];
  const onBranch = prefixes.some((p) => onPathPrefix(rest, p));
  if (!onBranch) return false;

  // Fulfillment: Receiving vs Purchasing share /purchasing
  if (tab.id === "receiving") {
    if (rest.startsWith("purchasing/receiving") || rest.startsWith("receiving")) {
      return true;
    }
    return rest === "purchasing" && params.get("tab") === "receiving";
  }
  if (tab.id === "purchasing") {
    if (rest.startsWith("purchasing/receiving")) return false;
    if (rest === "purchasing") return params.get("tab") !== "receiving";
    return onPathPrefix(rest, "purchasing");
  }

  // Query-scoped tabs (campaigns status, storefront/settings section)
  if (tab.search) {
    const current = params.get(tab.search.key);
    const isListRoot = rest === hrefPath;
    if (tab.search.value == null || tab.search.value === "") {
      // Default tab: list root without the query key, OR any detail page.
      if (isListRoot) return current == null || current === "";
      return true;
    }
    // Specific query tab: only on list root with matching query.
    if (!isListRoot) return false;
    return current === tab.search.value;
  }

  return true;
}

export function matchWorkspace(
  workspace: WorkspaceDef,
  pathname: string,
  storeId: string,
): boolean {
  const rest = stripStorePrefix(pathname, storeId);
  if (!rest) return false;
  // Messages is under Settings utility; notifications are header-only.
  return workspace.matchPrefixes.some(
    (p) => rest === p || rest.startsWith(p + "/"),
  );
}

export function resolveWorkspace(
  pathname: string,
  storeId: string,
  permissions?: string[] | null,
): WorkspaceDef | null {
  const primary = livePrimaryWorkspaces(permissions);
  for (const w of primary) {
    if (matchWorkspace(w, pathname, storeId)) return w;
  }
  const settings = liveSettingsWorkspace(permissions);
  if (settings && matchWorkspace(settings, pathname, storeId)) return settings;
  return null;
}

export function resolveActiveTab(
  workspace: WorkspaceDef,
  pathname: string,
  storeId: string,
  search: string,
): NavTab | null {
  const tabs = workspace.tabs;
  // Prefer the most specific matching tab (query-specific before defaults).
  const ranked = [...tabs].sort((a, b) => {
    const aQ = a.search?.value ? 1 : 0;
    const bQ = b.search?.value ? 1 : 0;
    return bQ - aQ;
  });
  for (const tab of ranked) {
    if (matchTab(tab, pathname, storeId, search)) return tab;
  }
  // Detail pages: fall back to first implemented tab whose prefix matches.
  const rest = stripStorePrefix(pathname, storeId);
  for (const tab of tabs) {
    const prefixes = tab.matchPrefixes || [pathAndQuery(tab.href).path];
    if (prefixes.some((p) => rest === p || rest.startsWith(p + "/"))) {
      if (!tab.search?.value) return tab;
    }
  }
  return tabs[0] || null;
}

/** @deprecated Prefer livePrimaryWorkspaces — kept for legacy preview surfaces. */
export type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
  implemented: boolean;
};

/** @deprecated Prefer livePrimaryWorkspaces. */
export type NavGroup = {
  id: string;
  label: string;
  items: NavItem[];
};

/** Flattened live groups for the component preview catalog. */
export function liveNavGroups(
  _groups?: NavGroup[],
  permissions?: string[] | null,
): NavGroup[] {
  const primary = livePrimaryWorkspaces(permissions);
  const settings = liveSettingsWorkspace(permissions);
  const groups: NavGroup[] = [
    {
      id: "primary",
      label: "Workspaces",
      items: primary.map((w) => ({
        href: w.defaultHref,
        label: w.label,
        icon: w.icon,
        implemented: w.implemented,
      })),
    },
  ];
  if (settings) {
    groups.push({
      id: "utility",
      label: "Utility",
      items: [
        {
          href: settings.defaultHref,
          label: settings.label,
          icon: settings.icon,
          implemented: true,
        },
      ],
    });
  }
  return groups;
}
