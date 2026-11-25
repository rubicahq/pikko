import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, getStore, serverFetch } from "@/lib/server-api";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { CatalogDetailLinks } from "@/components/catalog/catalog-detail-links";
import { CatalogCoverEditor } from "@/components/catalog/catalog-cover-editor";
import { CatalogStatusControls } from "@/components/catalog/catalog-status-controls";
import { urls } from "@/lib/urls";

const CATALOG_ID_UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

type CatalogDoc = {
  id: string;
  title: string;
  status: string;
  page_count: number;
  source_import_job_id?: string;
  cover_file_id?: string | null;
  campaign_ids?: string[];
  preorder_batch_ids?: string[];
  created_at: string;
};

export default async function CatalogDetailPage({
  params,
}: {
  params: Promise<{ storeId: string; catalogId: string }>;
}) {
  const { storeId, catalogId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());
  if (!CATALOG_ID_UUID.test(catalogId)) {
    redirect(urls.storeCatalogs(storeId));
  }

  const data = await serverFetch<{ catalog: CatalogDoc }>(
    `/api/v1/stores/${storeId}/catalogs/${catalogId}`,
  );
  if (!data?.catalog) redirect(urls.storeCatalogs(storeId));
  const catalog = data.catalog;

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <div>
        <p className="text-sm text-foreground-secondary">
          <Link href={urls.storeCatalogs(storeId)} className="hover:text-foreground">
            Catalogs
          </Link>
          <span className="mx-1.5">/</span>
          {catalog.title}
        </p>
        <div className="mt-2 flex flex-wrap items-start justify-between gap-4">
          <div>
            <h1 className="text-[30px] font-medium tracking-tight text-foreground">
              {catalog.title}
            </h1>
            <div className="mt-2 flex flex-wrap items-center gap-3 text-sm text-foreground-secondary">
              <StatusBadge status={catalog.status} />
              <span>{catalog.page_count} pages</span>
            </div>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button asChild variant="outline">
              <Link href={urls.catalogsFlipbook(storeId)}>Open flipbook</Link>
            </Button>
          </div>
        </div>
      </div>

      <CatalogCoverEditor
        storeId={storeId}
        catalogId={catalog.id}
        coverFileId={catalog.cover_file_id}
      />

      <CatalogStatusControls
        storeId={storeId}
        catalogId={catalog.id}
        status={catalog.status}
      />

      <CatalogDetailLinks
        storeId={storeId}
        catalogId={catalog.id}
        campaignIds={(catalog.campaign_ids || []).map(String)}
        preorderBatchIds={(catalog.preorder_batch_ids || []).map(String)}
      />
    </div>
  );
}
