"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Package, CalendarClock, Layers } from "lucide-react";
import type { OnboardingDraft } from "@/lib/api";
import { apiFetch, ApiError } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Wordmark } from "@/components/brand/wordmark";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";

const CATEGORIES = [
  { value: "personal_shopper_jastip", label: "Personal shopper / Jastip" },
  { value: "fashion_and_accessories", label: "Fashion and accessories" },
  { value: "beauty_and_skincare", label: "Beauty and skincare" },
  { value: "food_and_beverages", label: "Food and beverages" },
  { value: "home_and_living", label: "Home and living" },
  { value: "baby_and_kids", label: "Baby and kids" },
  { value: "electronics_and_gadgets", label: "Electronics and gadgets" },
  { value: "handmade_and_crafts", label: "Handmade and crafts" },
  { value: "collectibles_and_hobbies", label: "Collectibles and hobbies" },
  { value: "health_and_wellness", label: "Health and wellness" },
  { value: "merchandise", label: "Merchandise" },
  { value: "other", label: "Other" },
];

const MODELS = [
  {
    value: "ready_stock",
    title: "Ready Stock",
    body: "Available and ready to ship.",
    icon: Package,
  },
  {
    value: "preorder",
    title: "Preorder",
    body: "Collect orders before sourcing or preparing.",
    icon: CalendarClock,
  },
  {
    value: "both",
    title: "Both",
    body: "Sell ready stock and preorder side by side.",
    icon: Layers,
  },
] as const;

type Plan = {
  code: string;
  name: string;
  description: string;
  price_monthly: number;
  price_annual: number;
  recommended: boolean;
};

function formatIdr(n: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(n);
}

function initialStep(draft: OnboardingDraft) {
  if (draft.step >= 4 || (draft.plan_code && draft.billing_cycle)) return 4;
  if (draft.step >= 3 || draft.selling_model) return 3;
  if (draft.step >= 2) return 2;
  return 1;
}

