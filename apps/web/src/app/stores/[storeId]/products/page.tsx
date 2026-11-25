import Link from "next/link";
import { redirect } from "next/navigation";
import {
  getSession,
  getStore,
  getProducts,
  getStoreAttributes,
} from "@/lib/server-api";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { ProductsTable } from "@/components/catalog/products-table";


export default async function ProductsPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());
  const productPage = await getProducts(storeId);
  const products = productPage.products ?? [];
  const attributes = await getStoreAttributes(storeId);

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Products"
        description={`SKU, price, variants, and publication status for ${store.name}.`}
        actions={
          <>
            <Button asChild variant="outline">
              <Link href={urls.productsImport(storeId)}>Import CSV</Link>
            </Button>
            <Button asChild>
              <Link href={urls.storeProductNew(storeId)}>Add product</Link>
            </Button>
          </>
        }
      />

      {products.length === 0 ? (
        <EmptyState
          title="No products yet"
          description="Add your first product to start selling."
          action={
            <Button asChild size="sm">
              <Link href={urls.storeProductNew(storeId)}>Add product</Link>
            </Button>
          }
        />
      ) : (
        <ProductsTable storeId={storeId} products={products} attributes={attributes} />
      )}
    </div>
  );
}
