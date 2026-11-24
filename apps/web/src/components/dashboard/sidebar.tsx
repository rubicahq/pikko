"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { UserRound } from "lucide-react";
import type { Store as StoreType, User } from "@/lib/api";
import { Wordmark } from "@/components/brand/wordmark";
import { StoreSwitcher } from "@/components/dashboard/store-switcher";
import { SidebarNav } from "@/components/dashboard/sidebar-nav";
import { cn } from "@/lib/utils";
import { urls } from "@/lib/urls";

export function DashboardSidebar({
  user,
  stores,
  currentStore,
  className,
}: {
  user: User;
  stores: StoreType[];
  currentStore?: StoreType | null;
  className?: string;
}) {
  const pathname = usePathname();
  const storeId = currentStore?.id;

  return (
    <aside
      className={cn(
        "flex h-full w-sidebar flex-col border-r border-border bg-background",
        className,
      )}
    >
      <div className="px-4 py-5">
        <Wordmark href={storeId ? urls.storeOverview(storeId) : urls.stores()} />
      </div>
      <div className="px-3 pb-4">
        <StoreSwitcher stores={stores} current={currentStore} />
      </div>
      <SidebarNav storeId={storeId} />
      <div className="mt-auto border-t border-border p-3">
        <Link
          href={urls.accountSettings()}
          className={cn(
            "flex items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground-secondary hover:bg-surface hover:text-foreground",
            pathname.startsWith("/account") && "bg-surface font-medium text-foreground",
          )}
        >
          <UserRound className="h-5 w-5 shrink-0" strokeWidth={1.75} />
          <span className="min-w-0 truncate">{user.display_name || user.email}</span>
        </Link>
      </div>
    </aside>
  );
}
