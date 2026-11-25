import { redirect } from "next/navigation";
import { getSession, getStore, getStoreCategories } from "@/lib/server-api";
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
  const categories = await getStoreCategories(storeId);
  return <ProductForm storeId={storeId} categories={categories} />;
}
