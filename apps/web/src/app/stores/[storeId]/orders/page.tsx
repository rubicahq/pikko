import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type OrderRow = {
  id: string;
  order_number: string;
  customer_name: string;
  total_idr: number;
  order_status: string;
  payment_status: string;
  fulfillment_status: string;
  created_at: string;
};

export default async function OrdersPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const data = await serverFetch<{ orders: OrderRow[]; total: number }>(
    `/api/v1/stores/${storeId}/orders`,
  );
  const orders = data?.orders || [];

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Orders"
        description="Review customer and admin orders, verify bank transfers, and update fulfillment."
        actions={
          <>
            <Link
              href={`/stores/${storeId}/orders/import`}
              className="inline-flex h-10 items-center rounded-md border border-border px-4 text-sm font-medium text-foreground hover:bg-background-subtle"
            >
              Import CSV
            </Link>
            <Link
              href={`/stores/${storeId}/orders/new`}
              className="inline-flex h-10 items-center rounded-md bg-dark px-4 text-sm font-medium text-foreground-inverse hover:bg-foreground"
            >
              New order
            </Link>
          </>
        }
      />

      {orders.length === 0 ? (
        <EmptyState
          title="No orders yet"
          description="Share your storefront or create an order manually."
          action={
            <Link
              href={`/stores/${storeId}/orders/new`}
              className="text-sm font-medium text-brand-accessible-dark underline"
            >
              Create admin order
            </Link>
          }
        />
      ) : (
        <Table stickyHeader>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead sticky>Order</TableHead>
              <TableHead sticky>Customer</TableHead>
              <TableHead sticky>Status</TableHead>
              <TableHead sticky className="text-right">
                Total
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {orders.map((o) => (
              <TableRow key={o.id}>
                <TableCell>
                  <Link
                    href={`/stores/${storeId}/orders/${o.id}`}
                    className="font-medium text-foreground hover:underline"
                  >
                    {o.order_number}
                  </Link>
                  <p className="text-caption text-foreground-muted">
                    {new Date(o.created_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })}
                  </p>
                </TableCell>
                <TableCell>{o.customer_name}</TableCell>
                <TableCell>
                  <div className="flex flex-wrap gap-1">
                    <StatusBadge status={o.order_status} />
                    <StatusBadge status={o.payment_status} />
                  </div>
                </TableCell>
                <TableCell className="text-right tabular-nums">{formatIDR(o.total_idr)}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
