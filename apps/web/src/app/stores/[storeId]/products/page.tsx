import Link from "next/link";
import { redirect } from "next/navigation";
import {
  getSession,
  getStore,
  getProducts,
  getStoreCategories,
} from "@/lib/server-api";
import { Button } from "@/components/ui/button";
import { CategoryManager } from "@/components/catalog/category-manager";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

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
  const [productPage, categories] = await Promise.all([
    getProducts(storeId),
    getStoreCategories(storeId),
  ]);
  const products = productPage.products ?? [];

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

      <CategoryManager storeId={storeId} categories={categories} />

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
        <Table stickyHeader>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead sticky>Product</TableHead>
              <TableHead sticky>SKU</TableHead>
              <TableHead sticky className="text-right">
                Price
              </TableHead>
              <TableHead sticky>Status</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {products.map((p) => (
              <TableRow key={p.id}>
                <TableCell>
                  <Link
                    href={urls.storeProduct(storeId, p.id)}
                    className="font-medium text-foreground hover:underline"
                  >
                    {p.name}
                  </Link>
                </TableCell>
                <TableCell className="text-foreground-secondary">{p.sku || "—"}</TableCell>
                <TableCell className="text-right tabular-nums">{formatIDR(p.base_price_idr)}</TableCell>
                <TableCell>
                  <StatusBadge status={p.status} />
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
