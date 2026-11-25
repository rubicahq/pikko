import { redirect } from "next/navigation";
import { getSession, getStore, getStoreAttributes } from "@/lib/server-api";
import { AttributesPanel } from "@/components/catalog/attributes-panel";
import { urls } from "@/lib/urls";

export default async function AttributesPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());
  const attributes = await getStoreAttributes(storeId);
  return <AttributesPanel storeId={storeId} attributes={attributes} />;
}
