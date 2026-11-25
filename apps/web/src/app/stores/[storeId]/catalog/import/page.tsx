import { redirect } from "next/navigation";
import { urls } from "@/lib/urls";

/** Legacy mixed import → Products CSV import (PDF lives under Catalogs → Add Catalog). */
export default async function LegacyCatalogImportRedirect({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  redirect(urls.productsImport(storeId));
}
