import { redirect } from "next/navigation";
import { getSession, getStore, getStoreAttributes } from "@/lib/server-api";
import { ProductForm } from "@/components/catalog/product-form";
import { urls } from "@/lib/urls";

export default async function NewProductPage({
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
  return <ProductForm storeId={storeId} attributes={attributes} />;
}
