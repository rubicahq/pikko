"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { Button } from "@/components/ui/button";
import { Reveal } from "@/components/marketing/reveal";
import { apiFetch } from "@/lib/api";
import { cn } from "@/lib/utils";

type Plan = {
  code: string;
  name: string;
  description: string;
  price_monthly: number;
  price_annual: number;
  recommended: boolean;
  features: Record<string, boolean>;
  limits: Record<string, number>;
};

/** Released differentiators only — do not advertise unreleased keys. */
const PLAN_BULLETS: Record<string, string[]> = {
  basic: [
    "Full sell & fulfill journey",
    "Storefront, themes, catalogs & flipbook",
    "PDF catalog flipbook",
    "Preorders, live shopping, campaigns",
    "Purchasing, packing by order, shipments",
    "Customer portal, payments & basic analytics",
  ],
  standard: [
    "Everything in Basic",
    "Remove Pikko branding",
    "Team collaboration",
    "Quick order entry",
    "Product bulk import",
    "Pack by customer · consolidated payments",
  ],
  premium: [
    "Everything in Standard",
    "Order bulk ops (CSV)",
    "Store credit (when enabled)",
    "Higher usage limits",
  ],
  ultimate: [
    "Everything in Premium",
    "Highest usage limits",
  ],
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

export function PricingPlans() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [cycle, setCycle] = useState<"monthly" | "annual">("monthly");
  const [testing, setTesting] = useState(true);

  useEffect(() => {
    apiFetch<{ plans: Plan[]; testing_mode?: boolean }>("/api/v1/billing/plans")
      .then((res) => {
        setPlans(res.plans || []);
        setTesting(!!res.testing_mode);
      })
      .catch(() => setPlans([]));
  }, []);

  return (
    <>
      <Reveal>
        <p className="text-sm font-medium text-brand">Pricing</p>
        <h1 className="mt-3 max-w-3xl font-display text-4xl font-medium tracking-[-0.03em] text-foreground md:text-5xl">
          Plans for every stage of selling.
        </h1>
        <p className="mt-5 max-w-reading text-lg leading-relaxed text-foreground-secondary">
          Four store-based plans in IDR. Annual billing includes two months free. No platform
          transaction fee. Basic already includes the full sell and fulfill journey.
          {testing
            ? " Testing mode is on — activate any plan at Rp0 from onboarding or Settings."
            : null}
        </p>
      </Reveal>

      <div className="mt-10 flex justify-center">
        <div className="inline-flex rounded-[10px] border border-border bg-background p-1">
          {(["monthly", "annual"] as const).map((c) => (
            <button
              key={c}
              type="button"
              onClick={() => setCycle(c)}
              className={cn(
                "rounded-[8px] px-4 py-2 text-sm font-medium transition-colors",
                cycle === c
                  ? "bg-foreground text-background"
                  : "text-foreground-secondary hover:text-foreground",
              )}
            >
              {c === "monthly" ? "Monthly" : "Annually"}
            </button>
          ))}
        </div>
      </div>

      <div className="mt-12 grid gap-6 lg:grid-cols-4">
        {plans.map((p, i) => {
          const price = cycle === "monthly" ? p.price_monthly : p.price_annual;
          const bullets = PLAN_BULLETS[p.code] || [p.description];
          return (
            <Reveal key={p.code} delayMs={i * 60}>
              <div
                className={cn(
                  "flex h-full flex-col rounded-2xl border bg-background p-6",
                  p.recommended ? "border-foreground shadow-sm" : "border-border",
                )}
              >
                <p className="text-sm font-medium text-foreground-muted">{p.name}</p>
                {p.recommended ? (
                  <p className="mt-1 text-xs font-medium text-brand">Recommended</p>
                ) : (
                  <p className="mt-1 text-xs text-transparent">.</p>
                )}
                <p className="mt-4 text-3xl font-semibold tracking-tight tabular-nums text-foreground">
                  {formatIdr(price)}
                </p>
                <p className="mt-1 text-xs text-foreground-muted">
                  {cycle === "monthly" ? "per month" : "per year"}
                </p>
                <p className="mt-4 text-sm leading-relaxed text-foreground-secondary">
                  {p.description}
                </p>
                <ul className="mt-6 flex-1 space-y-2 text-sm text-foreground">
                  {bullets.map((b) => (
                    <li key={b}>{b}</li>
                  ))}
                </ul>
                <p className="mt-4 text-xs text-foreground-muted">
                  {formatLimit(p.limits.orders_per_month ?? 0)} orders/mo ·{" "}
                  {formatLimit(p.limits.active_products ?? 0)} products ·{" "}
                  {formatLimit(p.limits.catalogs ?? 0)} catalogs ·{" "}
                  {formatLimit(p.limits.team_members ?? 0)} team ·{" "}
                  {formatLimit(p.limits.ai_pdf_pages_per_month ?? 0)} AI PDF pages/mo
                </p>
                <div className="mt-6">
                  <Button
                    asChild
                    size="large"
                    className="w-full"
                    variant={p.recommended ? "primary" : "outline"}
                  >
                    <Link href="/auth">
                      {testing ? "Get started — activate plan" : "Get started"}
                    </Link>
                  </Button>
                </div>
              </div>
            </Reveal>
          );
        })}
      </div>

      <p className="mt-12 text-center text-sm text-foreground-muted">
        Higher plans inherit lower-plan features. Only released product capabilities are listed.
      </p>
    </>
  );
}
