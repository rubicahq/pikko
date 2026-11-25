import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { CatalogPagesProgress } from "@/components/catalog/catalog-pages-progress";

export default async function CatalogsImportJobPage({
  params,
}: {
  params: Promise<{ storeId: string; jobId: string }>;
}) {
  const { storeId, jobId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  return (
    <div className="mx-auto max-w-3xl space-y-6">
      <div>
        <p className="text-sm text-foreground-secondary">
          <Link href={urls.storeCatalogs(storeId)} className="hover:text-foreground">
            Catalogs
          </Link>
          <span className="mx-1.5">/</span>
          <Link href={urls.storeCatalogsNew(storeId)} className="hover:text-foreground">
            Add Catalog
          </Link>
          <span className="mx-1.5">/</span>
          Preparing
        </p>
        <h1 className="mt-2 text-[30px] font-medium tracking-tight text-foreground">
          Preparing flipbook
        </h1>
      </div>
      <CatalogPagesProgress storeId={storeId} jobId={jobId} />
    </div>
  );
}
