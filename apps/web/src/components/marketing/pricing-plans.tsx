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

const HIGHLIGHT_FEATURES = [
  { key: "storefront", label: "Storefront & checkout" },
  { key: "catalogs", label: "Catalogs & flipbook" },
  { key: "campaigns", label: "Campaigns & preorders" },
  { key: "ai_pdf_import", label: "AI PDF import" },
  { key: "team", label: "Team members" },
  { key: "analytics", label: "Analytics" },
  { key: "live_shopping", label: "Live shopping ops" },
  { key: "purchasing", label: "Purchasing & receiving" },
  { key: "whatsapp_templates", label: "WhatsApp templates" },
  { key: "product_attributes", label: "Product attributes" },
];

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
          transaction fee.
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
                <ul className="mt-6 flex-1 space-y-2 text-sm">
                  {HIGHLIGHT_FEATURES.map((f) => (
                    <li
                      key={f.key}
                      className={
                        p.features[f.key]
                          ? "text-foreground"
                          : "text-foreground-muted line-through"
                      }
                    >
                      {f.label}
                    </li>
                  ))}
                </ul>
                <p className="mt-4 text-xs text-foreground-muted">
                  Up to {formatLimit(p.limits.orders_per_month ?? -1)} orders/mo ·{" "}
                  {formatLimit(p.limits.active_products ?? -1)} products
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
        Basic includes the full sell and fulfill journey. Higher plans inherit lower-plan features.
        Unreleased features are not listed.
      </p>
    </>
  );
}
