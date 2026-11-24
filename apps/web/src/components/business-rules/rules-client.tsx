"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

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
    min_subtotal_idr?: number;
    min_quantity?: number;
  };
};

type Batch = { id: string; name: string };

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
  const [feeMode, setFeeMode] = useState("fixed");
  const [feeFixed, setFeeFixed] = useState("10000");
  const [feePercent, setFeePercent] = useState("5");
  const [minSubtotal, setMinSubtotal] = useState("100000");
  const [minQty, setMinQty] = useState("2");

  const [previewModel, setPreviewModel] = useState("preorder");
  const [previewBatch, setPreviewBatch] = useState("");
  const [previewSubtotal, setPreviewSubtotal] = useState("200000");
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
        return { fee_mode: "fixed", fee_fixed_idr: Number(feeFixed) };
      }
      return { fee_mode: "percent", fee_percent: Number(feePercent) };
    }
    if (ruleType === "minimum_order") {
      return {
        min_subtotal_idr: minSubtotal ? Number(minSubtotal) : undefined,
        min_quantity: minQty ? Number(minQty) : undefined,
      };
    }
    return { require_approval: true };
  }

  return (
    <div className="mt-8 space-y-10">
      {error ? <p className="text-sm text-error">{error}</p> : null}

      <section className="rounded-[10px] border border-border p-5">
        <h2 className="text-base font-medium">Add rule</h2>
        <form
          className="mt-4 grid gap-4 sm:grid-cols-2"
          onSubmit={(e) => {
            e.preventDefault();
            run(async () => {
              await apiFetch(`/api/v1/stores/${storeId}/business-rules`, {
                method: "POST",
                body: {
                  rule_type: ruleType,
                  name,
                  scope_type: scopeType,
                  scope_selling_model: scopeType === "selling_model" ? scopeModel : undefined,
                  scope_batch_id: scopeType === "batch" ? scopeBatch : undefined,
                  config: buildConfig(),
                },
              });
            });
          }}
        >
          <div className="space-y-2">
            <Label htmlFor="rule-type">Type</Label>
            <select
              id="rule-type"
              className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
              value={ruleType}
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
                  <option value="fixed">Fixed IDR</option>
                  <option value="percent">Percent of subtotal</option>
                </select>
              </div>
              {feeMode === "fixed" ? (
                <div className="space-y-2">
                  <Label>Fixed amount</Label>
                  <Input value={feeFixed} onChange={(e) => setFeeFixed(e.target.value)} />
                </div>
              ) : (
                <div className="space-y-2">
                  <Label>Percent</Label>
                  <Input value={feePercent} onChange={(e) => setFeePercent(e.target.value)} />
                </div>
              )}
            </>
          ) : null}
          {ruleType === "minimum_order" ? (
            <>
              <div className="space-y-2">
                <Label>Min subtotal (IDR)</Label>
                <Input value={minSubtotal} onChange={(e) => setMinSubtotal(e.target.value)} />
              </div>
              <div className="space-y-2">
                <Label>Min quantity</Label>
                <Input value={minQty} onChange={(e) => setMinQty(e.target.value)} />
              </div>
            </>
          ) : null}
          <div className="sm:col-span-2">
            <Button type="submit" disabled={busy}>
              Publish rule
            </Button>
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
                  <td className="px-4 py-3 tabular-nums">v{r.current_version}</td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      type="button"
                      variant="ghost"
                      disabled={busy}
                      onClick={() =>
                        run(async () => {
                          await apiFetch(`/api/v1/stores/${storeId}/business-rules/${r.id}`, {
                            method: "DELETE",
                          });
                        })
                      }
                    >
                      Delete
                    </Button>
                  </td>
                </tr>
              ))}
              {rules.length === 0 ? (
                <tr>
                  <td colSpan={5} className="px-4 py-8 text-center text-foreground-secondary">
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
                  subtotal_idr: Number(previewSubtotal),
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
            <Input value={previewSubtotal} onChange={(e) => setPreviewSubtotal(e.target.value)} />
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
