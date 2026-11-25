"use client";

import { useEffect, useMemo, useState } from "react";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

type Plan = {
  code: string;
  name: string;
  rank: number;
  description: string;
  price_monthly: number;
  price_annual: number;
  currency: string;
  recommended: boolean;
  features: Record<string, boolean>;
  limits: Record<string, number>;
};

type UsageEntry = {
  limit_key: string;
  used: number;
  limit: number;
  period_ym: string;
};

type Snapshot = {
  subscription: {
    plan_code: string;
    plan_name?: string;
    billing_cycle: string;
    status: string;
    is_test: boolean;
    activation_mode: string;
  } | null;
  features: Record<string, boolean>;
  limits: Record<string, number>;
  usage: UsageEntry[];
  flags: {
    test_bypass_enabled: boolean;
    plan_change_enabled: boolean;
  };
  can_create: boolean;
  is_expired: boolean;
};

const FEATURE_LABELS: Record<string, string> = {
  storefront: "Storefront",
  products: "Products",
  product_attributes: "Variants & attributes",
  catalogs: "Catalogs & flipbook",
  ai_pdf_import: "AI PDF import",
  campaigns: "Campaigns",
  preorders: "Preorders",
  live_shopping: "Live shopping",
  orders: "Orders & checkout",
  payments: "Payments & finance",
  purchasing: "Purchasing & receiving",
  packing: "Packing by order",
  packing_by_customer: "Pack by customer",
  shipments: "Shipments",
  customer_portal: "Customer portal",
  whatsapp_templates: "WhatsApp templates",
  business_rules: "Business rules",
  analytics: "Analytics",
  remove_branding: "Remove Pikko branding",
  quick_order: "Quick order entry",
  bulk_import: "Product bulk import",
  consolidated_payments: "Consolidated payments",
  team: "Team collaboration",
  bulk_ops: "Order bulk ops",
  store_credit: "Store credit",
};

const LIMIT_LABELS: Record<string, string> = {
  orders_per_month: "Orders / month",
  active_products: "Active products",
  catalogs: "Catalogs",
  campaigns: "Campaigns",
  team_members: "Team members",
  storage_gb: "Storage (GB)",
  ai_pdf_pages_per_month: "AI PDF pages / month",
};

function formatIdr(n: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(n);
}

function formatLimit(n: number) {
  return n < 0 ? "Unlimited" : n.toLocaleString("id-ID");
}

