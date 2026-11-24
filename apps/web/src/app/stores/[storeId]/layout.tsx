import { redirect } from "next/navigation";
import { getSession, getStore, getStores } from "@/lib/server-api";
import { DashboardSidebar } from "@/components/dashboard/sidebar";
import { urls } from "@/lib/urls";

export default async function StoreLayout({
  children,
  params,
}: {
  children: React.ReactNode;
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const [stores, store] = await Promise.all([getStores(), getStore(storeId)]);
  if (!store || store.archived_at) {
    redirect(urls.stores());
  }

  return (
    <div className="flex min-h-screen bg-background">
      <div className="hidden md:block">
        <DashboardSidebar user={session.user} stores={stores} currentStore={store} />
      </div>
      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-topbar items-center border-b border-border px-4 md:hidden">
          <span className="truncate text-sm font-medium">{store.name}</span>
        </header>
        <main className="flex-1 px-4 py-6 md:px-8 md:py-8">{children}</main>
      </div>
    </div>
  );
}
