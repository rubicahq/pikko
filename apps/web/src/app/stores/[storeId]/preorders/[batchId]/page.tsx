import Link from "next/link";
import { redirect, notFound } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { BatchDetailClient } from "@/components/preorders/batch-detail-client";

export default async function BatchDetailPage({
  params,
}: {
  params: Promise<{ storeId: string; batchId: string }>;
}) {
  const { storeId, batchId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const [batchData, productsData, purchasingData, ordersData] = await Promise.all([
    serverFetch<{ batch: Parameters<typeof BatchDetailClient>[0]["batch"] }>(
      `/api/v1/stores/${storeId}/preorders/${batchId}`,
    ),
    serverFetch<{ products: Parameters<typeof BatchDetailClient>[0]["catalog"] }>(
      `/api/v1/stores/${storeId}/products?limit=100`,
    ),
    serverFetch<{ purchasing: Parameters<typeof BatchDetailClient>[0]["purchasing"] }>(
      `/api/v1/stores/${storeId}/preorders/${batchId}/purchasing`,
    ),
    serverFetch<{ orders: Parameters<typeof BatchDetailClient>[0]["orders"] }>(
      `/api/v1/stores/${storeId}/preorders/${batchId}/orders`,
    ),
  ]);
  if (!batchData?.batch) notFound();
  const b = batchData.batch;

  return (
    <div className="mx-auto max-w-dashboard">
      <Link href={`/stores/${storeId}/preorders`} className="text-sm text-foreground-secondary hover:text-foreground">
        ← Preorders
      </Link>
      <div className="mt-4 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-[30px] font-medium text-foreground">{b.name}</h1>
          <p className="mt-2 text-sm text-foreground-secondary capitalize">
            {b.operational_status.replaceAll("_", " ")} · {b.publication_status}
          </p>
        </div>
      </div>
      <BatchDetailClient
        storeId={storeId}
        batch={b}
        catalog={productsData?.products || []}
        purchasing={purchasingData?.purchasing || []}
        orders={ordersData?.orders || []}
      />
    </div>
  );
}
