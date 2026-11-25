"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { ExternalLink } from "lucide-react";
import type { Store as StoreType, User } from "@/lib/api";
import { Wordmark } from "@/components/brand/wordmark";
import { StoreSwitcher } from "@/components/dashboard/store-switcher";
import { SidebarNav } from "@/components/dashboard/sidebar-nav";
import { AccountMenu } from "@/components/dashboard/account-menu";
import {
  liveSettingsWorkspace,
  matchWorkspace,
  storeHref,
} from "@/components/dashboard/nav-config";
import { cn } from "@/lib/utils";
import { urls } from "@/lib/urls";

export function DashboardSidebar({
  user,
  stores,
  currentStore,
  permissions,
  className,
  collapsed = false,
}: {
  user: User;
  stores: StoreType[];
  currentStore?: StoreType | null;
  permissions?: string[] | null;
  className?: string;
  collapsed?: boolean;
}) {
  const pathname = usePathname();
  const storeId = currentStore?.id;
  const storefrontHref =
    currentStore?.storefront_url ||
    (currentStore?.slug ? urls.storefront(currentStore.slug) : null);
  const settings = liveSettingsWorkspace(permissions);
  const settingsActive =
    storeId && settings
      ? matchWorkspace(settings, pathname, storeId)
      : false;

  return (
    <aside
      className={cn(
        "flex h-full flex-col border-r border-border bg-background",
        collapsed ? "w-sidebar-collapsed" : "w-sidebar",
        className,
      )}
    >
      <div className={cn("pb-3 pt-4", collapsed ? "px-2" : "px-3")}>
        {collapsed ? (
          <Link
            href={storeId ? urls.storeOverview(storeId) : urls.stores()}
            className="flex h-10 items-center justify-center"
            aria-label="Pikko"
          >
            {/* eslint-disable-next-line @next/next/no-img-element */}
            <img src="/brand/pikko-icon.png" alt="" className="h-7 w-7" />
          </Link>
        ) : (
          <Wordmark href={storeId ? urls.storeOverview(storeId) : urls.stores()} />
        )}
      </div>

      <div className={cn("pb-3", collapsed ? "px-2" : "px-3")}>
        {collapsed ? (
          <StoreSwitcher stores={stores} current={currentStore} compact />
        ) : (
          <StoreSwitcher stores={stores} current={currentStore} />
        )}
      </div>

      {storefrontHref ? (
        <div className={cn("pb-3", collapsed ? "px-2" : "px-3")}>
          <a
            href={storefrontHref}
            target="_blank"
            rel="noreferrer"
            title={collapsed ? "View store" : undefined}
            className={cn(
              "flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium text-foreground-secondary transition-colors hover:bg-background-subtle hover:text-foreground",
              collapsed && "justify-center px-0",
            )}
          >
            <ExternalLink className="h-5 w-5 shrink-0" strokeWidth={1.75} />
            {!collapsed ? <span className="truncate">View store</span> : (
              <span className="sr-only">View store</span>
            )}
          </a>
        </div>
      ) : null}

      <div className={cn("mx-3 border-t border-border", collapsed && "mx-2")} />

      <div className="pt-3">
        <SidebarNav
          storeId={storeId}
          permissions={permissions}
          collapsed={collapsed}
        />
      </div>

      <div
        className={cn(
          "mt-auto space-y-1 border-t border-border p-3",
          collapsed && "px-2",
        )}
      >
        {storeId && settings ? (
          <Link
            href={storeHref(storeId, settings.defaultHref)}
            title={collapsed ? settings.label : undefined}
            className={cn(
              "flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium text-foreground-secondary hover:bg-surface-hover hover:text-foreground",
              settingsActive && "bg-surface-active font-semibold text-foreground",
              collapsed && "justify-center px-0",
            )}
            aria-current={settingsActive ? "page" : undefined}
          >
            <settings.icon
              className={cn("h-5 w-5 shrink-0", settingsActive && "text-brand")}
              strokeWidth={1.75}
            />
            {!collapsed ? <span className="truncate">{settings.label}</span> : (
              <span className="sr-only">{settings.label}</span>
            )}
          </Link>
        ) : null}

        <AccountMenu user={user} collapsed={collapsed} />
      </div>
    </aside>
  );
}
