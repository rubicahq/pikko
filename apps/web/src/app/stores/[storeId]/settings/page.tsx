import { redirect } from "next/navigation";
import { getSession, getStore, getStorefront } from "@/lib/server-api";
import { StoreSettingsForm } from "@/components/settings/store-settings-form";
import { urls } from "@/lib/urls";

export default async function StoreSettingsPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const [store, storefront] = await Promise.all([
    getStore(storeId),
    getStorefront(storeId, "draft"),
  ]);
  if (!store || !storefront) redirect(urls.stores());

  return <StoreSettingsForm store={store} storefront={storefront} />;
}
