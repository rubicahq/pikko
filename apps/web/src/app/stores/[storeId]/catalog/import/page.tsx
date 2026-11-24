import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { ImportUpload } from "@/components/catalog/import-upload";

type Job = {
  id: string;
  status: string;
  item_count: number;
  created_at: string;
  progress_pct: number;
};

export default async function CatalogImportPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const data = await serverFetch<{ jobs: Job[] }>(
    `/api/v1/stores/${storeId}/catalog-imports`,
  );
  const jobs = data?.jobs || [];

  return (
    <div className="mx-auto max-w-reading space-y-8">
      <div>
        <p className="text-sm text-foreground-secondary">
          <Link href={urls.storeCatalog(storeId)} className="hover:text-foreground">
            Catalog
          </Link>
          <span className="mx-1.5">/</span>
          Import
        </p>
        <h1 className="mt-2 text-[30px] font-medium tracking-tight text-foreground">
          Catalog import
        </h1>
      </div>

      <ImportUpload storeId={storeId} />

      <div>
        <h2 className="text-base font-medium text-foreground">Recent jobs</h2>
        {jobs.length === 0 ? (
          <p className="mt-2 text-sm text-foreground-secondary">No imports yet.</p>
        ) : (
          <ul className="mt-3 divide-y divide-border rounded-lg border border-border">
            {jobs.map((j) => (
              <li key={j.id}>
                <Link
                  href={urls.catalogImportJob(storeId, j.id)}
                  className="flex items-center justify-between gap-3 px-4 py-3 text-sm hover:bg-background-subtle"
                >
                  <span className="font-medium text-foreground">{j.status}</span>
                  <span className="text-foreground-secondary">
                    {j.item_count} items · {j.progress_pct}%
                  </span>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
