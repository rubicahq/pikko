import type { LucideIcon } from "lucide-react";
import {
  LayoutDashboard,
  Bell,
  BarChart3,
  ShoppingBag,
  Users,
  Package,
  BookOpen,
  CalendarClock,
  Megaphone,
  Radio,
  Store,
  SlidersHorizontal,
  Settings,
  ClipboardList,
  Boxes,
  Truck,
  Wallet,
  Building2,
  UserCog,
  MessageCircle,
} from "lucide-react";

export type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
  /** When false, item is omitted from the live nav (module not shipped yet). */
  implemented: boolean;
};

export type NavGroup = {
  id: "workspace" | "operations" | "relationships" | "manage";
  label: string;
  items: NavItem[];
};

/**
 * Merchant IA — Friendly Commerce Workspace (BMC-like groups).
 * Unimplemented modules stay with implemented:false so later agents can flip them on.
 */
export const MERCHANT_NAV: NavGroup[] = [
  {
    id: "workspace",
    label: "Workspace",
    items: [
      { href: "overview", label: "Overview", icon: LayoutDashboard, implemented: true },
      { href: "notifications", label: "Notifications", icon: Bell, implemented: true },
      { href: "messages", label: "Messages", icon: MessageCircle, implemented: true },
      { href: "analytics", label: "Analytics", icon: BarChart3, implemented: true },
    ],
  },
  {
    id: "operations",
    label: "Operations",
    items: [
      { href: "orders", label: "Orders", icon: ShoppingBag, implemented: true },
      { href: "live", label: "Live", icon: Radio, implemented: true },
      { href: "purchasing", label: "Purchasing", icon: ClipboardList, implemented: true },
      { href: "packing", label: "Packing", icon: Boxes, implemented: true },
      { href: "shipments", label: "Shipments", icon: Truck, implemented: true },
      { href: "finance", label: "Payments", icon: Wallet, implemented: true },
    ],
  },
  {
    id: "relationships",
    label: "Relationships",
    items: [
      { href: "customers", label: "Customers", icon: Users, implemented: true },
      { href: "vendors", label: "Vendors", icon: Building2, implemented: true },
      { href: "team", label: "Team", icon: UserCog, implemented: true },
    ],
  },
  {
    id: "manage",
    label: "Manage",
    items: [
      { href: "products", label: "Products", icon: Package, implemented: true },
      { href: "catalogs", label: "Catalogs", icon: BookOpen, implemented: true },
      { href: "campaigns", label: "Campaigns", icon: Megaphone, implemented: true },
      { href: "preorders", label: "Preorders", icon: CalendarClock, implemented: true },
      { href: "storefront", label: "Storefront", icon: Store, implemented: true },
      { href: "business-rules", label: "Business Rules", icon: SlidersHorizontal, implemented: true },
      { href: "settings", label: "Settings", icon: Settings, implemented: true },
    ],
  },
];

export function liveNavGroups(
  groups: NavGroup[] = MERCHANT_NAV,
  permissions?: string[] | null,
): NavGroup[] {
  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        if (!item.implemented) return false;
        if (!permissions) return true;
        const required = NAV_PERM[item.href];
        if (!required) return true;
        return permissions.includes(required);
      }),
    }))
    .filter((group) => group.items.length > 0);
}

const NAV_PERM: Record<string, string> = {
  overview: "stores.read",
  notifications: "notifications.read",
  messages: "notifications.read",
  analytics: "analytics.read",
  orders: "orders.read",
  live: "campaign.read",
  customers: "customers.read",
  purchasing: "purchasing.read",
  packing: "packing.read",
  shipments: "packing.read",
  finance: "finance.read",
  products: "catalog.read",
  catalogs: "catalog.read",
  campaigns: "campaign.read",
  preorders: "campaign.read",
  storefront: "storefront.read",
  "business-rules": "business_rules.read",
  settings: "stores.settings",
  vendors: "purchasing.read",
  team: "team.read",
};
