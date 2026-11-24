import Link from "next/link";
import { redirect } from "next/navigation";
import { ExternalLink, Share2, Sparkles } from "lucide-react";
import { getSession, getSetupChecklist, getStore, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardDescription, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/ui/status-badge";
import { urls } from "@/lib/urls";

const CHECKLIST_LABELS: { key: string; label: string; href?: (id: string) => string }[] = [
  { key: "create_store", label: "Create store" },
  { key: "first_product", label: "Add your first product", href: urls.storeCatalogNew },
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
  const orders = ordersData?.orders || [];
  const recent = orders.slice(0, 5);
  const salesTotal = orders.reduce((sum, o) => sum + (o.total_idr || 0), 0);
  const awaitingPayment = orders.filter((o) =>
    ["pending", "awaiting_payment", "unpaid"].includes(o.payment_status),
  ).length;
  const openOrders = orders.filter((o) =>
    !["completed", "cancelled", "canceled"].includes(o.order_status),
  ).length;

  const firstName =
    session.user.display_name?.split(/\s+/)[0] ||
    session.user.email.split("@")[0] ||
    "there";
  // Asia/Jakarta greeting without relying on server TZ
  const jakartaHour = Number(
    new Intl.DateTimeFormat("en-GB", {
      hour: "numeric",
      hour12: false,
      timeZone: "Asia/Jakarta",
    }).format(new Date()),
  );
  const greeting = greetingForHour(jakartaHour);
  const storefrontHref = store.storefront_url || urls.storefront(store.slug);

  const attention: { label: string; href: string; done?: boolean }[] = [];
  for (const item of CHECKLIST_LABELS) {
    if (checklist[item.key]) continue;
    if (!item.href) continue;
    attention.push({ label: item.label, href: item.href(storeId) });
  }
  if (awaitingPayment > 0) {
    attention.unshift({
      label: `${awaitingPayment} order${awaitingPayment === 1 ? "" : "s"} awaiting payment review`,
      href: urls.storeOrders(storeId),
    });
  }

  return (
    <div className="mx-auto max-w-overview space-y-6">
      {/* A) Store + business summary — one card, not four KPI cards */}
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
            <Button asChild variant="secondary" size="sm">
              <a href={storefrontHref} target="_blank" rel="noreferrer">
                <ExternalLink className="h-4 w-4" strokeWidth={1.75} />
                View store
              </a>
            </Button>
            <Button asChild variant="primary" size="sm">
              <Link href={urls.storeStorefront(storeId)}>
                <Share2 className="h-4 w-4" strokeWidth={1.75} />
                Share storefront
              </Link>
            </Button>
          </div>
        </div>
        <div className="border-t border-border px-6 py-5">
          <div className="flex flex-wrap items-end gap-8">
            <div>
              <p className="text-label text-foreground-muted">Sales (all orders)</p>
              <p className="mt-1 text-metric tabular-nums text-foreground">
                {formatIDR(salesTotal)}
              </p>
            </div>
            <div>
              <p className="text-label text-foreground-muted">Open orders</p>
              <p className="mt-1 text-metric-secondary tabular-nums text-foreground">{openOrders}</p>
            </div>
            <div>
              <p className="text-label text-foreground-muted">Awaiting payment</p>
              <p className="mt-1 text-metric-secondary tabular-nums text-foreground">
                {awaitingPayment}
              </p>
            </div>
          </div>
          <p className="mt-3 text-caption text-foreground-muted">
            Metrics reflect live order records — never fabricated totals.
          </p>
        </div>
      </Card>

      {/* B) Needs attention */}
      <section>
        <h2 className="text-section font-bold text-foreground">Needs attention</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Setup gaps and payment reviews that need a decision.
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

      {/* D) Connie compact entry — visual only, no AI backend */}
      <Card className="bg-warm-feature p-6 shadow-none">
        <div className="flex flex-wrap items-start gap-4">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-background shadow-xs">
            <Sparkles className="h-5 w-5 text-brand" strokeWidth={1.75} />
          </div>
          <div className="min-w-0 flex-1">
            <CardTitle>Ask Connie</CardTitle>
            <CardDescription>
              Connie helps with catalog setup, order questions, and next steps — as a teammate, not a
              chatbot. AI replies are not connected yet; this is a visual entry point.
            </CardDescription>
            <Button className="mt-4" variant="secondary" size="sm" disabled>
              Coming soon
            </Button>
          </div>
        </div>
      </Card>
    </div>
  );
}
