"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch, formatIDR, formatIDRDigits, maskIDRInput, parseIDRInput } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type FeeTier = { min_subtotal_idr: number; fee_fixed_idr: number };

type Rule = {
  id: string;
  rule_type: string;
  name: string;
  scope_type: string;
  scope_selling_model?: string | null;
  scope_batch_id?: string | null;
  is_enabled: boolean;
  current_version: number;
  config: {
    fee_mode?: string;
    fee_fixed_idr?: number;
    fee_percent?: number;
    tiers?: FeeTier[];
    min_subtotal_idr?: number;
    min_quantity?: number;
  };
};

type Batch = { id: string; name: string };

type TierDraft = { id: string; threshold: string; fee: string };

function newTierDraft(threshold = "", fee = ""): TierDraft {
  return {
    id: `tier-${Math.random().toString(36).slice(2, 9)}`,
    threshold,
    fee,
  };
}

export function RulesClient({
  storeId,
  rules,
  batches,
}: {
  storeId: string;
  rules: Rule[];
  batches: Batch[];
}) {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [ruleType, setRuleType] = useState("handling_fee");
  const [name, setName] = useState("Store handling fee");
  const [scopeType, setScopeType] = useState("store");
  const [scopeModel, setScopeModel] = useState("preorder");
  const [scopeBatch, setScopeBatch] = useState("");
  const [feeMode, setFeeMode] = useState("conditional");
  const [feeFixed, setFeeFixed] = useState(formatIDRDigits(10000));
  const [feePercent, setFeePercent] = useState("5");
  const [tiers, setTiers] = useState<TierDraft[]>([
    newTierDraft(formatIDRDigits(200000), formatIDRDigits(10000)),
    newTierDraft(formatIDRDigits(100000), formatIDRDigits(5000)),
  ]);
  const [minSubtotal, setMinSubtotal] = useState(formatIDRDigits(100000));
  const [minQty, setMinQty] = useState("2");
  const [editingId, setEditingId] = useState<string | null>(null);

  const [previewModel, setPreviewModel] = useState("preorder");
  const [previewBatch, setPreviewBatch] = useState("");
  const [previewSubtotal, setPreviewSubtotal] = useState(formatIDRDigits(200000));
  const [previewQty, setPreviewQty] = useState("3");
  const [previewResult, setPreviewResult] = useState<{
    handling_fee_idr?: number;
    shipping_fee_idr?: number;
    total_idr?: number;
    requires_approval?: boolean;
    minimum_ok?: boolean;
    minimum_message?: string;
  } | null>(null);

  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await fn();
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Failed.");
    } finally {
      setBusy(false);
    }
  }

  function buildConfig() {
    if (ruleType === "handling_fee") {
      if (feeMode === "fixed") {
        return { fee_mode: "fixed", fee_fixed_idr: parseIDRInput(feeFixed) ?? 0 };
      }
      if (feeMode === "percent") {
        return { fee_mode: "percent", fee_percent: Number(feePercent) };
      }
      return {
        fee_mode: "conditional",
        tiers: tiers.map((t) => ({
          min_subtotal_idr: parseIDRInput(t.threshold) ?? 0,
          fee_fixed_idr: parseIDRInput(t.fee) ?? 0,
        })),
      };
    }
    if (ruleType === "minimum_order") {
      const minSub = parseIDRInput(minSubtotal);
      return {
        min_subtotal_idr: minSub ?? undefined,
        min_quantity: minQty ? Number(minQty) : undefined,
      };
    }
    return { require_approval: true };
  }

  function ruleConfigSummary(r: Rule): string {
    const c = r.config || {};
    if (r.rule_type === "handling_fee") {
      if (c.fee_mode === "conditional" && c.tiers?.length) {
        return c.tiers
          .map((t) => `>${formatIDR(t.min_subtotal_idr)} → ${formatIDR(t.fee_fixed_idr)}`)
          .join("; ");
      }
      if (c.fee_mode === "percent" && c.fee_percent != null) return `${c.fee_percent}%`;
      if (c.fee_fixed_idr != null) return formatIDR(c.fee_fixed_idr);
      return "—";
    }
    if (r.rule_type === "minimum_order") {
      const parts: string[] = [];
      if (c.min_subtotal_idr != null) parts.push(formatIDR(c.min_subtotal_idr));
      if (c.min_quantity != null) parts.push(`qty ≥ ${c.min_quantity}`);
      return parts.length ? parts.join(" · ") : "—";
    }
    return "Approval";
  }

  function moveTier(index: number, dir: -1 | 1) {
    setTiers((prev) => {
      const next = [...prev];
      const j = index + dir;
      if (j < 0 || j >= next.length) return prev;
      const tmp = next[index];
      next[index] = next[j];
      next[j] = tmp;
      return next;
    });
  }

  function loadRuleForEdit(r: Rule) {
    setEditingId(r.id);
    setRuleType(r.rule_type);
    setName(r.name);
    setScopeType(r.scope_type);
    setScopeModel(r.scope_selling_model || "preorder");
    setScopeBatch(r.scope_batch_id || "");
    const c = r.config || {};
    if (r.rule_type === "handling_fee") {
      const mode = c.fee_mode || "fixed";
      setFeeMode(mode);
      if (mode === "percent") {
        setFeePercent(String(c.fee_percent ?? 5));
      } else if (mode === "conditional") {
        const loaded = (c.tiers || []).map((t) =>
          newTierDraft(formatIDRDigits(t.min_subtotal_idr), formatIDRDigits(t.fee_fixed_idr)),
        );
        setTiers(loaded.length ? loaded : [newTierDraft()]);
      } else {
        setFeeFixed(formatIDRDigits(c.fee_fixed_idr ?? 0));
      }
    }
    if (r.rule_type === "minimum_order") {
      if (c.min_subtotal_idr != null) setMinSubtotal(formatIDRDigits(c.min_subtotal_idr));
      if (c.min_quantity != null) setMinQty(String(c.min_quantity));
    }
  }

  function resetForm() {
    setEditingId(null);
    setRuleType("handling_fee");
    setName("Store handling fee");
    setScopeType("store");
    setFeeMode("conditional");
    setTiers([
      newTierDraft(formatIDRDigits(200000), formatIDRDigits(10000)),
      newTierDraft(formatIDRDigits(100000), formatIDRDigits(5000)),
    ]);
  }

  return (
    <div className="mt-8 space-y-10">
      {error ? <p className="text-sm text-error">{error}</p> : null}

      <section className="rounded-[10px] border border-border p-5">
        <h2 className="text-base font-medium">{editingId ? "Edit rule" : "Add rule"}</h2>
        <form
          className="mt-4 grid gap-4 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              const body = {
                rule_type: ruleType,
                name,
                scope_type: scopeType,
                scope_selling_model: scopeType === "selling_model" ? scopeModel : undefined,
                scope_batch_id: scopeType === "batch" ? scopeBatch : undefined,
                config: buildConfig(),
              };
              if (editingId) {
                await apiFetch(`/api/v1/stores/${storeId}/business-rules/${editingId}`, {
                  method: "PATCH",
                  body,
                });
              } else {
                await apiFetch(`/api/v1/stores/${storeId}/business-rules`, {
                  method: "POST",
                  body,
                });
              }
              resetForm();
            });
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="rule-type">Type</Label>
            <select
              id="rule-type"
              className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
              value={ruleType}
              disabled={Boolean(editingId)}
              onChange={(e) => setRuleType(e.target.value)}
            >
              <option value="handling_fee">Handling fee</option>
              <option value="minimum_order">Minimum order</option>
              <option value="order_approval">Order approval</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label htmlFor="rule-name">Name</Label>
            <Input id="rule-name" value={name} onChange={(e) => setName(e.target.value)} required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="rule-scope">Scope</Label>
            <select
              id="rule-scope"
              className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
              value={scopeType}
              onChange={(e) => setScopeType(e.target.value)}
            >
              <option value="store">Entire store</option>
              <option value="selling_model">Selling model</option>
              <option value="batch">Specific batch</option>
            </select>
          </div>
          {scopeType === "selling_model" ? (
            <div className="space-y-2">
              <Label>Model</Label>
              <select
                className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
                value={scopeModel}
                onChange={(e) => setScopeModel(e.target.value)}
              >
                <option value="ready_stock">Ready stock</option>
                <option value="preorder">Preorder</option>
              </select>
            </div>
          ) : null}
          {scopeType === "batch" ? (
            <div className="space-y-2">
              <Label>Batch</Label>
              <select
                className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
                value={scopeBatch}
                onChange={(e) => setScopeBatch(e.target.value)}
                required
              >
                <option value="">Select batch</option>
                {batches.map((b) => (
                  <option key={b.id} value={b.id}>
                    {b.name}
                  </option>
                ))}
              </select>
            </div>
          ) : null}
          {ruleType === "handling_fee" ? (
            <>
              <div className="space-y-2">
                <Label>Fee mode</Label>
                <select
                  className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
                  value={feeMode}
                  onChange={(e) => setFeeMode(e.target.value)}
                >
                  <option value="conditional">Conditional (by subtotal)</option>
                  <option value="fixed">Fixed IDR</option>
                  <option value="percent">Percent of subtotal</option>
                </select>
              </div>
              {feeMode === "fixed" ? (
                <div className="space-y-2">
                  <Label>Fixed amount</Label>
                  <Input
                    inputMode="numeric"
                    autoComplete="off"
                    value={feeFixed}
                    onChange={(e) => setFeeFixed(maskIDRInput(e.target.value))}
                  />
                </div>
              ) : null}
              {feeMode === "percent" ? (
                <div className="space-y-2">
                  <Label>Percent</Label>
                  <Input value={feePercent} onChange={(e) => setFeePercent(e.target.value)} />
                </div>
              ) : null}
              {feeMode === "conditional" ? (
                <div className="space-y-3 sm:col-span-2">
                  <div>
                    <Label>Tiers (first match wins)</Label>
                    <p className="mt-1 text-caption text-foreground-secondary">
                      Top row is highest priority. Match when product subtotal is greater than the
                      threshold. Example: &gt;200.000 → Rp10.000, else &gt;100.000 → Rp5.000.
                    </p>
                  </div>
                  <ul className="space-y-2">
                    {tiers.map((tier, index) => (
                      <li
                        key={tier.id}
                        className="flex flex-wrap items-end gap-2 rounded-md border border-border bg-background-subtle/40 p-2.5"
                      >
                        <span className="w-6 pb-2 text-caption tabular-nums text-foreground-muted">
                          {index + 1}.
                        </span>
                        <div className="min-w-[8rem] flex-1 space-y-1">
                          <Label className="text-caption text-foreground-muted">
                            If subtotal &gt;
                          </Label>
                          <Input
                            inputMode="numeric"
                            autoComplete="off"
                            value={tier.threshold}
                            onChange={(e) =>
                              setTiers((prev) =>
                                prev.map((t) =>
                                  t.id === tier.id
                                    ? { ...t, threshold: maskIDRInput(e.target.value) }
                                    : t,
                                ),
                              )
                            }
                          />
                        </div>
                        <div className="min-w-[8rem] flex-1 space-y-1">
                          <Label className="text-caption text-foreground-muted">Fee (IDR)</Label>
                          <Input
                            inputMode="numeric"
                            autoComplete="off"
                            value={tier.fee}
                            onChange={(e) =>
                              setTiers((prev) =>
                                prev.map((t) =>
                                  t.id === tier.id
                                    ? { ...t, fee: maskIDRInput(e.target.value) }
                                    : t,
                                ),
                              )
                            }
                          />
                        </div>
                        <div className="flex gap-1 pb-0.5">
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={index === 0 || busy}
                            onClick={() => moveTier(index, -1)}
                          >
                            Up
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={index === tiers.length - 1 || busy}
                            onClick={() => moveTier(index, 1)}
                          >
                            Down
                          </Button>
                          <Button
                            type="button"
                            size="sm"
                            variant="ghost"
                            disabled={tiers.length <= 1 || busy}
                            onClick={() =>
                              setTiers((prev) => prev.filter((t) => t.id !== tier.id))
                            }
                          >
                            Remove
                          </Button>
                        </div>
                      </li>
                    ))}
                  </ul>
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    disabled={busy}
                    onClick={() => setTiers((prev) => [...prev, newTierDraft()])}
                  >
                    Add tier
                  </Button>
                </div>
              ) : null}
            </>
          ) : null}
          {ruleType === "minimum_order" ? (
            <>
              <div className="space-y-2">
                <Label>Min subtotal (IDR)</Label>
                <Input
                  inputMode="numeric"
                  autoComplete="off"
                  value={minSubtotal}
                  onChange={(e) => setMinSubtotal(maskIDRInput(e.target.value))}
                />
              </div>
              <div className="space-y-2">
                <Label>Min quantity</Label>
                <Input value={minQty} onChange={(e) => setMinQty(e.target.value)} />
              </div>
            </>
          ) : null}
          <div className="flex flex-wrap gap-2 sm:col-span-2">
            <Button type="submit" disabled={busy}>
              {editingId ? "Save changes" : "Publish rule"}
            </Button>
            {editingId ? (
              <Button type="button" variant="ghost" disabled={busy} onClick={resetForm}>
                Cancel edit
              </Button>
            ) : null}
          </div>
        </form>
      </section>

      <section>
        <h2 className="text-base font-medium">Published rules</h2>
        <div className="mt-4 overflow-hidden rounded-[10px] border border-border">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-border bg-background-subtle text-xs font-medium text-foreground-muted">
              <tr>
                <th className="px-4 py-3">Name</th>
                <th className="px-4 py-3">Type</th>
                <th className="px-4 py-3">Scope</th>
                <th className="px-4 py-3">Amount</th>
                <th className="px-4 py-3">Version</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {rules.map((r) => (
                <tr key={r.id} className="border-b border-border last:border-0">
                  <td className="px-4 py-3">{r.name}</td>
                  <td className="px-4 py-3 capitalize">{r.rule_type.replaceAll("_", " ")}</td>
                  <td className="px-4 py-3 capitalize">
                    {r.scope_type}
                    {r.scope_selling_model ? ` · ${r.scope_selling_model}` : ""}
                  </td>
                  <td className="px-4 py-3 tabular-nums">{ruleConfigSummary(r)}</td>
                  <td className="px-4 py-3 tabular-nums">v{r.current_version}</td>
                  <td className="px-4 py-3 text-right">
                    <div className="flex justify-end gap-1">
                      <Button
                        type="button"
                        variant="ghost"
                        disabled={busy}
                        onClick={() => loadRuleForEdit(r)}
                      >
                        Edit
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        disabled={busy}
                        onClick={() =>
                          run(async () => {
                            await apiFetch(`/api/v1/stores/${storeId}/business-rules/${r.id}`, {
                              method: "DELETE",
                            });
                            if (editingId === r.id) resetForm();
                          })
                        }
                      >
                        Delete
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
              {rules.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-foreground-secondary">
                    No rules yet. Defaults apply: no handling fee, no minimum, no approval.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      </section>

      <section className="rounded-[10px] border border-border p-5">
        <h2 className="text-base font-medium">Rule preview</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Uses the same backend evaluation as live checkout.
        </p>
        <form
          className="mt-4 grid gap-4 sm:grid-cols-4"
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              const res = await apiFetch<{
                preview: {
                  handling_fee_idr?: number;
                  shipping_fee_idr?: number;
                  total_idr?: number;
                  requires_approval?: boolean;
                  minimum_ok?: boolean;
                  minimum_message?: string;
                };
              }>(`/api/v1/stores/${storeId}/business-rules/preview`, {
                method: "POST",
                body: {
                  selling_model: previewModel,
                  batch_id: previewBatch || undefined,
                  subtotal_idr: parseIDRInput(previewSubtotal) ?? 0,
                  item_quantity: Number(previewQty),
                  fulfillment_method: "pickup",
                },
              });
              setPreviewResult(res.preview);
            });
          }}
        >
          <div className="space-y-2">
            <Label>Model</Label>
            <select
              className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
              value={previewModel}
              onChange={(e) => setPreviewModel(e.target.value)}
            >
              <option value="ready_stock">Ready stock</option>
              <option value="preorder">Preorder</option>
            </select>
          </div>
          <div className="space-y-2">
            <Label>Batch</Label>
            <select
              className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
              value={previewBatch}
              onChange={(e) => setPreviewBatch(e.target.value)}
            >
              <option value="">None</option>
              {batches.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </div>
          <div className="space-y-2">
            <Label>Subtotal</Label>
            <Input
              inputMode="numeric"
              autoComplete="off"
              value={previewSubtotal}
              onChange={(e) => setPreviewSubtotal(maskIDRInput(e.target.value))}
            />
          </div>
          <div className="space-y-2">
            <Label>Quantity</Label>
            <Input value={previewQty} onChange={(e) => setPreviewQty(e.target.value)} />
          </div>
          <div className="sm:col-span-4">
            <Button type="submit" variant="outline" disabled={busy}>
              Preview
            </Button>
          </div>
        </form>
        {previewResult ? (
          <div className="mt-4 rounded-md border border-border bg-surface p-4 text-sm">
            <p>Handling fee: {formatIDR(previewResult.handling_fee_idr || 0)}</p>
            <p>Shipping: {formatIDR(previewResult.shipping_fee_idr || 0)}</p>
            <p>Total: {formatIDR(previewResult.total_idr || 0)}</p>
            <p>Approval required: {previewResult.requires_approval ? "yes" : "no"}</p>
            <p>Minimum OK: {previewResult.minimum_ok ? "yes" : "no"}</p>
            {previewResult.minimum_message ? (
              <p className="text-error">{previewResult.minimum_message}</p>
            ) : null}
          </div>
        ) : null}
      </section>
    </div>
  );
}
