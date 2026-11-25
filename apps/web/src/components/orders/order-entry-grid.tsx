"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Suggest = {
  product_id: string;
  variant_id?: string | null;
  name: string;
  sku?: string | null;
  variant_label?: string | null;
  unit_price_idr: number;
};

type Line = {
  key: string;
  product_id: string;
  variant_id?: string;
  product_label: string;
  quantity: number;
  unit_price_idr: number;
  error?: string;
};

type DraftPayload = {
  customer_name: string;
  customer_email: string;
  customer_phone: string;
  fulfillment_method: string;
  internal_notes: string;
  campaign_id: string;
  preorder_batch_id: string;
  items: {
    product_id: string;
    variant_id?: string;
    quantity: number;
    product_label?: string;
    unit_price_idr?: number;
  }[];
};

type CampaignOption = {
  id: string;
  name: string;
  campaign_type: string;
  operational_status?: string;
  preorder_batch_id?: string | null;
};

type BatchOption = {
  id: string;
  name: string;
  operational_status: string;
};

function newKey() {
  return Math.random().toString(36).slice(2, 10);
}

export function OrderEntryGrid({ storeId }: { storeId: string }) {
  const router = useRouter();
  const [customerName, setCustomerName] = useState("");
  const [customerEmail, setCustomerEmail] = useState("");
  const [customerPhone, setCustomerPhone] = useState("");
  const [fulfillment, setFulfillment] = useState("pickup");
  const [notes, setNotes] = useState("");
  const [campaignId, setCampaignId] = useState("");
  const [batchId, setBatchId] = useState("");
  const [campaigns, setCampaigns] = useState<CampaignOption[]>([]);
  const [batches, setBatches] = useState<BatchOption[]>([]);
  const [lines, setLines] = useState<Line[]>([
    { key: newKey(), product_id: "", product_label: "", quantity: 1, unit_price_idr: 0 },
  ]);
  const [draftId, setDraftId] = useState<string | null>(null);
  const [suggestQ, setSuggestQ] = useState<{ key: string; q: string } | null>(null);
  const [suggestions, setSuggestions] = useState<Suggest[]>([]);
  const [activeCell, setActiveCell] = useState<{ row: number; col: number } | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [savedAt, setSavedAt] = useState<string | null>(null);
  const saveTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const inputRefs = useRef<Record<string, HTMLInputElement | null>>({});
  const syncingRelation = useRef(false);

  useEffect(() => {
    void Promise.all([
      apiFetch<{ campaigns: CampaignOption[] }>(`/api/v1/stores/${storeId}/campaigns`),
      apiFetch<{ batches: BatchOption[] }>(`/api/v1/stores/${storeId}/preorders`),
    ])
      .then(([campRes, batchRes]) => {
        setCampaigns(campRes.campaigns || []);
        setBatches(batchRes.batches || []);
      })
      .catch(() => {
        /* selectors stay empty; create still works without relations */
      });
  }, [storeId]);

  const payload = useCallback((): DraftPayload => {
    return {
      customer_name: customerName,
      customer_email: customerEmail,
      customer_phone: customerPhone,
      fulfillment_method: fulfillment,
      internal_notes: notes,
      campaign_id: campaignId,
      preorder_batch_id: batchId,
      items: lines
        .filter((l) => l.product_id)
        .map((l) => ({
          product_id: l.product_id,
          variant_id: l.variant_id,
          quantity: l.quantity,
          product_label: l.product_label,
          unit_price_idr: l.unit_price_idr,
        })),
    };
  }, [customerName, customerEmail, customerPhone, fulfillment, notes, campaignId, batchId, lines]);

  function onCampaignChange(next: string) {
    setCampaignId(next);
    if (syncingRelation.current) return;
    if (!next) return;
    const camp = campaigns.find((c) => c.id === next);
    if (camp?.preorder_batch_id) {
      syncingRelation.current = true;
      setBatchId(camp.preorder_batch_id);
      queueMicrotask(() => {
        syncingRelation.current = false;
      });
    }
  }

  function onBatchChange(next: string) {
    setBatchId(next);
    if (syncingRelation.current) return;
    if (!next) return;
    const linked = campaigns.find((c) => c.preorder_batch_id === next);
    if (linked && !campaignId) {
      syncingRelation.current = true;
      setCampaignId(linked.id);
      queueMicrotask(() => {
        syncingRelation.current = false;
      });
    }
  }

  const autosave = useCallback(async () => {
    try {
      const body = { title: customerName || "Order draft", payload: payload() };
      if (draftId) {
        await apiFetch(`/api/v1/stores/${storeId}/order-entry/drafts/${draftId}`, {
          method: "PUT",
          body,
        });
      } else {
        const res = await apiFetch<{ draft: { id: string } }>(
          `/api/v1/stores/${storeId}/order-entry/drafts`,
          { method: "POST", body },
        );
        setDraftId(res.draft.id);
      }
      setSavedAt(new Date().toLocaleTimeString("id-ID", { timeZone: "Asia/Jakarta" }));
    } catch {
      /* keep typing */
    }
  }, [draftId, payload, storeId, customerName]);

  useEffect(() => {
    if (saveTimer.current) clearTimeout(saveTimer.current);
    saveTimer.current = setTimeout(() => {
      void autosave();
    }, 800);
    return () => {
      if (saveTimer.current) clearTimeout(saveTimer.current);
    };
  }, [autosave]);

  useEffect(() => {
    if (!suggestQ || suggestQ.q.trim().length < 1) {
      setSuggestions([]);
      return;
    }
    const t = setTimeout(() => {
      void apiFetch<{ suggestions: Suggest[] }>(
        `/api/v1/stores/${storeId}/order-entry/catalog-suggest?q=${encodeURIComponent(suggestQ.q)}`,
      ).then((res) => setSuggestions(res.suggestions || []))
        .catch(() => setSuggestions([]));
    }, 200);
    return () => clearTimeout(t);
  }, [suggestQ, storeId]);

  function updateLine(key: string, patch: Partial<Line>) {
    setLines((prev) => prev.map((l) => (l.key === key ? { ...l, ...patch } : l)));
  }

  function addLine() {
    setLines((prev) => [
      ...prev,
      { key: newKey(), product_id: "", product_label: "", quantity: 1, unit_price_idr: 0 },
    ]);
  }

  function pickSuggest(lineKey: string, s: Suggest) {
    updateLine(lineKey, {
      product_id: s.product_id,
      variant_id: s.variant_id || undefined,
      product_label: s.variant_label ? `${s.name} · ${s.variant_label}` : s.name,
      unit_price_idr: s.unit_price_idr,
    });
    setSuggestQ(null);
    setSuggestions([]);
  }

  async function validateBulk() {
    setError(null);
    setBusy(true);
    try {
      const items = lines
        .filter((l) => l.product_id)
        .map((l) => ({
          product_id: l.product_id,
          variant_id: l.variant_id,
          quantity: l.quantity,
        }));
      const res = await apiFetch<{
        valid: boolean;
        errors: { index: number; message: string }[];
        lines?: { product_id: string; unit_price_idr: number; line_total_idr: number }[];
      }>(`/api/v1/stores/${storeId}/order-entry/validate`, {
        method: "POST",
        body: { items },
      });
      if (!res.valid) {
        setError(res.errors?.map((e) => e.message).join(" · ") || "Validation failed");
        return false;
      }
      if (res.lines) {
        setLines((prev) => {
          const priced = res.lines!;
          let pi = 0;
          return prev.map((l) => {
            if (!l.product_id) return l;
            const p = priced[pi++];
            return p ? { ...l, unit_price_idr: p.unit_price_idr, error: undefined } : l;
          });
        });
      }
      return true;
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Validation failed");
      return false;
    } finally {
      setBusy(false);
    }
  }

  async function submit() {
    setError(null);
    const ok = await validateBulk();
    if (!ok) return;
    setBusy(true);
    try {
      const items = lines
        .filter((l) => l.product_id)
        .map((l) => ({
          product_id: l.product_id,
          variant_id: l.variant_id,
          quantity: l.quantity,
        }));
      const res = await apiFetch<{ order: { id: string } }>(`/api/v1/stores/${storeId}/orders`, {
        method: "POST",
        body: {
          items,
          customer_name: customerName,
          customer_email: customerEmail,
          customer_phone: customerPhone || undefined,
          fulfillment_method: fulfillment,
          internal_notes: notes || undefined,
          campaign_id: campaignId || undefined,
          preorder_batch_id: batchId || undefined,
          idempotency_key: crypto.randomUUID(),
        },
      });
      if (draftId) {
        try {
          await apiFetch(`/api/v1/stores/${storeId}/order-entry/drafts/${draftId}`, {
            method: "DELETE",
          });
        } catch {
          /* ignore */
        }
      }
      router.push(`/stores/${storeId}/orders/${res.order.id}`);
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Create failed");
    } finally {
      setBusy(false);
    }
  }

  function onGridKeyDown(e: React.KeyboardEvent, row: number, col: number) {
    const cols = 2;
    if (e.key === "Enter" || e.key === "ArrowDown") {
      e.preventDefault();
      const nextRow = Math.min(row + 1, lines.length - 1);
      if (e.key === "Enter" && row === lines.length - 1) addLine();
      setActiveCell({ row: nextRow, col });
      inputRefs.current[`${nextRow}-${col}`]?.focus();
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      const nextRow = Math.max(row - 1, 0);
      setActiveCell({ row: nextRow, col });
      inputRefs.current[`${nextRow}-${col}`]?.focus();
    } else if (e.key === "ArrowRight" && col < cols - 1) {
      e.preventDefault();
      setActiveCell({ row, col: col + 1 });
      inputRefs.current[`${row}-${col + 1}`]?.focus();
    } else if (e.key === "ArrowLeft" && col > 0) {
      e.preventDefault();
      setActiveCell({ row, col: col - 1 });
      inputRefs.current[`${row}-${col - 1}`]?.focus();
    } else if (e.key === "Tab" && !e.shiftKey && row === lines.length - 1 && col === cols - 1) {
      addLine();
    }
  }

  const subtotal = lines.reduce(
    (sum, l) => sum + (l.product_id ? l.unit_price_idr * l.quantity : 0),
    0,
  );

  return (
    <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <div>
          <Label htmlFor="cust-name">Customer name</Label>
          <Input
            id="cust-name"
            value={customerName}
            onChange={(e) => setCustomerName(e.target.value)}
            className="mt-1.5"
          />
        </div>
        <div>
          <Label htmlFor="cust-email">Email</Label>
          <Input
            id="cust-email"
            type="email"
            value={customerEmail}
            onChange={(e) => setCustomerEmail(e.target.value)}
            className="mt-1.5"
          />
        </div>
        <div>
          <Label htmlFor="cust-phone">Phone</Label>
          <Input
            id="cust-phone"
            value={customerPhone}
            onChange={(e) => setCustomerPhone(e.target.value)}
            className="mt-1.5"
          />
        </div>
        <div>
          <Label htmlFor="fulfill">Fulfillment</Label>
          <select
            id="fulfill"
            className="mt-1.5 flex h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={fulfillment}
            onChange={(e) => setFulfillment(e.target.value)}
          >
            <option value="pickup">Pickup</option>
            <option value="shipping">Shipping</option>
            <option value="free_shipping">Free shipping</option>
          </select>
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        <div>
          <Label htmlFor="campaign">Campaign (optional)</Label>
          <select
            id="campaign"
            className="mt-1.5 flex h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={campaignId}
            onChange={(e) => onCampaignChange(e.target.value)}
          >
            <option value="">No campaign</option>
            {campaigns.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
                {c.campaign_type ? ` · ${c.campaign_type}` : ""}
                {c.operational_status ? ` · ${c.operational_status}` : ""}
              </option>
            ))}
          </select>
        </div>
        <div>
          <Label htmlFor="preorder">Preorder batch (optional)</Label>
          <select
            id="preorder"
            className="mt-1.5 flex h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={batchId}
            onChange={(e) => onBatchChange(e.target.value)}
          >
            <option value="">No preorder</option>
            {batches.map((b) => (
              <option key={b.id} value={b.id}>
                {b.name} · {b.operational_status}
              </option>
            ))}
          </select>
          <p className="mt-1.5 text-caption text-foreground-muted">
            Leave both empty for a normal manual order. Selecting a preorder campaign fills its batch.
          </p>
        </div>
        <div>
          <Label htmlFor="notes">Internal notes</Label>
          <Input
            id="notes"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            className="mt-1.5"
          />
        </div>
      </div>

      <div className="overflow-x-auto rounded-md border border-border">
        <table className="w-full min-w-[640px] border-collapse text-sm">
          <thead className="bg-background-subtle text-left text-foreground-secondary">
            <tr>
              <th className="px-3 py-2 font-medium">#</th>
              <th className="px-3 py-2 font-medium">Product / variant</th>
              <th className="w-28 px-3 py-2 font-medium">Qty</th>
              <th className="w-36 px-3 py-2 text-right font-medium">Unit</th>
              <th className="w-36 px-3 py-2 text-right font-medium">Line</th>
            </tr>
          </thead>
          <tbody>
            {lines.map((line, row) => (
              <tr key={line.key} className="border-t border-border">
                <td className="px-3 py-1.5 text-foreground-muted tabular-nums">{row + 1}</td>
                <td className="relative px-2 py-1">
                  <Input
                    ref={(el) => {
                      inputRefs.current[`${row}-0`] = el;
                    }}
                    value={line.product_label}
                    onFocus={() => setActiveCell({ row, col: 0 })}
                    onChange={(e) => {
                      updateLine(line.key, {
                        product_label: e.target.value,
                        product_id: "",
                        variant_id: undefined,
                      });
                      setSuggestQ({ key: line.key, q: e.target.value });
                    }}
                    onKeyDown={(e) => onGridKeyDown(e, row, 0)}
                    placeholder="Type name or SKU…"
                    className="h-9 border-transparent bg-transparent px-1 focus-visible:border-border"
                    aria-label={`Product row ${row + 1}`}
                  />
                  {suggestQ?.key === line.key && suggestions.length > 0 ? (
                    <ul className="absolute z-20 mt-1 max-h-48 w-full overflow-auto rounded-md border border-border bg-background shadow-sm">
                      {suggestions.map((s) => (
                        <li key={`${s.product_id}-${s.variant_id || ""}`}>
                          <button
                            type="button"
                            className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left hover:bg-background-subtle"
                            onClick={() => pickSuggest(line.key, s)}
                          >
                            <span>
                              {s.name}
                              {s.variant_label ? ` · ${s.variant_label}` : ""}
                              {s.sku ? (
                                <span className="text-foreground-muted"> · {s.sku}</span>
                              ) : null}
                            </span>
                            <span className="tabular-nums text-foreground-secondary">
                              {formatIDR(s.unit_price_idr)}
                            </span>
                          </button>
                        </li>
                      ))}
                    </ul>
                  ) : null}
                </td>
                <td className="px-2 py-1">
                  <Input
                    ref={(el) => {
                      inputRefs.current[`${row}-1`] = el;
                    }}
                    type="number"
                    min={1}
                    value={line.quantity}
                    onFocus={() => setActiveCell({ row, col: 1 })}
                    onChange={(e) =>
                      updateLine(line.key, { quantity: Math.max(1, Number(e.target.value) || 1) })
                    }
                    onKeyDown={(e) => onGridKeyDown(e, row, 1)}
                    className="h-9 border-transparent bg-transparent px-1 tabular-nums focus-visible:border-border"
                    aria-label={`Quantity row ${row + 1}`}
                  />
                </td>
                <td className="px-3 py-1.5 text-right tabular-nums text-foreground-secondary">
                  {line.product_id ? formatIDR(line.unit_price_idr) : "—"}
                </td>
                <td className="px-3 py-1.5 text-right tabular-nums">
                  {line.product_id ? formatIDR(line.unit_price_idr * line.quantity) : "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant="secondary" onClick={addLine}>
            Add row
          </Button>
          <Button type="button" variant="secondary" disabled={busy} onClick={() => void validateBulk()}>
            Validate
          </Button>
          {savedAt ? (
            <span className="self-center text-caption text-foreground-muted">
              Draft saved {savedAt} (Asia/Jakarta)
              {activeCell ? ` · cell ${activeCell.row + 1}:${activeCell.col + 1}` : ""}
            </span>
          ) : null}
        </div>
        <div className="flex items-center gap-4">
          <p className="text-sm tabular-nums">
            Subtotal <span className="font-medium">{formatIDR(subtotal)}</span>
          </p>
          <Button type="button" disabled={busy} onClick={() => void submit()}>
            Create order
          </Button>
        </div>
      </div>
      {error ? <p className="text-sm text-error">{error}</p> : null}
    </div>
  );
}
