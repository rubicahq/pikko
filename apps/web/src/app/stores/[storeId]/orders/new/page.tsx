import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { PageHeader } from "@/components/ui/page-header";
import { OrderEntryGrid } from "@/components/orders/order-entry-grid";

export default async function NewOrderPage({
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
          title="New order"
          description="Spreadsheet-style entry with autocomplete, keyboard navigation, and draft autosave."
        />
      </div>
      <OrderEntryGrid storeId={storeId} />
    </div>
  );
}
