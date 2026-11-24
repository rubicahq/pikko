import { redirect } from "next/navigation";
import {
  getSession,
  getStore,
  getProduct,
  getStoreCategories,
} from "@/lib/server-api";
import { ProductForm } from "@/components/catalog/product-form";
import { urls } from "@/lib/urls";

const PRODUCT_ID_UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

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
  // Product IDs are UUIDs. If a static catalog segment (new/import/flipbook)
  // ever lands here, bounce to the catalog index instead of calling the API.
  if (!PRODUCT_ID_UUID.test(productId)) {
    redirect(urls.storeCatalog(storeId));
  }
  const [product, categories] = await Promise.all([
    getProduct(storeId, productId),
    getStoreCategories(storeId),
  ]);
  if (!product) redirect(urls.storeCatalog(storeId));
  return <ProductForm storeId={storeId} categories={categories} product={product} />;
}
