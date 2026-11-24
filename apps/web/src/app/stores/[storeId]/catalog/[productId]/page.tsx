import { redirect } from "next/navigation";
import {
  getSession,
  getStore,
  getProduct,
  getStoreCategories,
} from "@/lib/server-api";
import { ProductForm } from "@/components/catalog/product-form";
import { urls } from "@/lib/urls";

export default async function EditProductPage({
  params,
}: {
  params: Promise<{ storeId: string; productId: string }>;
}) {
  const { storeId, productId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());
  const [product, categories] = await Promise.all([
    getProduct(storeId, productId),
    getStoreCategories(storeId),
  ]);
  if (!product) redirect(urls.storeCatalog(storeId));
  return <ProductForm storeId={storeId} categories={categories} product={product} />;
}
