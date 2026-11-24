import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardTitle } from "@/components/ui/card";
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
  fulfilled_order_items: number;
  fulfilled_order_item_value_idr: number;
  fulfilled_order_items_all_time: number;
  daily: Array<{
    date: string;
    orders: number;
    paid_orders: number;
    paid_value_idr: number;
    order_value_idr: number;
    fulfilled_items: number;
    fulfilled_value_idr: number;
  }>;
  top_products: Array<{
    product_id: string;
    product_name: string;
    quantity: number;
    revenue_idr: number;
  }>;
  activation: {
    has_product: boolean;
    has_published_product: boolean;
    has_payment_method: boolean;
    has_fulfillment_config: boolean;
    is_published: boolean;
    has_order: boolean;
    has_paid_order: boolean;
    has_fulfilled_item: boolean;
    checklist_complete: number;
    checklist_total: number;
  };
  engagement: {
    orders_in_range: number;
    paid_orders_in_range: number;
    live_sessions_in_range: number;
    live_quick_adds_in_range: number;
    active_campaigns: number;
    products_sold_sku: number;
  };
  operational: {
    awaiting_payment_verification: number;
    pending_confirmation: number;
    ready_to_ship: number;
    ready_for_pickup: number;
    packing_queue_orders: number;
    open_refunds: number;
    unpaid_order_value_idr: number;
    live_sessions_open: number;
  };
  retention: {
    new_customers: number;
    returning_customers: number;
    repeat_order_rate: number;
    customers_with_2_plus_orders: number;
  };
  events: {
    total_in_range: number;
    by_name: Array<{ event_name: string; count: number }>;
  };
};

