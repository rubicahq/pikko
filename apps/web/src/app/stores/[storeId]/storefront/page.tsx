import { redirect } from "next/navigation";
import { getSession, getStore, getStorefront, getThemes } from "@/lib/server-api";
import { StorefrontEditor } from "@/components/storefront/storefront-editor";
import { urls } from "@/lib/urls";

export default async function StorefrontPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const [store, draft, themes] = await Promise.all([
    getStore(storeId),
    getStorefront(storeId, "draft"),
    getThemes(),
  ]);
  if (!store || !draft) redirect(urls.stores());

  return <StorefrontEditor store={store} draft={draft} themes={themes} />;
}
