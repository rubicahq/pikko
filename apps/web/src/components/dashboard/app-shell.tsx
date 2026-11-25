import { Suspense } from "react";
import type { Store as StoreType, User } from "@/lib/api";
import { DashboardSidebar } from "@/components/dashboard/sidebar";
import { MobileNav } from "@/components/dashboard/mobile-nav";
import { ContextualTabs } from "@/components/dashboard/contextual-tabs";
import { HeaderUtilities } from "@/components/dashboard/header-utilities";

export function AppShell({
  user,
  stores,
  currentStore,
  children,
}: {
  user: User;
  stores: StoreType[];
  currentStore: StoreType;
  children: React.ReactNode;
}) {
  const permissions = currentStore.permissions;
  return (
    <div className="flex min-h-screen bg-workspace">
      {/* Tablet: collapsed icon rail */}
      <div className="sticky top-0 hidden h-screen md:block lg:hidden">
        <DashboardSidebar
          user={user}
          stores={stores}
          currentStore={currentStore}
          permissions={permissions}
          collapsed
        />
      </div>
      {/* Desktop: full sidebar */}
      <div className="sticky top-0 hidden h-screen lg:block">
        <DashboardSidebar
          user={user}
          stores={stores}
          currentStore={currentStore}
          permissions={permissions}
        />
      </div>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-topbar items-stretch justify-between gap-3 border-b border-border bg-background px-4 md:px-6">
          <div className="flex min-w-0 flex-1 items-stretch gap-2">
            <div className="flex items-center md:hidden">
              <MobileNav
                user={user}
                stores={stores}
                currentStore={currentStore}
                permissions={permissions}
              />
            </div>
            <Suspense fallback={<div className="min-w-0 flex-1" />}>
              <ContextualTabs
                storeId={currentStore.id}
                permissions={permissions}
              />
            </Suspense>
          </div>
          <div className="flex items-center">
            <HeaderUtilities
              storeId={currentStore.id}
              permissions={permissions}
            />
          </div>
        </header>
        <main className="flex-1 px-4 py-6 md:px-8 md:py-8">{children}</main>
      </div>
    </div>
  );
}
