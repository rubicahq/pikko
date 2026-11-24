import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";

type Analytics = {
  timezone: string;
  from: string;
  to: string;
  total_orders: number;
  paid_orders: number;
  verified_payment_value_idr: number;
  unpaid_order_value_idr: number;
  total_order_value_idr: number;
  unique_customers: number;
  conversion_rate: number;
  daily: Array<{
    date: string;
    orders: number;
    paid_orders: number;
    paid_value_idr: number;
    order_value_idr: number;
  }>;
  top_products: Array<{
    product_id: string;
    product_name: string;
    quantity: number;
    revenue_idr: number;
  }>;
};

export default async function AnalyticsPage({
  params,
  searchParams,
}: {
  params: Promise<{ storeId: string }>;
  searchParams: Promise<{ from?: string; to?: string }>;
}) {
  const { storeId } = await params;
  const sp = await searchParams;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const q = new URLSearchParams();
  if (sp.from) q.set("from", sp.from);
  if (sp.to) q.set("to", sp.to);
  q.set("tz", "Asia/Jakarta");
  const qs = q.toString() ? `?${q}` : "?tz=Asia/Jakarta";

  const data = await serverFetch<{ analytics: Analytics }>(
    `/api/v1/stores/${storeId}/analytics${qs}`,
  );
  const a = data?.analytics;

  return (
    <div className="mx-auto max-w-dashboard">
      <h1 className="text-[30px] font-medium text-foreground">Analytics</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Sales and conversion for this store · timezone Asia/Jakarta
        {a ? ` · ${a.from} → ${a.to}` : ""}
      </p>

      <form className="mt-6 flex flex-wrap items-end gap-3" method="get">
        <label className="space-y-1 text-sm">
          <span className="text-foreground-secondary">From</span>
          <input
            type="date"
            name="from"
            defaultValue={sp.from || a?.from || ""}
            className="block h-11 rounded-md border border-border bg-background px-3 text-sm"
          />
        </label>
        <label className="space-y-1 text-sm">
          <span className="text-foreground-secondary">To</span>
          <input
            type="date"
            name="to"
            defaultValue={sp.to || a?.to || ""}
            className="block h-11 rounded-md border border-border bg-background px-3 text-sm"
          />
        </label>
        <button
          type="submit"
          className="h-11 rounded-md border border-border bg-background px-4 text-sm font-medium hover:bg-surface"
        >
          Apply
        </button>
      </form>

      {!a ? (
        <p className="mt-8 text-sm text-foreground-secondary">Unable to load analytics.</p>
      ) : (
        <>
          <div className="mt-8 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <Metric label="Verified payments" value={formatIDR(a.verified_payment_value_idr)} />
            <Metric label="Paid orders" value={String(a.paid_orders)} />
            <Metric label="Total orders" value={String(a.total_orders)} />
            <Metric
              label="Conversion (paid / orders)"
              value={`${Math.round(a.conversion_rate * 1000) / 10}%`}
            />
          </div>
          <p className="mt-3 text-xs text-foreground-muted">
            Unpaid order value {formatIDR(a.unpaid_order_value_idr)} is not counted as revenue.
            Unique purchasing customers: {a.unique_customers}.
          </p>

          <section className="mt-10">
            <h2 className="text-xl font-medium">Daily activity</h2>
            {a.daily.length === 0 ? (
              <p className="mt-4 text-sm text-foreground-secondary">No orders in this range.</p>
            ) : (
              <div className="mt-4 overflow-hidden rounded-[10px] border border-border">
                <table className="w-full text-left text-sm">
                  <thead className="bg-surface text-xs font-medium uppercase tracking-wide text-foreground-secondary">
                    <tr>
                      <th className="px-4 py-3">Date</th>
                      <th className="px-4 py-3 text-right">Orders</th>
                      <th className="px-4 py-3 text-right">Paid</th>
                      <th className="px-4 py-3 text-right">Paid value</th>
                    </tr>
                  </thead>
                  <tbody>
                    {a.daily.map((d) => (
                      <tr key={d.date} className="border-t border-border">
                        <td className="px-4 py-3">{d.date}</td>
                        <td className="px-4 py-3 text-right tabular-nums">{d.orders}</td>
                        <td className="px-4 py-3 text-right tabular-nums">{d.paid_orders}</td>
                        <td className="px-4 py-3 text-right tabular-nums">
                          {formatIDR(d.paid_value_idr)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="mt-10">
            <h2 className="text-xl font-medium">Top products</h2>
            {a.top_products.length === 0 ? (
              <p className="mt-4 text-sm text-foreground-secondary">
                No paid product sales in this range.
              </p>
            ) : (
              <div className="mt-4 overflow-hidden rounded-[10px] border border-border">
                <table className="w-full text-left text-sm">
                  <thead className="bg-surface text-xs font-medium uppercase tracking-wide text-foreground-secondary">
                    <tr>
                      <th className="px-4 py-3">Product</th>
                      <th className="px-4 py-3 text-right">Qty</th>
                      <th className="px-4 py-3 text-right">Revenue</th>
                    </tr>
                  </thead>
                  <tbody>
                    {a.top_products.map((p) => (
                      <tr key={p.product_id} className="border-t border-border">
                        <td className="px-4 py-3">{p.product_name}</td>
                        <td className="px-4 py-3 text-right tabular-nums">{p.quantity}</td>
                        <td className="px-4 py-3 text-right tabular-nums">
                          {formatIDR(p.revenue_idr)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="rounded-[10px] border border-border p-5">
      <p className="text-xs font-medium uppercase text-foreground-muted">{label}</p>
      <p className="mt-2 text-[32px] font-medium tabular-nums leading-[38px] text-foreground">
        {value}
      </p>
    </div>
  );
}