function pct(n: number): string {
  return `${Math.round(n * 1000) / 10}%`;
}

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
  const qs = `?${q}`;

  const data = await serverFetch<{ analytics: Analytics }>(
    `/api/v1/stores/${storeId}/analytics${qs}`,
  );
  const a = data?.analytics;

  return (
    <div className="mx-auto max-w-overview space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-[26px] font-bold leading-8 tracking-tight text-foreground">
            Analytics
          </h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            Real store metrics · Asia/Jakarta
            {a ? ` · ${a.from} → ${a.to}` : ""}
          </p>
        </div>
        <form className="flex flex-wrap items-end gap-3" method="get">
          <label className="space-y-1 text-sm">
            <span className="text-[13px] font-semibold text-foreground-secondary">From</span>
            <input
              type="date"
              name="from"
              defaultValue={sp.from || a?.from || ""}
              className="block h-[42px] rounded-[10px] border border-border bg-background px-3 text-sm"
            />
          </label>
          <label className="space-y-1 text-sm">
            <span className="text-[13px] font-semibold text-foreground-secondary">To</span>
            <input
              type="date"
              name="to"
              defaultValue={sp.to || a?.to || ""}
              className="block h-[42px] rounded-[10px] border border-border bg-background px-3 text-sm"
            />
          </label>
          <Button type="submit" variant="secondary" size="sm">
            Apply
          </Button>
        </form>
      </div>

      {!a ? (
        <Card className="p-6">
          <p className="text-sm text-foreground-secondary">
            Unable to load analytics. Try again or check your connection.
          </p>
        </Card>
      ) : (
        <>
          {/* A) North star — one summary card, not four KPI cards */}
          <Card className="overflow-hidden p-0">
            <div className="p-6">
              <p className="text-[12px] font-semibold uppercase tracking-wide text-foreground-muted">
                North star
              </p>
              <CardTitle className="mt-1">Successfully fulfilled order items</CardTitle>
              <CardDescription>
                Paid items that reached shipped, delivered, or collected — never fabricated.
              </CardDescription>
            </div>
            <div className="border-t border-border px-6 py-5">
              <div className="flex flex-wrap items-end gap-8">
                <div>
                  <p className="text-[12px] font-medium text-foreground-muted">In range</p>
                  <p className="mt-1 text-[32px] font-bold tabular-nums leading-[38px] text-foreground">
                    {a.fulfilled_order_items}
                  </p>
                </div>
                <div>
                  <p className="text-[12px] font-medium text-foreground-muted">Item value (IDR)</p>
                  <p className="mt-1 text-[24px] font-semibold tabular-nums text-foreground">
                    {a.fulfilled_order_items > 0
                      ? formatIDR(a.fulfilled_order_item_value_idr)
                      : "—"}
                  </p>
                </div>
                <div>
                  <p className="text-[12px] font-medium text-foreground-muted">All-time</p>
                  <p className="mt-1 text-[24px] font-semibold tabular-nums text-foreground">
                    {a.fulfilled_order_items_all_time}
                  </p>
                </div>
              </div>
              {a.fulfilled_order_items === 0 && a.fulfilled_order_items_all_time === 0 ? (
                <p className="mt-4 text-sm text-foreground-secondary">
                  No fulfilled items yet. Ship or collect paid orders to move this number.
                </p>
              ) : null}
            </div>
          </Card>

          {/* B) Sales support (inline, not card grid) */}
          <section>
            <h2 className="text-lg font-bold text-foreground">Sales in range</h2>
            <p className="mt-1 text-sm text-foreground-secondary">
              Verified payments only count as received value — unpaid is shown separately.
            </p>
            <Card className="mt-4 p-6">
              {a.total_orders === 0 ? (
                <p className="text-sm text-foreground-secondary">No orders in this range.</p>
              ) : (
                <div className="flex flex-wrap items-end gap-8">
                  <div>
                    <p className="text-[12px] font-medium text-foreground-muted">
                      Verified payments
                    </p>
                    <p className="mt-1 text-[24px] font-semibold tabular-nums text-foreground">
                      {formatIDR(a.verified_payment_value_idr)}
                    </p>
                  </div>
                  <div>
                    <p className="text-[12px] font-medium text-foreground-muted">Paid orders</p>
                    <p className="mt-1 text-[24px] font-semibold tabular-nums text-foreground">
                      {a.paid_orders}
                    </p>
                  </div>
                  <div>
                    <p className="text-[12px] font-medium text-foreground-muted">Conversion</p>
                    <p className="mt-1 text-[24px] font-semibold tabular-nums text-foreground">
                      {pct(a.conversion_rate)}
                    </p>
                  </div>
                  <div>
                    <p className="text-[12px] font-medium text-foreground-muted">
                      Unpaid order value
                    </p>
                    <p className="mt-1 text-[24px] font-semibold tabular-nums text-foreground">
                      {formatIDR(a.unpaid_order_value_idr)}
                    </p>
                  </div>
                </div>
              )}
            </Card>
          </section>

          {/* C) Operational widgets */}
          <section>
            <h2 className="text-lg font-bold text-foreground">Operations snapshot</h2>
            <p className="mt-1 text-sm text-foreground-secondary">
              Live counts from packing, payments, finance, and live sessions.
            </p>
            <Card className="mt-4 p-0">
              <ul className="divide-y divide-border">
                <OpsRow
                  label="Awaiting payment verification"
                  value={a.operational.awaiting_payment_verification}
                  href={urls.storeOrders(storeId)}
                />
                <OpsRow
                  label="Packing queue"
                  value={a.operational.packing_queue_orders}
                  href={urls.storePacking(storeId)}
                />
                <OpsRow
                  label="Ready to ship"
                  value={a.operational.ready_to_ship}
                  href={urls.storeShipments(storeId)}
                />
                <OpsRow
                  label="Ready for pickup"
                  value={a.operational.ready_for_pickup}
                  href={urls.storeOrders(storeId)}
                />
                <OpsRow
                  label="Open refunds"
                  value={a.operational.open_refunds}
                  href={urls.storeFinance(storeId)}
                />
                <OpsRow
                  label="Live sessions open now"
                  value={a.operational.live_sessions_open}
                  href={urls.storeLive(storeId)}
                />
              </ul>
            </Card>
          </section>

          {/* D) Activation + engagement + retention */}
          <section className="grid gap-6 lg:grid-cols-2">
            <div>
              <h2 className="text-lg font-bold text-foreground">Activation</h2>
              <p className="mt-1 text-sm text-foreground-secondary">
                Setup progress for this store ({a.activation.checklist_complete}/
                {a.activation.checklist_total}).
              </p>
              <Card className="mt-4 p-0">
                <ul className="divide-y divide-border text-sm">
                  <CheckRow ok={a.activation.has_product} label="Has product" />
                  <CheckRow ok={a.activation.has_published_product} label="Published product" />
                  <CheckRow ok={a.activation.has_payment_method} label="Payment method" />
                  <CheckRow ok={a.activation.has_fulfillment_config} label="Fulfillment config" />
                  <CheckRow ok={a.activation.is_published} label="Storefront published" />
                  <CheckRow ok={a.activation.has_order} label="First order" />
                  <CheckRow ok={a.activation.has_fulfilled_item} label="First fulfilled item" />
                </ul>
              </Card>
            </div>
            <div>
              <h2 className="text-lg font-bold text-foreground">Engagement & retention</h2>
              <p className="mt-1 text-sm text-foreground-secondary">
                Activity and returning buyers in the selected range.
              </p>
              <Card className="mt-4 space-y-4 p-6 text-sm">
                <MetricLine
                  label="Live quick-adds"
                  value={String(a.engagement.live_quick_adds_in_range)}
                />
                <MetricLine
                  label="Live sessions created"
                  value={String(a.engagement.live_sessions_in_range)}
                />
                <MetricLine
                  label="SKUs sold (paid)"
                  value={String(a.engagement.products_sold_sku)}
                />
                <MetricLine label="New customers" value={String(a.retention.new_customers)} />
                <MetricLine
                  label="Returning customers"
                  value={String(a.retention.returning_customers)}
                />
                <MetricLine
                  label="Repeat rate"
                  value={
                    a.retention.new_customers + a.retention.returning_customers > 0
                      ? pct(a.retention.repeat_order_rate)
                      : "—"
                  }
                />
                <MetricLine
                  label="Customers with 2+ orders (all-time)"
                  value={String(a.retention.customers_with_2_plus_orders)}
                />
              </Card>
            </div>
          </section>

          {/* E) Daily + top products */}
          <section>
            <h2 className="text-lg font-bold text-foreground">Daily activity</h2>
            {a.daily.length === 0 ? (
              <p className="mt-4 text-sm text-foreground-secondary">No activity in this range.</p>
            ) : (
              <div className="mt-4 overflow-hidden rounded-[14px] border border-border bg-background">
                <table className="w-full text-left text-[13px]">
                  <thead className="bg-surface text-[12px] font-semibold text-foreground-secondary">
                    <tr>
                      <th className="h-11 px-4">Date</th>
                      <th className="h-11 px-4 text-right">Orders</th>
                      <th className="h-11 px-4 text-right">Paid</th>
                      <th className="h-11 px-4 text-right">Fulfilled items</th>
                      <th className="h-11 px-4 text-right">Paid value</th>
                    </tr>
                  </thead>
                  <tbody>
                    {a.daily.map((d) => (
                      <tr key={d.date} className="border-t border-border">
                        <td className="h-[52px] px-4">{d.date}</td>
                        <td className="px-4 text-right tabular-nums">{d.orders}</td>
                        <td className="px-4 text-right tabular-nums">{d.paid_orders}</td>
                        <td className="px-4 text-right tabular-nums">{d.fulfilled_items}</td>
                        <td className="px-4 text-right tabular-nums">
                          {formatIDR(d.paid_value_idr)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section>
            <h2 className="text-lg font-bold text-foreground">Top products</h2>
            {a.top_products.length === 0 ? (
              <p className="mt-4 text-sm text-foreground-secondary">
                No paid product sales in this range.
              </p>
            ) : (
              <div className="mt-4 overflow-hidden rounded-[14px] border border-border bg-background">
                <table className="w-full text-left text-[13px]">
                  <thead className="bg-surface text-[12px] font-semibold text-foreground-secondary">
                    <tr>
                      <th className="h-11 px-4">Product</th>
                      <th className="h-11 px-4 text-right">Qty</th>
                      <th className="h-11 px-4 text-right">Revenue</th>
                    </tr>
                  </thead>
                  <tbody>
                    {a.top_products.map((p) => (
                      <tr key={p.product_id} className="border-t border-border">
                        <td className="h-[52px] px-4">{p.product_name}</td>
                        <td className="px-4 text-right tabular-nums">{p.quantity}</td>
                        <td className="px-4 text-right tabular-nums">
                          {formatIDR(p.revenue_idr)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          {/* F) Event sink */}
          <section>
            <h2 className="text-lg font-bold text-foreground">Recorded events</h2>
            <p className="mt-1 text-sm text-foreground-secondary">
              From the analytics_events sink (payments, fulfillment, live).
            </p>
            {a.events.total_in_range === 0 ? (
              <p className="mt-4 text-sm text-foreground-secondary">
                No events recorded in this range yet.
              </p>
            ) : (
              <Card className="mt-4 p-0">
                <ul className="divide-y divide-border">
                  {a.events.by_name.map((e) => (
                    <li
                      key={e.event_name}
                      className="flex items-center justify-between gap-3 px-6 py-3.5 text-sm"
                    >
                      <span className="font-medium text-foreground">{e.event_name}</span>
                      <span className="tabular-nums text-foreground-secondary">{e.count}</span>
                    </li>
                  ))}
                </ul>
              </Card>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function OpsRow({
  label,
  value,
  href,
}: {
  label: string;
  value: number;
  href: string;
}) {
  return (
    <li>
      <Link
        href={href}
        className="flex items-center justify-between gap-3 px-6 py-3.5 text-sm transition-colors hover:bg-surface-hover"
      >
        <span className="font-medium text-foreground">{label}</span>
        <span className="flex items-center gap-3">
          <span className="tabular-nums font-semibold text-foreground">{value}</span>
          <span className="text-[12px] font-semibold text-brand">Open</span>
        </span>
      </Link>
    </li>
  );
}

function CheckRow({ ok, label }: { ok: boolean; label: string }) {
  return (
    <li className="flex items-center justify-between gap-3 px-6 py-3">
      <span className="text-foreground">{label}</span>
      <Badge tone={ok ? "success" : "neutral"}>{ok ? "Done" : "Open"}</Badge>
    </li>
  );
}

function MetricLine({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-foreground-secondary">{label}</span>
      <span className="font-semibold tabular-nums text-foreground">{value}</span>
    </div>
  );
}
