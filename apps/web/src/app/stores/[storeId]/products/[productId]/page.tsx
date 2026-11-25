import { redirect } from "next/navigation";
import {
  getSession,
  getStore,
  getProduct,
  getStoreCategories,
} from "@/lib/server-api";
import { ProductForm } from "@/components/catalog/product-form";
import { ProductRelations } from "@/components/catalog/product-relations";
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
  if (!PRODUCT_ID_UUID.test(productId)) {
    redirect(urls.storeProducts(storeId));
  }
  const [product, categories] = await Promise.all([
    getProduct(storeId, productId),
    getStoreCategories(storeId),
  ]);
  if (!product) redirect(urls.storeProducts(storeId));
  return (
    <div className="space-y-8">
      <ProductForm storeId={storeId} categories={categories} product={product} />
      <ProductRelations storeId={storeId} productId={productId} />
    </div>
  );
}
