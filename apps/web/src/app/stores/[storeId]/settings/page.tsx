import { redirect } from "next/navigation";
import { getSession, getStore, getStorefront } from "@/lib/server-api";
import { StoreSettingsForm } from "@/components/settings/store-settings-form";
import { CommerceSettings } from "@/components/settings/commerce-settings";
import { urls } from "@/lib/urls";

export default async function StoreSettingsPage({
  params,
  searchParams,
}: {
  params: Promise<{ storeId: string }>;
  searchParams: Promise<{ tab?: string }>;
}) {
  const { storeId } = await params;
  const { tab } = await searchParams;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const [store, storefront] = await Promise.all([
    getStore(storeId),
    getStorefront(storeId, "draft"),
  ]);
  if (!store || !storefront) redirect(urls.stores());

  const payment = tab === "payment";

  return (
    <div>
      {payment ? (
        <div className="mx-auto max-w-reading space-y-4">
          <div>
            <h1 className="text-[30px] font-medium text-foreground">Payment Settings</h1>
            <p className="mt-2 text-sm text-foreground-secondary">
              Bank transfer methods and fulfillment fees for this store.
            </p>
          </div>
          <CommerceSettings storeId={storeId} />
        </div>
      ) : (
        <>
          <StoreSettingsForm store={store} storefront={storefront} />
        </>
      )}
    </div>
  );
}
