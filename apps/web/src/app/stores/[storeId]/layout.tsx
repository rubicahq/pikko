import type { Metadata } from "next";
import { redirect } from "next/navigation";
import { getSession, getStore, getStores } from "@/lib/server-api";
import { AppShell } from "@/components/dashboard/app-shell";
import { urls } from "@/lib/urls";

export const metadata: Metadata = {
  robots: { index: false, follow: false },
};

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
    <AppShell user={session.user} stores={stores} currentStore={store}>
      {children}
    </AppShell>
  );
}
