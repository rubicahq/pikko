import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, getStore } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { ImportUpload } from "@/components/catalog/import-upload";

export default async function AddCatalogPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());

  return (
    <div className="mx-auto max-w-reading space-y-8">
      <div>
        <p className="text-sm text-foreground-secondary">
          <Link href={urls.storeCatalogs(storeId)} className="hover:text-foreground">
            Catalogs
          </Link>
          <span className="mx-1.5">/</span>
          Add Catalog
        </p>
        <h1 className="mt-2 text-[30px] font-medium tracking-tight text-foreground">
          Add Catalog
        </h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Set a catalog name first, then upload a PDF. We split it into flipbook pages for display —
          no AI product detection.
        </p>
      </div>

      <ImportUpload storeId={storeId} source="pdf" />
    </div>
  );
}
