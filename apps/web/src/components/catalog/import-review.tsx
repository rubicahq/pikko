"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { AlertTriangle, Check, Trash2, X } from "lucide-react";
import { parseIDRInput } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { IdrInput } from "@/components/ui/idr-input";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { urls } from "@/lib/urls";

type ImportItem = {
  id: string;
  selected: boolean;
  review_status?: string;
  uncertain: boolean;
  confidence_pct?: number | null;
  warning?: string | null;
  name?: string | null;
  description?: string | null;
  brand?: string | null;
  price_idr?: number | null;
  sku?: string | null;
  category_name?: string | null;
  image_url?: string;
  page_number?: number | null;
  created_product_id?: string | null;
};

type Job = {
  id: string;
  status: string;
  progress_pct: number;
  item_count: number;
  warning_count?: number;
  error_message?: string | null;
  ai_provider?: string | null;
  source_type?: string;
};

type Filter = "all" | "selected" | "uncertain" | "rejected";

function confidenceTone(pct: number | null | undefined): "success" | "warning" | "error" | "neutral" {
  if (pct == null) return "neutral";
  if (pct >= 80) return "success";
  if (pct >= 55) return "warning";
  return "error";
}

function confidenceLabel(pct: number | null | undefined): string {
  if (pct == null) return "—";
  if (pct >= 80) return `High ${Math.round(pct)}%`;
  if (pct >= 55) return `Medium ${Math.round(pct)}%`;
  return `Low ${Math.round(pct)}%`;
}

