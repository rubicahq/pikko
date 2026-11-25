import { redirect } from "next/navigation";
import { getSession, getStore, getStoreAttributes } from "@/lib/server-api";
import { AttributeForm } from "@/components/catalog/attribute-form";
import { urls } from "@/lib/urls";

const ATTR_ID_UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default async function EditAttributePage({
  params,
}: {
  params: Promise<{ storeId: string; attributeId: string }>;
}) {
  const { storeId, attributeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());
  if (!ATTR_ID_UUID.test(attributeId)) {
    redirect(urls.storeAttributes(storeId));
  }
  const attributes = await getStoreAttributes(storeId);
  const attribute = attributes.find((a) => a.id === attributeId) || null;
  if (!attribute) redirect(urls.storeAttributes(storeId));
  return <AttributeForm storeId={storeId} attribute={attribute} />;
}
