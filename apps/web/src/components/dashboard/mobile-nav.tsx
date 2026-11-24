"use client";

import { useState } from "react";
import Link from "next/link";
import { Menu, UserRound } from "lucide-react";
import type { Store as StoreType, User } from "@/lib/api";
import { Wordmark } from "@/components/brand/wordmark";
import { StoreSwitcher } from "@/components/dashboard/store-switcher";
import { SidebarNav } from "@/components/dashboard/sidebar-nav";
import { Sheet, SheetContent, SheetHeader, SheetTitle, SheetTrigger } from "@/components/ui/sheet";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";

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
  const storeId = currentStore?.id;

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
          <Wordmark href={storeId ? urls.storeOverview(storeId) : urls.stores()} />
        </SheetHeader>
        <div className="px-3 pb-4 pt-2">
          <StoreSwitcher stores={stores} current={currentStore} />
        </div>
        <SidebarNav
          storeId={storeId}
          permissions={permissions}
          onNavigate={() => setOpen(false)}
        />
        <div className="mt-auto border-t border-border p-3">
          <Link
            href={urls.accountSettings()}
            onClick={() => setOpen(false)}
            className="flex h-10 items-center gap-2 rounded-md px-3 text-sm text-foreground-secondary hover:bg-surface-hover hover:text-foreground"
          >
            <UserRound className="h-5 w-5 shrink-0" strokeWidth={1.75} />
            <span className="min-w-0 truncate">{user.display_name || user.email}</span>
          </Link>
        </div>
      </SheetContent>
    </Sheet>
  );
}
