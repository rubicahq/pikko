import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { ImportReview } from "@/components/catalog/import-review";

export default async function ProductsImportJobPage({
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
          <Link href={urls.storeProducts(storeId)} className="hover:text-foreground">
            Products
          </Link>
          <span className="mx-1.5">/</span>
          <Link href={urls.productsImport(storeId)} className="hover:text-foreground">
            Import CSV
          </Link>
          <span className="mx-1.5">/</span>
          Review
        </p>
        <h1 className="mt-2 text-[30px] font-medium tracking-tight text-foreground">
          Review import
        </h1>
      </div>
      <ImportReview storeId={storeId} jobId={jobId} backHref={urls.storeProducts(storeId)} />
    </div>
  );
}