export function SubscriptionBilling({ storeId }: { storeId: string }) {
  const [snap, setSnap] = useState<Snapshot | null>(null);
  const [plans, setPlans] = useState<Plan[]>([]);
  const [cycle, setCycle] = useState<"monthly" | "annual">("monthly");
  const [selected, setSelected] = useState<string>("");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [overLimit, setOverLimit] = useState<UsageEntry[] | null>(null);

  async function load() {
    const [s, p] = await Promise.all([
      apiFetch<Snapshot>(`/api/v1/stores/${storeId}/subscription`),
      apiFetch<{ plans: Plan[] }>("/api/v1/billing/plans"),
    ]);
    setSnap(s);
    setPlans(p.plans || []);
    setSelected(s.subscription?.plan_code || p.plans?.[2]?.code || "premium");
    setCycle((s.subscription?.billing_cycle as "monthly" | "annual") || "monthly");
  }

  useEffect(() => {
    load().catch((err) =>
      setError(err instanceof ApiError ? err.message : "Unable to load subscription."),
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storeId]);

  const currentCode = snap?.subscription?.plan_code;

  const highlight = useMemo(() => {
    return plans.find((p) => p.code === selected) || null;
  }, [plans, selected]);

  async function activateOrChange(force = false) {
    setLoading(true);
    setError(null);
    setNotice(null);
    setOverLimit(null);
    try {
      const path = currentCode
        ? `/api/v1/stores/${storeId}/subscription/change`
        : `/api/v1/stores/${storeId}/subscription/activate`;
      await apiFetch(path, {
        body: { plan_code: selected, billing_cycle: cycle, force },
      });
      setNotice(
        snap?.flags?.test_bypass_enabled
          ? "Plan activated in testing mode (Rp0). Not counted as revenue."
          : "Plan updated.",
      );
      await load();
    } catch (err) {
      if (err instanceof ApiError && err.code === "downgrade_over_limit") {
        setOverLimit([]);
        setError(
          "Current usage exceeds the target plan. Existing data is kept. Confirm to downgrade anyway — new creates stay blocked until usage drops.",
        );
      } else {
        setError(err instanceof ApiError ? err.message : "Unable to update plan.");
      }
    } finally {
      setLoading(false);
    }
  }

  if (!snap) {
    return (
      <div className="rounded-xl border border-border bg-background p-6 text-sm text-foreground-secondary">
        Loading subscription…
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-reading space-y-8">
      <div>
        <h1 className="text-[30px] font-medium text-foreground">Subscription & Billing</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Plans are per store. Testing mode activates any plan at Rp0 with a simulated payment.
        </p>
      </div>

      <section className="rounded-xl border border-border bg-background p-6">
        <p className="text-xs font-medium uppercase tracking-wide text-foreground-muted">
          Current plan
        </p>
        {snap.subscription ? (
          <div className="mt-3 space-y-1">
            <p className="text-xl font-medium text-foreground">
              {snap.subscription.plan_name || snap.subscription.plan_code}
            </p>
            <p className="text-sm text-foreground-secondary">
              {snap.subscription.billing_cycle} · {snap.subscription.status}
              {snap.subscription.is_test ? " · testing" : ""}
            </p>
          </div>
        ) : (
          <p className="mt-3 text-sm text-foreground-secondary">No active subscription.</p>
        )}
      </section>

      <section className="space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <h2 className="text-lg font-medium text-foreground">Choose a plan</h2>
          <div className="inline-flex rounded-[10px] border border-border bg-background p-1">
            {(["monthly", "annual"] as const).map((c) => (
              <button
                key={c}
                type="button"
                onClick={() => setCycle(c)}
                className={cn(
                  "rounded-[8px] px-3 py-1.5 text-sm font-medium transition-colors",
                  cycle === c
                    ? "bg-foreground text-background"
                    : "text-foreground-secondary hover:text-foreground",
                )}
              >
                {c === "monthly" ? "Monthly" : "Annually (2 mo free)"}
              </button>
            ))}
          </div>
        </div>

        <div className="grid gap-3">
          {plans.map((p) => {
            const price = cycle === "monthly" ? p.price_monthly : p.price_annual;
            const active = selected === p.code;
            return (
              <button
                key={p.code}
                type="button"
                onClick={() => setSelected(p.code)}
                className={cn(
                  "rounded-xl border p-5 text-left transition-colors",
                  active
                    ? "border-foreground bg-background-subtle"
                    : "border-border bg-background hover:bg-background-subtle",
                )}
              >
                <div className="flex items-start justify-between gap-4">
                  <div>
                    <p className="text-base font-semibold text-foreground">
                      {p.name}
                      {p.recommended ? (
                        <span className="ml-2 text-xs font-medium text-brand">Recommended</span>
                      ) : null}
                    </p>
                    <p className="mt-1 text-sm text-foreground-secondary">{p.description}</p>
                  </div>
                  <div className="text-right">
                    <p className="text-lg font-semibold tabular-nums text-foreground">
                      {formatIdr(price)}
                    </p>
                    <p className="text-xs text-foreground-muted">
                      {cycle === "monthly" ? "/ month" : "/ year"}
                    </p>
                  </div>
                </div>
              </button>
            );
          })}
        </div>

        {highlight ? (
          <div className="rounded-xl border border-border bg-background p-5">
            <p className="text-sm font-medium text-foreground">Included on {highlight.name}</p>
            <ul className="mt-3 grid gap-2 sm:grid-cols-2">
              {Object.entries(FEATURE_LABELS).map(([key, label]) => {
                const on = !!highlight.features[key];
                return (
                  <li
                    key={key}
                    className={cn(
                      "text-sm",
                      on ? "text-foreground" : "text-foreground-muted line-through",
                    )}
                  >
                    {label}
                  </li>
                );
              })}
            </ul>
          </div>
        ) : null}

        {error ? <p className="text-sm text-error">{error}</p> : null}
        {notice ? <p className="text-sm text-foreground-secondary">{notice}</p> : null}
        {overLimit !== null ? (
          <Button variant="outline" loading={loading} onClick={() => activateOrChange(true)}>
            Confirm downgrade anyway
          </Button>
        ) : null}

        <Button
          size="large"
          loading={loading}
          disabled={!selected}
          onClick={() => activateOrChange(false)}
        >
          {snap.flags?.test_bypass_enabled ? "Activate plan (testing, Rp0)" : "Update plan"}
        </Button>
      </section>

      <section className="space-y-3">
        <h2 className="text-lg font-medium text-foreground">Usage</h2>
        <div className="overflow-hidden rounded-xl border border-border bg-background">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-border text-xs text-foreground-muted">
              <tr>
                <th className="px-4 py-3 font-medium">Metric</th>
                <th className="px-4 py-3 font-medium">Used</th>
                <th className="px-4 py-3 font-medium">Limit</th>
              </tr>
            </thead>
            <tbody>
              {(snap.usage || []).map((u) => (
                <tr key={u.limit_key} className="border-b border-border last:border-0">
                  <td className="px-4 py-3">{LIMIT_LABELS[u.limit_key] || u.limit_key}</td>
                  <td className="px-4 py-3 tabular-nums">{u.used.toLocaleString("id-ID")}</td>
                  <td className="px-4 py-3 tabular-nums">{formatLimit(u.limit)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <p className="text-xs text-foreground-muted">
          Monthly counters reset on the Jakarta calendar month. Cancelled orders do not restore
          usage. Plan changes do not reset usage.
        </p>
      </section>
    </div>
  );
}
