"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch, formatIDR } from "@/lib/api";
import { PortalShell } from "@/components/portal/portal-shell";
import { usePortalSession } from "@/components/portal/use-portal-session";

type Dash = {
  store: { name: string; slug: string; primary_color?: string };
  customer: { name: string; email: string };
  orders: {
    id: string;
    order_number: string;
    order_status: string;
    payment_status: string;
    fulfillment_status: string;
    total_idr: number;
    created_at: string;
  }[];
  orders_total: number;
  store_credit_enabled: boolean;
  store_credit?: { balance_idr: number } | null;
  addresses_count: number;
  refund_preference?: { preferred_method: string } | null;
};

export default function PortalDashboardPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const { session, loading: sessionLoading } = usePortalSession(storeSlug);
  const [dash, setDash] = useState<Dash | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch<{ dashboard: Dash }>(`/api/v1/portal/${storeSlug}/dashboard`);
      setDash(res.dashboard);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to load portal.");
    }
  }, [storeSlug]);

  useEffect(() => {
    if (session) load();
  }, [session, load]);

  if (sessionLoading || !session) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-workspace">
        <p className="text-sm text-foreground-secondary">Loading…</p>
      </div>
    );
  }

  return (
    <PortalShell
      storeSlug={storeSlug}
      storeName={session.store.name}
      customerEmail={session.customer.email}
      primaryColor={session.store.primary_color}
    >
      <h1 className="text-page font-bold tracking-tight text-foreground">
        Hi{session.customer.name ? `, ${session.customer.name.split(" ")[0]}` : ""}.
      </h1>
      <p className="mt-1.5 text-body text-foreground-secondary">
        Orders, payments, and refund preferences for {session.store.name}.
      </p>

      {dash ? (
        <div className="mt-6 flex flex-wrap gap-3 text-sm">
          <span className="rounded-md border border-border bg-background px-3 py-2 text-foreground-secondary">
            {dash.orders_total} order{dash.orders_total === 1 ? "" : "s"}
          </span>
          <Link
            href={`/portal/${storeSlug}/addresses`}
            className="rounded-md border border-border bg-background px-3 py-2 text-foreground-secondary hover:bg-surface-hover"
          >
            {dash.addresses_count} saved address{dash.addresses_count === 1 ? "" : "es"}
          </Link>
          {dash.store_credit_enabled ? (
            <Link
              href={`/portal/${storeSlug}/credit`}
              className="rounded-md border border-border bg-background px-3 py-2 text-foreground-secondary hover:bg-surface-hover"
            >
              Credit {formatIDR(dash.store_credit?.balance_idr || 0)}
            </Link>
          ) : null}
        </div>
      ) : null}

      {error ? <p className="mt-6 text-sm text-error">{error}</p> : null}

      <section className="mt-8">
        <h2 className="text-section font-bold text-foreground">Orders</h2>
        <ul className="mt-4 divide-y divide-border-subtle overflow-hidden rounded-xl border border-border bg-background">
          {(dash?.orders || []).length === 0 ? (
            <li className="px-4 py-8 text-sm text-foreground-secondary">No orders yet.</li>
          ) : (
            (dash?.orders || []).map((o) => (
              <li key={o.id}>
                <Link
                  href={`/portal/${storeSlug}/orders/${o.id}`}
                  className="flex items-center justify-between gap-3 px-4 py-3.5 transition-colors hover:bg-surface-hover"
                >
                  <div className="min-w-0">
                    <p className="truncate text-sm font-semibold text-foreground">{o.order_number}</p>
                    <p className="mt-0.5 text-label capitalize text-foreground-muted">
                      {o.payment_status.replaceAll("_", " ")} ·{" "}
                      {o.fulfillment_status.replaceAll("_", " ")}
                    </p>
                  </div>
                  <span className="shrink-0 text-sm font-semibold tabular-nums text-foreground">
                    {formatIDR(o.total_idr)}
                  </span>
                </Link>
              </li>
            ))
          )}
        </ul>
      </section>
    </PortalShell>
  );
}
