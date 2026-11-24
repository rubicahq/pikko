import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Badge } from "@/components/ui/badge";

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
    <div className="mx-auto max-w-dashboard">
      <h1 className="text-[30px] font-medium text-foreground">Orders</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Review customer orders, verify bank transfers, and update fulfillment.
      </p>

      {orders.length === 0 ? (
        <div className="mt-8 rounded-[10px] border border-border p-8">
          <p className="text-sm text-foreground-secondary">You haven&apos;t received any orders yet.</p>
          <p className="mt-2 text-sm text-foreground-secondary">
            Share your storefront to start receiving orders.
          </p>
        </div>
      ) : (
        <div className="mt-8 overflow-hidden rounded-[10px] border border-border">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface text-xs font-medium uppercase tracking-wide text-foreground-secondary">
              <tr>
                <th className="px-4 py-3">Order</th>
                <th className="px-4 py-3">Customer</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3 text-right">Total</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.id} className="border-t border-border hover:bg-background-subtle">
                  <td className="px-4 py-3">
                    <Link
                      href={`/stores/${storeId}/orders/${o.id}`}
                      className="font-medium text-foreground hover:underline"
                    >
                      {o.order_number}
                    </Link>
                    <p className="text-xs text-foreground-muted">
                      {new Date(o.created_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })}
                    </p>
                  </td>
                  <td className="px-4 py-3">{o.customer_name}</td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1">
                      <Badge tone="neutral">{o.order_status.replaceAll("_", " ")}</Badge>
                      <Badge
                        tone={
                          o.payment_status === "paid"
                            ? "success"
                            : o.payment_status === "awaiting_verification"
                              ? "warning"
                              : "neutral"
                        }
                      >
                        {o.payment_status.replaceAll("_", " ")}
                      </Badge>
                    </div>
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums">{formatIDR(o.total_idr)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
