import type { LucideIcon } from "lucide-react";
import {
  LayoutDashboard,
  Bell,
  BarChart3,
  ShoppingBag,
  Users,
  Package,
  CalendarClock,
  Store,
  SlidersHorizontal,
  Settings,
  ClipboardList,
  Boxes,
  Truck,
  Wallet,
  Building2,
  UserCog,
} from "lucide-react";

export type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
  /** When false, item is omitted from the live nav (module not shipped yet). */
  implemented: boolean;
};

export type NavGroup = {
  id: "workspace" | "operations" | "configuration";
  label: string;
  items: NavItem[];
};

/**
 * Merchant IA per PRD §6.
 * Unimplemented modules stay in this config with implemented:false so later
 * agents can flip them on without reshaping the shell.
 */
export const MERCHANT_NAV: NavGroup[] = [
  {
    id: "workspace",
    label: "Workspace",
    items: [
      { href: "overview", label: "Overview", icon: LayoutDashboard, implemented: true },
      { href: "notifications", label: "Notifications", icon: Bell, implemented: true },
      { href: "analytics", label: "Analytics", icon: BarChart3, implemented: true },
    ],
  },
  {
    id: "operations",
    label: "Operations",
    items: [
      { href: "orders", label: "Orders", icon: ShoppingBag, implemented: true },
      { href: "customers", label: "Customers", icon: Users, implemented: true },
      // Deferred — Agents 06–09 / finance
      { href: "purchasing", label: "Purchasing", icon: ClipboardList, implemented: false },
      { href: "packing", label: "Packing", icon: Boxes, implemented: false },
      { href: "shipments", label: "Shipments", icon: Truck, implemented: false },
      { href: "finance", label: "Payments", icon: Wallet, implemented: false },
    ],
  },
  {
    id: "configuration",
    label: "Configuration",
    items: [
      { href: "catalog", label: "Catalog", icon: Package, implemented: true },
      { href: "preorders", label: "Preorders", icon: CalendarClock, implemented: true },
      { href: "storefront", label: "Storefront", icon: Store, implemented: true },
      { href: "business-rules", label: "Business Rules", icon: SlidersHorizontal, implemented: true },
      { href: "settings", label: "Settings", icon: Settings, implemented: true },
      // Deferred — Agents 06 / 03+10
      { href: "vendors", label: "Vendors", icon: Building2, implemented: false },
      { href: "team", label: "Team", icon: UserCog, implemented: false },
    ],
  },
];

export function liveNavGroups(groups: NavGroup[] = MERCHANT_NAV): NavGroup[] {
  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => item.implemented),
    }))
    .filter((group) => group.items.length > 0);
}
