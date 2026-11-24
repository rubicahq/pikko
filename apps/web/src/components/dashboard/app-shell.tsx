import type { Store as StoreType, User } from "@/lib/api";
import { DashboardSidebar } from "@/components/dashboard/sidebar";
import { MobileNav } from "@/components/dashboard/mobile-nav";
import { NotificationBell } from "@/components/notifications/notification-bell";

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
  return (
    <div className="flex min-h-screen bg-background">
      <div className="sticky top-0 hidden h-screen md:block">
        <DashboardSidebar user={user} stores={stores} currentStore={currentStore} />
      </div>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="sticky top-0 z-30 flex h-topbar items-center justify-between gap-3 border-b border-border bg-background/95 px-4 backdrop-blur supports-[backdrop-filter]:bg-background/80 md:px-8">
          <div className="flex min-w-0 items-center gap-3">
            <MobileNav user={user} stores={stores} currentStore={currentStore} />
            <div className="min-w-0">
              <p className="truncate text-sm font-medium text-foreground">{currentStore.name}</p>
              <p className="truncate text-caption text-foreground-muted">{currentStore.slug}</p>
            </div>
          </div>
          <NotificationBell storeId={currentStore.id} />
        </header>
        <main className="flex-1 px-4 py-6 md:px-8 md:py-8">{children}</main>
      </div>
    </div>
  );
}
