import Link from "next/link";
import { redirect } from "next/navigation";
import {
  getSession,
  getStore,
  getProducts,
  getStoreCategories,
} from "@/lib/server-api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CategoryManager } from "@/components/catalog/category-manager";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";

export default async function CatalogPage({
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
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-[30px] font-medium text-foreground">Catalog</h1>
          <p className="mt-2 text-sm text-foreground-secondary">
            Products, categories, images, and publication status for {store.name}.
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Button asChild variant="outline">
            <Link href={urls.catalogFlipbook(storeId)}>Flipbook</Link>
          </Button>
          <Button asChild variant="outline">
            <Link href={urls.catalogImport(storeId)}>Import</Link>
          </Button>
          <Button asChild>
            <Link href={urls.storeCatalogNew(storeId)}>Add product</Link>
          </Button>
        </div>
      </div>

      <CategoryManager storeId={storeId} categories={categories} />

      <div className="overflow-hidden rounded-lg border border-border">
        <table className="w-full text-left text-sm">
          <thead className="bg-background-subtle text-xs font-medium text-foreground-secondary">
            <tr>
              <th className="px-4 py-3">Product</th>
              <th className="px-4 py-3">SKU</th>
              <th className="px-4 py-3 text-right">Price</th>
              <th className="px-4 py-3">Status</th>
            </tr>
          </thead>
          <tbody>
            {products.length === 0 && (
              <tr>
                <td colSpan={4} className="px-4 py-10 text-center text-foreground-secondary">
                  No products yet. Add your first product to start selling.
                </td>
              </tr>
            )}
            {products.map((p) => (
              <tr key={p.id} className="border-t border-border hover:bg-background-subtle">
                <td className="px-4 py-3">
                  <Link
                    href={urls.storeCatalogProduct(storeId, p.id)}
                    className="font-medium text-foreground hover:underline"
                  >
                    {p.name}
                  </Link>
                </td>
                <td className="px-4 py-3 text-foreground-secondary">{p.sku || "—"}</td>
                <td className="px-4 py-3 text-right tabular-nums">{formatIDR(p.base_price_idr)}</td>
                <td className="px-4 py-3">
                  <Badge tone={p.status === "published" ? "success" : "neutral"}>{p.status}</Badge>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
