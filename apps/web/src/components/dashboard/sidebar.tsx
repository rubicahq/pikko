"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Store,
  Package,
  ShoppingBag,
  CalendarClock,
  Users,
  SlidersHorizontal,
  Settings,
  UserRound,
} from "lucide-react";
import type { Store as StoreType, User } from "@/lib/api";
import { Wordmark } from "@/components/brand/wordmark";
import { StoreSwitcher } from "@/components/dashboard/store-switcher";
import { cn } from "@/lib/utils";
import { urls } from "@/lib/urls";

const nav = [
  { href: "overview", label: "Overview", icon: LayoutDashboard },
  { href: "storefront", label: "Storefront", icon: Store },
  { href: "catalog", label: "Catalog", icon: Package },
  { href: "orders", label: "Orders", icon: ShoppingBag },
  { href: "preorders", label: "Preorders", icon: CalendarClock, soon: true },
  { href: "customers", label: "Customers", icon: Users, soon: true },
  { href: "business-rules", label: "Business Rules", icon: SlidersHorizontal, soon: true },
  { href: "settings", label: "Settings", icon: Settings },
];

export function DashboardSidebar({
  user,
  stores,
  currentStore,
}: {
  user: User;
  stores: StoreType[];
  currentStore?: StoreType | null;
}) {
  const pathname = usePathname();
  const storeId = currentStore?.id;

  return (
    <aside className="flex h-full w-sidebar flex-col border-r border-border bg-background">
      <div className="px-4 py-5">
        <Wordmark href={storeId ? urls.storeOverview(storeId) : urls.stores()} />
      </div>
      <div className="px-3 pb-4">
        <StoreSwitcher stores={stores} current={currentStore} />
      </div>
      <nav className="flex-1 space-y-0.5 px-3">
        {nav.map((item) => {
          const href = storeId ? `/stores/${storeId}/${item.href}` : urls.stores();
          const active = pathname.startsWith(href);
          const Icon = item.icon;
          return (
            <Link
              key={item.href}
              href={item.soon ? href : href}
              aria-disabled={item.soon}
              className={cn(
                "flex items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground-secondary hover:bg-surface hover:text-foreground",
                active && "bg-surface font-medium text-foreground",
                item.soon && item.href !== "overview" && "opacity-60",
              )}
              title={item.soon && item.href !== "overview" ? "Coming in a later phase" : undefined}
            >
              <Icon className="h-5 w-5" strokeWidth={1.75} />
              {item.label}
            </Link>
          );
        })}
      </nav>
      <div className="border-t border-border p-3">
        <Link
          href={urls.accountSettings()}
          className={cn(
            "flex items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground-secondary hover:bg-surface hover:text-foreground",
            pathname.startsWith("/account") && "bg-surface font-medium text-foreground",
          )}
        >
          <UserRound className="h-5 w-5" strokeWidth={1.75} />
          <span className="min-w-0 truncate">{user.display_name || user.email}</span>
        </Link>
      </div>
    </aside>
  );
}
