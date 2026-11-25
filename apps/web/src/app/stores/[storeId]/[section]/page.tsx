import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";

const PHASE_REDIRECT: Record<string, (storeId: string) => string> = {
  storefront: urls.storeStorefront,
  catalog: urls.storeProducts,
  products: urls.storeProducts,
  catalogs: urls.storeCatalogs,
  attributes: urls.storeAttributes,
  settings: urls.storeSettings,
  orders: urls.storeOrders,
  preorders: (storeId) => `/stores/${storeId}/preorders`,
  "business-rules": (storeId) => `/stores/${storeId}/business-rules`,
  customers: urls.storeCustomers,
  analytics: urls.storeAnalytics,
  notifications: urls.storeNotifications,
};

export default async function StoreSectionPlaceholder({
  params,
}: {
  params: Promise<{ storeId: string; section: string }>;
}) {
  const { storeId, section } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  if (section === "overview") redirect(urls.storeOverview(storeId));
  if (PHASE_REDIRECT[section]) redirect(PHASE_REDIRECT[section](storeId));
  redirect(urls.storeOverview(storeId));
}
