"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ExternalLink, Sparkles, UserRound } from "lucide-react";
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
  permissions,
  className,
}: {
  user: User;
  stores: StoreType[];
  currentStore?: StoreType | null;
  permissions?: string[] | null;
  className?: string;
}) {
  const pathname = usePathname();
  const storeId = currentStore?.id;
  const storefrontHref =
    currentStore?.storefront_url ||
    (currentStore?.slug ? urls.storefront(currentStore.slug) : null);

  return (
    <aside
      className={cn(
        "flex h-full w-sidebar flex-col border-r border-border bg-background",
        className,
      )}
    >
      <div className="px-4 pb-3 pt-4">
        <Wordmark href={storeId ? urls.storeOverview(storeId) : urls.stores()} />
      </div>
      <div className="px-3 pb-3">
        <StoreSwitcher stores={stores} current={currentStore} />
      </div>
      {storefrontHref ? (
        <div className="px-3 pb-3">
          <a
            href={storefrontHref}
            target="_blank"
            rel="noreferrer"
            className="flex h-10 items-center gap-2 rounded-md px-3 text-sm font-medium text-foreground-secondary transition-colors hover:bg-surface-hover hover:text-foreground"
          >
            <ExternalLink className="h-5 w-5 shrink-0" strokeWidth={1.75} />
            <span className="truncate">View store</span>
          </a>
        </div>
      ) : null}
      <SidebarNav storeId={storeId} permissions={permissions} />
      <div className="mt-auto space-y-1 border-t border-border p-3">
        <div className="rounded-xl bg-warm-feature px-3 py-3">
          <div className="flex items-center gap-2">
            <Sparkles className="h-4 w-4 text-brand" strokeWidth={1.75} />
            <p className="text-sm font-semibold text-foreground">Connie</p>
          </div>
          <p className="mt-1 text-caption text-foreground-secondary">
            Your commerce teammate — coming soon.
          </p>
        </div>
        {storeId ? (
          <Link
            href={urls.storeSettings(storeId)}
            className={cn(
              "flex h-10 items-center gap-2 rounded-md px-3 text-sm font-medium text-foreground-secondary hover:bg-surface-hover hover:text-foreground",
              pathname.includes("/settings") && "bg-surface-active text-foreground",
            )}
          >
            Settings
          </Link>
        ) : null}
        <Link
          href={urls.accountSettings()}
          className={cn(
            "flex h-10 items-center gap-2 rounded-md px-3 text-sm text-foreground-secondary hover:bg-surface-hover hover:text-foreground",
            pathname.startsWith("/account") && "bg-surface-active font-medium text-foreground",
          )}
        >
          <UserRound className="h-5 w-5 shrink-0" strokeWidth={1.75} />
          <span className="min-w-0 truncate">{user.display_name || user.email}</span>
        </Link>
      </div>
    </aside>
  );
}
