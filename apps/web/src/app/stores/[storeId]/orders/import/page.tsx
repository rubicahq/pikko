import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { PageHeader } from "@/components/ui/page-header";
import { OrderCsvImport } from "@/components/orders/order-csv-import";

export default async function OrderImportPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  return (
    <div className="mx-auto max-w-dashboard space-y-6">
      <div>
        <Link
          href={`/stores/${storeId}/orders`}
          className="text-sm text-foreground-secondary hover:text-foreground"
        >
          ← Orders
        </Link>
        <PageHeader
          className="mt-2"
          title="Import orders"
          description="Upload a CSV, preview customer/product matches, then create orders in the same operational system."
        />
      </div>
      <OrderCsvImport storeId={storeId} />
    </div>
  );
}