export function ImportReview({ storeId, jobId }: { storeId: string; jobId: string }) {
  const router = useRouter();
  const [job, setJob] = useState<Job | null>(null);
  const [items, setItems] = useState<ImportItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [filter, setFilter] = useState<Filter>("all");

  const load = useCallback(async () => {
    const [jRes, iRes] = await Promise.all([
      fetch(urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}`), { credentials: "include" }),
      fetch(urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}/items`), {
        credentials: "include",
      }),
    ]);
    if (!jRes.ok) {
      setError("Unable to load import job.");
      return;
    }
    const jData = await jRes.json();
    setJob(jData.job);
    if (iRes.ok) {
      const iData = await iRes.json();
      setItems(iData.items || []);
    }
  }, [storeId, jobId]);

  useEffect(() => {
    load();
    const t = setInterval(() => {
      setJob((prev) => {
        if (prev && (prev.status === "pending" || prev.status === "processing")) {
          load();
        }
        return prev;
      });
    }, 1500);
    return () => clearInterval(t);
  }, [load]);

  const counts = useMemo(() => {
    const selected = items.filter((i) => i.selected && i.review_status !== "rejected").length;
    const uncertain = items.filter((i) => i.uncertain).length;
    const rejected = items.filter((i) => i.review_status === "rejected" || !i.selected).length;
    return { selected, uncertain, rejected, total: items.length };
  }, [items]);

  const visible = useMemo(() => {
    switch (filter) {
      case "selected":
        return items.filter((i) => i.selected && i.review_status !== "rejected");
      case "uncertain":
        return items.filter((i) => i.uncertain);
      case "rejected":
        return items.filter((i) => i.review_status === "rejected" || !i.selected);
      default:
        return items;
    }
  }, [items, filter]);

  async function patchItem(id: string, body: Record<string, unknown>) {
    const res = await fetch(
      urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}/items/${id}`),
      {
        method: "PATCH",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    );
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      setError(data?.error?.message || "Unable to update item.");
      return;
    }
    const data = await res.json();
    setItems((prev) => prev.map((it) => (it.id === id ? { ...it, ...data.item } : it)));
  }

  async function bulk(action: string, extra?: Record<string, unknown>) {
    setBusy(true);
    setError(null);
    try {
      const res = await fetch(
        urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}/items/bulk`),
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ action, ...extra }),
        },
      );
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error?.message || "Bulk update failed");
      setMessage(`Updated ${data.updated} line(s).`);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Bulk update failed");
    } finally {
      setBusy(false);
    }
  }

  async function removeItem(id: string) {
    const res = await fetch(
      urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}/items/${id}`),
      { method: "DELETE", credentials: "include" },
    );
    if (res.ok) setItems((prev) => prev.filter((it) => it.id !== id));
  }

  async function confirmSelected() {
    setBusy(true);
    setError(null);
    setMessage(null);
    try {
      const selected = items
        .filter((i) => i.selected && i.review_status !== "rejected")
        .map((i) => i.id);
      if (selected.length === 0) {
        throw new Error("Select at least one item with a product name.");
      }
      const res = await fetch(
        urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}/confirm`),
        {
          method: "POST",
          credentials: "include",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ item_ids: selected }),
        },
      );
      const data = await res.json();
      if (!res.ok) throw new Error(data?.error?.message || "Confirm failed");
      const hotspots =
        typeof data.hotspot_count === "number" && data.hotspot_count > 0
          ? ` · ${data.hotspot_count} flipbook hotspot(s)`
          : "";
      setMessage(
        `Saved ${data.created_count} draft product(s)${hotspots}. Publish them from Catalog when ready.`,
      );
      await load();
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Confirm failed");
    } finally {
      setBusy(false);
    }
  }

  async function retry() {
    setBusy(true);
    try {
      const res = await fetch(
        urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}/retry`),
        { method: "POST", credentials: "include" },
      );
      const data = await res.json().catch(() => null);
      if (!res.ok) throw new Error(data?.error?.message || "Retry failed");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Retry failed");
    } finally {
      setBusy(false);
    }
  }

  if (!job) {
    return <p className="text-sm text-foreground-secondary">Loading import…</p>;
  }

  const reviewing = job.status === "review_required";

  return (
    <div className="space-y-6">
      <div className="space-y-3 rounded-lg border border-border bg-background p-4">
        <div className="flex flex-wrap items-center gap-3">
          <Badge tone={job.status === "failed" ? "error" : job.status === "completed" ? "success" : "neutral"}>
            {job.status}
          </Badge>
          <span className="text-sm text-foreground-secondary">
            {job.item_count} items
            {job.warning_count ? ` · ${job.warning_count} warnings` : ""}
            {job.ai_provider ? ` · AI: ${job.ai_provider}` : ""}
          </span>
          {job.status === "failed" && (
            <Button type="button" variant="outline" size="sm" onClick={retry} disabled={busy}>
              Retry
            </Button>
          )}
          {job.status === "completed" && (
            <Link
              href={urls.storeCatalog(storeId)}
              className="text-sm text-brand-dark underline-offset-2 hover:underline"
            >
              Open catalog
            </Link>
          )}
        </div>
        <div className="h-2 overflow-hidden rounded-sm bg-surface">
          <div
            className="h-full rounded-sm bg-brand transition-all duration-500"
            style={{ width: `${Math.min(100, Math.max(0, job.progress_pct))}%` }}
          />
        </div>
        <p className="text-xs text-foreground-muted">Progress {job.progress_pct}%</p>
      </div>

      {job.error_message && (
        <p className="rounded-md border border-warning-border bg-warning-bg px-3 py-2 text-sm text-warning">
          {job.error_message}
        </p>
      )}
      {(job.status === "pending" || job.status === "processing") && (
        <p className="text-sm text-foreground-secondary">
          Processing pages and extracting products. This page updates automatically.
        </p>
      )}
      {error && <p className="text-sm text-error">{error}</p>}
      {message && <p className="text-sm text-success">{message}</p>}

      {reviewing && (
        <div className="space-y-4">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="max-w-2xl text-sm text-foreground-secondary">
              Review extracted items. Confidence and warnings flag uncertain lines — edit, reject, or
              confirm. Confirmed items become <strong>draft</strong> products only.
            </p>
            <Button type="button" onClick={confirmSelected} disabled={busy || counts.selected === 0}>
              Confirm {counts.selected} selected
            </Button>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button type="button" size="sm" variant="outline" disabled={busy} onClick={() => bulk("select_all")}>
              Select all
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => bulk("accept_high_confidence")}
            >
              Accept high confidence
            </Button>
            <Button
              type="button"
              size="sm"
              variant="outline"
              disabled={busy}
              onClick={() => bulk("reject_uncertain")}
            >
              Reject uncertain
            </Button>
            <Button type="button" size="sm" variant="ghost" disabled={busy} onClick={() => bulk("reject_all")}>
              Reject all
            </Button>
          </div>

          <div className="flex flex-wrap gap-2 text-sm">
            {(
              [
                { id: "all" as const, label: `All (${counts.total})` },
                { id: "selected" as const, label: `Selected (${counts.selected})` },
                { id: "uncertain" as const, label: `Uncertain (${counts.uncertain})` },
                { id: "rejected" as const, label: `Rejected (${counts.rejected})` },
              ] as const
            ).map((f) => (
              <button
                key={f.id}
                type="button"
                className={`rounded-md px-3 py-1.5 ${
                  filter === f.id
                    ? "bg-surface-active font-medium text-foreground"
                    : "text-foreground-secondary hover:bg-surface-hover"
                }`}
                onClick={() => setFilter(f.id)}
              >
                {f.label}
              </button>
            ))}
          </div>
        </div>
      )}

      <ul className="space-y-4">
        {visible.map((it) => {
          const rejected = it.review_status === "rejected" || !it.selected;
          return (
            <li
              key={it.id}
              className={`rounded-lg border p-4 ${
                rejected ? "border-border bg-surface opacity-80" : "border-border bg-background"
              }`}
            >
              <div className="flex flex-col gap-4 md:flex-row">
                <div className="h-28 w-28 shrink-0 overflow-hidden rounded-md border border-border bg-surface">
                  {it.image_url ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={it.image_url} alt="" className="h-full w-full object-cover" />
                  ) : null}
                </div>
                <div className="min-w-0 flex-1 space-y-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <label className="flex items-center gap-2 text-sm">
                      <input
                        type="checkbox"
                        checked={it.selected && !rejected}
                        disabled={!reviewing || !!it.created_product_id}
                        onChange={(e) =>
                          patchItem(it.id, {
                            selected: e.target.checked,
                            review_status: e.target.checked ? "pending" : "rejected",
                          })
                        }
                      />
                      Include
                    </label>
                    <Badge tone={confidenceTone(it.confidence_pct)}>
                      {confidenceLabel(it.confidence_pct)}
                    </Badge>
                    {it.uncertain && (
                      <span className="inline-flex items-center gap-1 text-xs text-warning">
                        <AlertTriangle className="h-3.5 w-3.5" strokeWidth={1.75} />
                        Uncertain
                      </span>
                    )}
                    {rejected && <Badge tone="neutral">Rejected</Badge>}
                    {it.created_product_id && <Badge tone="success">Draft created</Badge>}
                    {it.page_number != null && (
                      <span className="text-xs text-foreground-secondary">Page {it.page_number}</span>
                    )}
                    {reviewing && !it.created_product_id && (
                      <div className="ml-auto flex items-center gap-1">
                        {!rejected ? (
                          <button
                            type="button"
                            className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-foreground-secondary hover:bg-surface-hover hover:text-error"
                            onClick={() => patchItem(it.id, { review_status: "rejected", selected: false })}
                          >
                            <X className="h-3.5 w-3.5" strokeWidth={1.75} />
                            Reject
                          </button>
                        ) : (
                          <button
                            type="button"
                            className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-xs text-foreground-secondary hover:bg-surface-hover hover:text-success"
                            onClick={() => patchItem(it.id, { review_status: "pending", selected: true })}
                          >
                            <Check className="h-3.5 w-3.5" strokeWidth={1.75} />
                            Restore
                          </button>
                        )}
                        <button
                          type="button"
                          className="text-foreground-secondary hover:text-error"
                          aria-label="Delete item"
                          onClick={() => removeItem(it.id)}
                        >
                          <Trash2 className="h-4 w-4" strokeWidth={1.75} />
                        </button>
                      </div>
                    )}
                  </div>
                  {it.warning && <p className="text-xs text-warning">{it.warning}</p>}
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div>
                      <Label>Name</Label>
                      <Input
                        value={it.name || ""}
                        disabled={!reviewing || !!it.created_product_id}
                        onChange={(e) =>
                          setItems((prev) =>
                            prev.map((x) => (x.id === it.id ? { ...x, name: e.target.value } : x)),
                          )
                        }
                        onBlur={(e) => patchItem(it.id, { name: e.target.value || null })}
                      />
                    </div>
                    <div>
                      <Label>Price (IDR)</Label>
                      <IdrInput
                        value={it.price_idr}
                        disabled={!reviewing || !!it.created_product_id}
                        onValueChange={(n) =>
                          setItems((prev) =>
                            prev.map((x) => (x.id === it.id ? { ...x, price_idr: n } : x)),
                          )
                        }
                        onBlur={(e) =>
                          patchItem(it.id, {
                            price_idr: parseIDRInput(e.target.value),
                          })
                        }
                      />
                    </div>
                    <div>
                      <Label>Brand</Label>
                      <Input
                        value={it.brand || ""}
                        disabled={!reviewing || !!it.created_product_id}
                        onBlur={(e) => patchItem(it.id, { brand: e.target.value || null })}
                        onChange={(e) =>
                          setItems((prev) =>
                            prev.map((x) => (x.id === it.id ? { ...x, brand: e.target.value } : x)),
                          )
                        }
                      />
                    </div>
                    <div>
                      <Label>SKU</Label>
                      <Input
                        value={it.sku || ""}
                        disabled={!reviewing || !!it.created_product_id}
                        onBlur={(e) => patchItem(it.id, { sku: e.target.value || null })}
                        onChange={(e) =>
                          setItems((prev) =>
                            prev.map((x) => (x.id === it.id ? { ...x, sku: e.target.value } : x)),
                          )
                        }
                      />
                    </div>
                    <div>
                      <Label>Category</Label>
                      <Input
                        value={it.category_name || ""}
                        disabled={!reviewing || !!it.created_product_id}
                        onBlur={(e) => patchItem(it.id, { category_name: e.target.value || null })}
                        onChange={(e) =>
                          setItems((prev) =>
                            prev.map((x) =>
                              x.id === it.id ? { ...x, category_name: e.target.value } : x,
                            ),
                          )
                        }
                      />
                    </div>
                    <div className="sm:col-span-2">
                      <Label>Description</Label>
                      <Input
                        value={it.description || ""}
                        disabled={!reviewing || !!it.created_product_id}
                        onBlur={(e) => patchItem(it.id, { description: e.target.value || null })}
                        onChange={(e) =>
                          setItems((prev) =>
                            prev.map((x) =>
                              x.id === it.id ? { ...x, description: e.target.value } : x,
                            ),
                          )
                        }
                      />
                    </div>
                  </div>
                </div>
              </div>
            </li>
          );
        })}
      </ul>

      {reviewing && visible.length === 0 && (
        <p className="text-sm text-foreground-secondary">No items in this filter.</p>
      )}
    </div>
  );
}
