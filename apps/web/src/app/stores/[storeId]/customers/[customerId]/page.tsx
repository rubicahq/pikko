import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, getStore, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Badge } from "@/components/ui/badge";
import { CustomerNotesForm } from "@/components/customers/customer-notes-form";
import { CustomerPortalLink } from "@/components/customers/customer-portal-link";

type CustomerDetail = {
  id: string;
  name: string;
  email?: string | null;
  phone?: string | null;
  total_orders: number;
  completed_order_value_idr: number;
  internal_notes?: string | null;
  first_order_at?: string | null;
  last_order_at?: string | null;
  orders?: Array<{
    id: string;
    order_number: string;
    total_idr: number;
    order_status: string;
    payment_status: string;
    created_at: string;
  }>;
};

export default async function CustomerDetailPage({
  params,
}: {
  params: Promise<{ storeId: string; customerId: string }>;
}) {
  const { storeId, customerId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const [data, store] = await Promise.all([
    serverFetch<{ customer: CustomerDetail; portal_url?: string; store_slug?: string }>(
      `/api/v1/stores/${storeId}/customers/${customerId}`,
    ),
    getStore(storeId),
  ]);
  if (!data?.customer) redirect(urls.storeCustomers(storeId));
  const c = data.customer;
  const orders = c.orders || [];
  const portalUrl =
    data.portal_url ||
    (data.store_slug
      ? urls.portalAbsolute(data.store_slug)
      : store?.slug
        ? urls.portalAbsolute(store.slug)
        : null);

  return (
    <div className="mx-auto max-w-dashboard">
      <Link
        href={urls.storeCustomers(storeId)}
        className="text-sm text-foreground-secondary hover:text-foreground"
      >
        ← Customers
      </Link>
      <h1 className="mt-4 text-[30px] font-medium text-foreground">{c.name}</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        {[c.phone, c.email].filter(Boolean).join(" · ") || "No contact yet"}
      </p>

      {portalUrl ? (
        <section className="mt-6 rounded-[10px] border border-border bg-background-subtle/40 p-5">
          <CustomerPortalLink url={portalUrl} />
        </section>
      ) : null}

      <div className="mt-8 grid gap-4 sm:grid-cols-3">
        <div className="rounded-[10px] border border-border p-5">
          <p className="text-xs font-medium uppercase text-foreground-muted">Orders</p>
          <p className="mt-2 text-[32px] font-medium tabular-nums leading-[38px]">{c.total_orders}</p>
        </div>
        <div className="rounded-[10px] border border-border p-5">
          <p className="text-xs font-medium uppercase text-foreground-muted">Completed value</p>
          <p className="mt-2 text-[32px] font-medium tabular-nums leading-[38px]">
            {formatIDR(c.completed_order_value_idr)}
          </p>
        </div>
        <div className="rounded-[10px] border border-border p-5">
          <p className="text-xs font-medium uppercase text-foreground-muted">Last order</p>
          <p className="mt-2 text-sm text-foreground">
            {c.last_order_at
              ? new Date(c.last_order_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })
              : "—"}
          </p>
        </div>
      </div>

      <section className="mt-8 rounded-[10px] border border-border p-5">
        <h2 className="text-base font-medium">Internal notes</h2>
        <p className="mt-1 text-sm text-foreground-secondary">Visible only to your store team.</p>
        <div className="mt-4">
          <CustomerNotesForm
            storeId={storeId}
            customerId={c.id}
            initialNotes={c.internal_notes || ""}
          />
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-xl font-medium text-foreground">Order history</h2>
        {orders.length === 0 ? (
          <p className="mt-4 text-sm text-foreground-secondary">No orders linked yet.</p>
        ) : (
          <div className="mt-4 overflow-hidden rounded-[10px] border border-border">
            <table className="w-full text-left text-sm">
              <thead className="bg-surface text-xs font-medium uppercase tracking-wide text-foreground-secondary">
                <tr>
                  <th className="px-4 py-3">Order</th>
                  <th className="px-4 py-3">Status</th>
                  <th className="px-4 py-3 text-right">Total</th>
                </tr>
              </thead>
              <tbody>
                {orders.map((o) => (
                  <tr key={o.id} className="border-t border-border">
                    <td className="px-4 py-3">
                      <Link
                        href={urls.storeOrder(storeId, o.id)}
                        className="font-medium hover:underline"
                      >
                        {o.order_number}
                      </Link>
                      <p className="text-xs text-foreground-muted">
                        {new Date(o.created_at).toLocaleString("id-ID", {
                          timeZone: "Asia/Jakarta",
                        })}
                      </p>
                    </td>
                    <td className="px-4 py-3">
                      <div className="flex flex-wrap gap-1">
                        <Badge tone="neutral">{o.order_status.replaceAll("_", " ")}</Badge>
                        <Badge tone={o.payment_status === "paid" ? "success" : "neutral"}>
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
      </section>
    </div>
  );
}
