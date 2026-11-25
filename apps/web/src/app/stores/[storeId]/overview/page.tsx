import Link from "next/link";
import { redirect } from "next/navigation";
import { ExternalLink, Share2, Sparkles } from "lucide-react";
import { getSession, getSetupChecklist, getStore, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/ui/status-badge";
import { ConniePanel } from "@/components/connie/connie-panel";
import { urls } from "@/lib/urls";

const CHECKLIST_LABELS: { key: string; label: string; href?: (id: string) => string }[] = [
  { key: "create_store", label: "Create store" },
  { key: "first_product", label: "Add your first product", href: urls.storeProductNew },
  {
    key: "customize_storefront",
    label: "Customize storefront",
    href: urls.storeStorefront,
  },
  { key: "payment_methods", label: "Configure payment methods", href: urls.storeSettings },
  { key: "fulfillment", label: "Configure fulfillment", href: urls.storeSettings },
  { key: "publish_and_share", label: "Publish and share", href: urls.storeStorefront },
];

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

type AnalyticsOverview = {
  verified_payment_value_idr: number;
  unpaid_order_value_idr: number;
  paid_orders: number;
  total_orders: number;
  fulfilled_order_items: number;
  fulfilled_order_items_all_time: number;
  fulfilled_order_item_value_idr: number;
  from: string;
  to: string;
  operational: {
    awaiting_payment_verification: number;
    pending_confirmation: number;
    ready_to_ship: number;
    ready_for_pickup: number;
    packing_queue_orders: number;
    open_refunds: number;
    live_sessions_open: number;
  };
};

function greetingForHour(hour: number): string {
  if (hour < 11) return "Good morning";
  if (hour < 17) return "Good afternoon";
  return "Good evening";
}

export default async function StoreOverviewPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());
  const checklist = await getSetupChecklist(storeId);
  const ordersData = await serverFetch<{ orders: OrderRow[]; total: number }>(
    `/api/v1/stores/${storeId}/orders`,
  );
  const analyticsData = await serverFetch<{ analytics: AnalyticsOverview }>(
    `/api/v1/stores/${storeId}/analytics?tz=Asia/Jakarta`,
  );
  const a = analyticsData?.analytics;
  const orders = ordersData?.orders || [];
  const recent = orders.slice(0, 5);

  const firstName =
    session.user.display_name?.split(/\s+/)[0] ||
    session.user.email.split("@")[0] ||
    "there";
  const jakartaHour = Number(
    new Intl.DateTimeFormat("en-GB", {
      hour: "numeric",
      hour12: false,
      timeZone: "Asia/Jakarta",
    }).format(new Date()),
  );
  const greeting = greetingForHour(jakartaHour);
  const storefrontHref = store.storefront_url || urls.storefront(store.slug);

  const attention: { label: string; href: string }[] = [];
  if (a) {
    if (a.operational.awaiting_payment_verification > 0) {
      attention.push({
        label: `${a.operational.awaiting_payment_verification} payment${
          a.operational.awaiting_payment_verification === 1 ? "" : "s"
        } awaiting verification`,
        href: urls.storeOrders(storeId),
      });
    }
    if (a.operational.pending_confirmation > 0) {
      attention.push({
        label: `${a.operational.pending_confirmation} order${
          a.operational.pending_confirmation === 1 ? "" : "s"
        } pending confirmation`,
        href: urls.storeOrders(storeId),
      });
    }
    if (a.operational.packing_queue_orders > 0) {
      attention.push({
        label: `${a.operational.packing_queue_orders} order${
          a.operational.packing_queue_orders === 1 ? "" : "s"
        } in packing queue`,
        href: urls.storePacking(storeId),
      });
    }
    if (a.operational.ready_to_ship > 0) {
      attention.push({
        label: `${a.operational.ready_to_ship} ready to ship`,
        href: urls.storeShipments(storeId),
      });
    }
    if (a.operational.open_refunds > 0) {
      attention.push({
        label: `${a.operational.open_refunds} open refund${
          a.operational.open_refunds === 1 ? "" : "s"
        }`,
        href: urls.storeFinance(storeId),
      });
    }
    if (a.operational.live_sessions_open > 0) {
      attention.push({
        label: `${a.operational.live_sessions_open} live session${
          a.operational.live_sessions_open === 1 ? "" : "s"
        } open`,
        href: urls.storeLive(storeId),
      });
    }
  }
  for (const item of CHECKLIST_LABELS) {
    if (checklist[item.key]) continue;
    if (!item.href) continue;
    attention.push({ label: item.label, href: item.href(storeId) });
  }

  const hasSalesSignal = Boolean(
    a && (a.paid_orders > 0 || a.fulfilled_order_items_all_time > 0 || a.total_orders > 0),
  );

  return (
    <div className="mx-auto max-w-overview space-y-6">
      {/* A) Store + business summary — one card, vertical hierarchy */}
      <Card className="overflow-hidden p-0">
        <div className="flex flex-wrap items-start justify-between gap-4 p-6">
          <div className="min-w-0">
            <p className="text-[24px] font-bold leading-8 tracking-tight text-foreground">
              {greeting}, {firstName}
            </p>
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <h1 className="text-section font-bold text-foreground">{store.name}</h1>
              <Badge tone={store.publication_status === "published" ? "success" : "neutral"}>
                {store.publication_status}
              </Badge>
            </div>
            <p className="mt-1 text-sm text-foreground-secondary">
              {store.slug}
              {store.role ? ` · ${store.role}` : ""}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {store.publication_status === "published" ? (
              <Button asChild variant="secondary" size="sm">
                <a href={storefrontHref} target="_blank" rel="noreferrer">
                  <ExternalLink className="h-4 w-4" strokeWidth={1.75} />
                  View store
                </a>
              </Button>
            ) : (
              <Button asChild variant="secondary" size="sm">
                <Link href={urls.storeStorefront(storeId)}>
                  <ExternalLink className="h-4 w-4" strokeWidth={1.75} />
                  Publish to view
                </Link>
              </Button>
            )}
            <Button asChild variant="primary" size="sm">
              <Link href={urls.storeStorefront(storeId)}>
                <Share2 className="h-4 w-4" strokeWidth={1.75} />
                Share storefront
              </Link>
            </Button>
          </div>
        </div>
        <div className="border-t border-border px-6 py-5">
          {!a ? (
            <p className="text-sm text-foreground-secondary">
              Metrics unavailable right now. Open Analytics when the connection is back.
            </p>
          ) : !hasSalesSignal ? (
            <div>
              <p className="text-label text-foreground-muted">Fulfilled order items</p>
              <p className="mt-1 text-metric tabular-nums text-foreground">0</p>
              <p className="mt-3 text-caption text-foreground-muted">
                No sales data yet — empty on purpose, never fabricated.
              </p>
            </div>
          ) : (
            <>
              <div className="flex flex-wrap items-end gap-8">
                <div>
                  <p className="text-label text-foreground-muted">
                    Fulfilled items ({a.from} → {a.to})
                  </p>
                  <p className="mt-1 text-metric tabular-nums text-foreground">
                    {a.fulfilled_order_items}
                  </p>
                </div>
                <div>
                  <p className="text-label text-foreground-muted">Verified payments</p>
                  <p className="mt-1 text-metric-secondary tabular-nums text-foreground">
                    {formatIDR(a.verified_payment_value_idr)}
                  </p>
                </div>
                <div>
                  <p className="text-label text-foreground-muted">All-time fulfilled</p>
                  <p className="mt-1 text-metric-secondary tabular-nums text-foreground">
                    {a.fulfilled_order_items_all_time}
                  </p>
                </div>
              </div>
              <p className="mt-3 text-caption text-foreground-muted">
                Unpaid order value {formatIDR(a.unpaid_order_value_idr)} is not revenue.{" "}
                <Link href={urls.storeAnalytics(storeId)} className="font-semibold text-brand">
                  Open analytics
                </Link>
              </p>
            </>
          )}
        </div>
      </Card>

      {/* B) Needs attention */}
      <section>
        <h2 className="text-section font-bold text-foreground">Needs attention</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Ops tasks and setup gaps from live store data.
        </p>
        <Card className="mt-4 p-0">
          {attention.length === 0 ? (
            <p className="px-6 py-8 text-sm text-foreground-secondary">
              You&apos;re caught up. Nothing needs attention right now.
            </p>
          ) : (
            <ul className="divide-y divide-border">
              {attention.map((item) => (
                <li key={item.label}>
                  <Link
                    href={item.href}
                    className="flex items-center justify-between gap-3 px-6 py-3.5 text-sm transition-colors hover:bg-surface-hover"
                  >
                    <span className="font-medium text-foreground">{item.label}</span>
                    <span className="shrink-0 text-caption font-semibold text-brand">Open</span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </section>

      {/* C) Recent orders */}
      <section>
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="text-section font-bold text-foreground">Recent orders</h2>
            <p className="mt-1 text-sm text-foreground-secondary">Latest activity from your store.</p>
          </div>
          <Button asChild variant="ghost" size="sm">
            <Link href={urls.storeOrders(storeId)}>View all</Link>
          </Button>
        </div>
        <Card className="mt-4 p-0">
          {recent.length === 0 ? (
            <div className="px-6 py-8">
              <p className="text-sm font-medium text-foreground">No orders yet</p>
              <p className="mt-1 text-sm text-foreground-secondary">
                Share your storefront or create an order manually.
              </p>
              <div className="mt-4">
                <Button asChild size="sm">
                  <Link href={urls.storeOrderNew(storeId)}>New order</Link>
                </Button>
              </div>
            </div>
          ) : (
            <ul className="divide-y divide-border">
              {recent.map((o) => (
                <li key={o.id}>
                  <Link
                    href={urls.storeOrder(storeId, o.id)}
                    className="flex flex-wrap items-center justify-between gap-3 px-6 py-3.5 transition-colors hover:bg-surface-hover"
                  >
                    <div className="min-w-0">
                      <p className="truncate text-sm font-semibold text-foreground">
                        {o.order_number}
                      </p>
                      <p className="truncate text-caption text-foreground-secondary">
                        {o.customer_name || "Customer"}
                      </p>
                    </div>
                    <div className="flex items-center gap-3">
                      <StatusBadge status={o.order_status} />
                      <span className="min-w-[7rem] text-right text-sm font-semibold tabular-nums text-foreground">
                        {formatIDR(o.total_idr)}
                      </span>
                    </div>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </section>

      {/* D) Connie compact entry — MCP store-ops tools */}
      <Card className="bg-warm-feature p-6 shadow-none">
        <div className="flex flex-wrap items-start gap-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-background shadow-xs">
            <Sparkles className="h-5 w-5 text-brand" strokeWidth={1.75} />
          </div>
          <div className="min-w-0 flex-1">
            <CardTitle>Ask Connie</CardTitle>
            <CardDescription>
              Connie helps with catalog setup, order questions, and next steps — as a teammate, not a
              chatbot. She can look up orders, products, catalogs, customers, and more via store tools.
            </CardDescription>
            <div className="mt-4">
              <ConniePanel storeId={storeId} />
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}
