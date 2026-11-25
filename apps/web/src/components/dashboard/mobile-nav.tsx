"use client";

import { useState } from "react";
import { Menu } from "lucide-react";
import type { Store as StoreType, User } from "@/lib/api";
import { Wordmark } from "@/components/brand/wordmark";
import { StoreSwitcher } from "@/components/dashboard/store-switcher";
import { SidebarNav } from "@/components/dashboard/sidebar-nav";
import { AccountMenu } from "@/components/dashboard/account-menu";
import {
  CONNIE_NAV,
  liveSettingsWorkspace,
  matchWorkspace,
  storeHref,
} from "@/components/dashboard/nav-config";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";
import { cn } from "@/lib/utils";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { ExternalLink, Sparkles } from "lucide-react";

export function MobileNav({
  user,
  stores,
  currentStore,
  permissions,
}: {
  user: User;
  stores: StoreType[];
  currentStore?: StoreType | null;
  permissions?: string[] | null;
}) {
  const [open, setOpen] = useState(false);
  const pathname = usePathname();
  const storeId = currentStore?.id;
  const settings = liveSettingsWorkspace(permissions);
  const settingsActive =
    storeId && settings ? matchWorkspace(settings, pathname, storeId) : false;
  const storefrontHref =
    currentStore?.storefront_url ||
    (currentStore?.slug ? urls.storefront(currentStore.slug) : null);

  return (
    <Sheet open={open} onOpenChange={setOpen}>
      <SheetTrigger asChild>
        <Button type="button" variant="outline" size="icon" className="md:hidden" aria-label="Open menu">
          <Menu className="h-5 w-5" strokeWidth={1.75} />
        </Button>
      </SheetTrigger>
      <SheetContent side="left" className="w-sidebar p-0">
        <SheetHeader>
          <SheetTitle className="sr-only">Store navigation</SheetTitle>
          <div className="px-3 pt-4">
            <Wordmark href={storeId ? urls.storeOverview(storeId) : urls.stores()} />
          </div>
        </SheetHeader>
        <div className="px-3 pb-3 pt-2">
          <StoreSwitcher stores={stores} current={currentStore} />
        </div>
        {storefrontHref ? (
          <div className="px-3 pb-3">
            <a
              href={storefrontHref}
              target="_blank"
              rel="noreferrer"
              onClick={() => setOpen(false)}
              className="flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium text-foreground-secondary hover:bg-background-subtle hover:text-foreground"
            >
              <ExternalLink className="h-5 w-5 shrink-0" strokeWidth={1.75} />
              <span className="truncate">View store</span>
            </a>
          </div>
        ) : null}
        <div className="mx-3 border-t border-border" />
        <div className="pt-3">
          <SidebarNav
            storeId={storeId}
            permissions={permissions}
            onNavigate={() => setOpen(false)}
          />
        </div>
        <div className="mt-auto space-y-1 border-t border-border p-3">
          <div
            className="flex h-11 items-center gap-3 rounded-md bg-warm-feature px-3 text-sm font-medium text-foreground"
            title="Connie — coming soon"
          >
            <Sparkles className="h-5 w-5 shrink-0 text-brand" strokeWidth={1.75} />
            <span>{CONNIE_NAV.label}</span>
            <span className="ml-auto rounded-sm bg-background/70 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-foreground-muted">
              Soon
            </span>
          </div>
          {storeId && settings ? (
            <Link
              href={storeHref(storeId, settings.defaultHref)}
              onClick={() => setOpen(false)}
              className={cn(
                "flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium text-foreground-secondary hover:bg-surface-hover hover:text-foreground",
                settingsActive && "bg-surface-active font-semibold text-foreground",
              )}
            >
              <settings.icon
                className={cn("h-5 w-5 shrink-0", settingsActive && "text-brand")}
                strokeWidth={1.75}
              />
              <span className="truncate">{settings.label}</span>
            </Link>
          ) : null}
          <AccountMenu user={user} />
        </div>
      </SheetContent>
    </Sheet>
  );
}