export function OnboardingWizard({
  draft,
  hasExistingStores,
  storefrontBase,
}: {
  draft: OnboardingDraft;
  hasExistingStores: boolean;
  storefrontBase: string;
}) {
  const router = useRouter();
  const [step, setStep] = useState(initialStep(draft));
  const [name, setName] = useState(draft.store_name || "");
  const [slug, setSlug] = useState(draft.store_slug || "");
  const [category, setCategory] = useState(draft.category || "");
  const [model, setModel] = useState(draft.selling_model || "");
  const [planCode, setPlanCode] = useState(draft.plan_code || "premium");
  const [billingCycle, setBillingCycle] = useState<"monthly" | "annual">(
    (draft.billing_cycle as "monthly" | "annual") || "monthly",
  );
  const [plans, setPlans] = useState<Plan[]>([]);
  const [slugTouched, setSlugTouched] = useState(!!draft.store_slug);
  const slugTouchedRef = useRef(slugTouched);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    apiFetch<{ plans: Plan[] }>("/api/v1/billing/plans")
      .then((res) => setPlans(res.plans || []))
      .catch(() => setPlans([]));
  }, []);

  const title = useMemo(() => {
    if (step === 1) {
      return draft.is_first_store && !hasExistingStores
        ? "Create your first store."
        : "Create a new store.";
    }
    if (step === 2) return "How do you sell?";
    if (step === 3) return "Choose a plan.";
    return "Activate your plan.";
  }, [step, draft.is_first_store, hasExistingStores]);

  async function suggestFromName(nextName: string) {
    setName(nextName);
    if (slugTouchedRef.current) return;
    try {
      const res = await apiFetch<{ slug: string }>(
        `/api/v1/stores/suggest-slug?name=${encodeURIComponent(nextName)}`,
      );
      if (!slugTouchedRef.current) setSlug(res.slug);
    } catch {
      // ignore
    }
  }

  async function continueIdentity() {
    setLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/onboarding/identity", {
        body: { name, slug, category },
      });
      setStep(2);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to continue.");
    } finally {
      setLoading(false);
    }
  }

  async function continueSellingModel() {
    setLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/onboarding/selling-model", {
        body: { selling_model: model },
      });
      setStep(3);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to continue.");
    } finally {
      setLoading(false);
    }
  }

  async function continuePlan() {
    setLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/onboarding/plan", {
        body: { plan_code: planCode, billing_cycle: billingCycle },
      });
      setStep(4);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to save plan.");
    } finally {
      setLoading(false);
    }
  }

  async function activateStore() {
    setLoading(true);
    setError(null);
    try {
      await apiFetch("/api/v1/onboarding/plan", {
        body: { plan_code: planCode, billing_cycle: billingCycle },
      });
      const res = await apiFetch<{ store: { id: string } }>("/api/v1/onboarding/complete", {
        method: "POST",
        body: {},
      });
      router.replace(urls.storeOverview(res.store.id));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to activate store.");
      setLoading(false);
    }
  }

  const selectedPlan = plans.find((p) => p.code === planCode);
  const selectedPrice =
    billingCycle === "monthly"
      ? selectedPlan?.price_monthly
      : selectedPlan?.price_annual;

  return (
    <div className="min-h-screen bg-background">
      <header className="flex items-center justify-between border-b border-border px-4 py-4 md:px-8">
        <Wordmark href="/" />
        <div className="flex items-center gap-2 text-sm text-foreground-secondary">
          {[1, 2, 3, 4].map((n, i) => (
            <span key={n} className="flex items-center gap-2">
              {i > 0 ? <span className="text-foreground-muted">/</span> : null}
              <span className={cn(step === n && "font-medium text-foreground")}>{n}</span>
            </span>
          ))}
        </div>
      </header>

      <div className="mx-auto grid max-w-5xl gap-0 md:grid-cols-2">
        <section className="px-4 py-10 md:px-10 md:py-16">
          <h1 className="text-[28px] font-medium leading-[1.12] tracking-[-0.02em] text-foreground md:text-4xl">
            {title}
          </h1>

          {step === 1 ? (
            <div className="mt-8 space-y-5">
              <div className="space-y-2">
                <Label htmlFor="name">Store name</Label>
                <Input
                  id="name"
                  value={name}
                  onChange={(e) => suggestFromName(e.target.value)}
                  placeholder="Japan Picks"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="slug">Store URL</Label>
                <div className="flex items-center gap-2">
                  <span className="hidden text-sm text-foreground-muted sm:inline">
                    {storefrontBase}/
                  </span>
                  <Input
                    id="slug"
                    value={slug}
                    onChange={(e) => {
                      slugTouchedRef.current = true;
                      setSlugTouched(true);
                      setSlug(e.target.value.toLowerCase());
                    }}
                    placeholder="japanpicks"
                  />
                </div>
                <p className="text-xs text-foreground-muted">
                  {storefrontBase}/{slug || "your-store"}
                </p>
              </div>
              <div className="space-y-2">
                <Label htmlFor="category">Business category</Label>
                <select
                  id="category"
                  className="flex h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                >
                  <option value="">Select a category</option>
                  {CATEGORIES.map((c) => (
                    <option key={c.value} value={c.value}>
                      {c.label}
                    </option>
                  ))}
                </select>
              </div>
              {error ? <p className="text-sm text-error">{error}</p> : null}
              <Button
                className="w-full md:w-auto"
                size="large"
                loading={loading}
                disabled={!name || !slug || !category}
                onClick={continueIdentity}
              >
                Continue.
              </Button>
            </div>
          ) : null}

          {step === 2 ? (
            <div className="mt-8 space-y-4">
              {MODELS.map((m) => {
                const Icon = m.icon;
                const selected = model === m.value;
                return (
                  <button
                    key={m.value}
                    type="button"
                    onClick={() => setModel(m.value)}
                    className={cn(
                      "flex w-full items-start gap-4 rounded-lg border p-4 text-left transition-colors duration-hover",
                      selected
                        ? "border-brand bg-brand-soft"
                        : "border-border bg-background hover:bg-background-subtle",
                    )}
                  >
                    <span
                      className={cn(
                        "mt-0.5 flex h-10 w-10 items-center justify-center rounded-md border",
                        selected ? "border-brand-border bg-background" : "border-border bg-surface",
                      )}
                    >
                      <Icon className="h-5 w-5 text-foreground" strokeWidth={1.75} />
                    </span>
                    <span>
                      <span className="block text-base font-medium text-foreground">{m.title}</span>
                      <span className="mt-1 block text-sm text-foreground-secondary">{m.body}</span>
                    </span>
                  </button>
                );
              })}
              {error ? <p className="text-sm text-error">{error}</p> : null}
              <div className="flex flex-wrap gap-3 pt-2">
                <Button variant="outline" onClick={() => setStep(1)}>
                  Back
                </Button>
                <Button size="large" loading={loading} disabled={!model} onClick={continueSellingModel}>
                  Continue.
                </Button>
              </div>
            </div>
          ) : null}

          {step === 3 ? (
            <div className="mt-8 space-y-5">
              <div className="inline-flex rounded-[10px] border border-border bg-background p-1">
                {(["monthly", "annual"] as const).map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setBillingCycle(c)}
                    className={cn(
                      "rounded-[8px] px-3 py-1.5 text-sm font-medium",
                      billingCycle === c
                        ? "bg-foreground text-background"
                        : "text-foreground-secondary",
                    )}
                  >
                    {c === "monthly" ? "Monthly" : "Annually"}
                  </button>
                ))}
              </div>
              <div className="space-y-3">
                {plans.map((p) => {
                  const price = billingCycle === "monthly" ? p.price_monthly : p.price_annual;
                  const selected = planCode === p.code;
                  return (
                    <button
                      key={p.code}
                      type="button"
                      onClick={() => setPlanCode(p.code)}
                      className={cn(
                        "flex w-full items-start justify-between gap-4 rounded-lg border p-4 text-left",
                        selected
                          ? "border-foreground bg-background-subtle"
                          : "border-border hover:bg-background-subtle",
                      )}
                    >
                      <span>
                        <span className="block text-base font-medium text-foreground">
                          {p.name}
                          {p.recommended ? (
                            <span className="ml-2 text-xs text-brand">Recommended</span>
                          ) : null}
                        </span>
                        <span className="mt-1 block text-sm text-foreground-secondary">
                          {p.description}
                        </span>
                      </span>
                      <span className="shrink-0 text-right text-sm font-semibold tabular-nums">
                        {formatIdr(price)}
                      </span>
                    </button>
                  );
                })}
              </div>
              <p className="text-xs text-foreground-muted">
                Testing mode: activation is Rp0 with a simulated payment. You can change plans later
                in Settings → Subscription & Billing.
              </p>
              {error ? <p className="text-sm text-error">{error}</p> : null}
              <div className="flex flex-wrap gap-3 pt-2">
                <Button variant="outline" onClick={() => setStep(2)}>
                  Back
                </Button>
                <Button size="large" loading={loading} disabled={!planCode} onClick={continuePlan}>
                  Continue.
                </Button>
              </div>
            </div>
          ) : null}

          {step === 4 ? (
            <div className="mt-8 space-y-5">
              <div className="rounded-xl border border-border bg-background-subtle p-5">
                <p className="text-sm text-foreground-muted">You are activating</p>
                <p className="mt-2 text-2xl font-medium text-foreground">
                  {selectedPlan?.name || planCode}
                </p>
                <p className="mt-1 text-sm text-foreground-secondary">
                  {billingCycle} · listed {selectedPrice != null ? formatIdr(selectedPrice) : "—"} ·
                  charged Rp0 in testing
                </p>
              </div>
              {error ? <p className="text-sm text-error">{error}</p> : null}
              <div className="flex flex-wrap gap-3 pt-2">
                <Button variant="outline" onClick={() => setStep(3)}>
                  Back
                </Button>
                <Button size="large" loading={loading} onClick={activateStore}>
                  Activate plan.
                </Button>
              </div>
            </div>
          ) : null}
        </section>

        <aside className="hidden border-l border-border bg-background-subtle md:block">
          <div className="flex h-full items-center justify-center p-10">
            <div className="w-full max-w-sm rounded-xl border border-border bg-background p-6">
              <p className="text-xs font-medium uppercase tracking-wide text-foreground-muted">
                Storefront preview
              </p>
              <p className="mt-4 text-2xl font-medium tracking-tight text-foreground">
                {name || "Your store"}
              </p>
              <p className="mt-2 text-sm text-foreground-secondary">
                {storefrontBase}/{slug || "your-store"}
              </p>
              <div className="mt-6 h-40 rounded-lg border border-dashed border-border bg-surface" />
              <p className="mt-4 text-sm text-foreground-muted">
                {step === 1
                  ? "Choose a name customers will recognize."
                  : step === 2
                    ? "You can change your selling model later in settings."
                    : step === 3
                      ? "Pick the plan that matches how you sell today."
                      : "Activate to open your dashboard — testing is free."}
              </p>
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}
