"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { AlertTriangle, Trash2 } from "lucide-react";
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
  uncertain: boolean;
  warning?: string | null;
  name?: string | null;
  description?: string | null;
  brand?: string | null;
  price_idr?: number | null;
  sku?: string | null;
  category_name?: string | null;
  image_url?: string;
  page_number?: number | null;
};

type Job = {
  id: string;
  status: string;
  progress_pct: number;
  item_count: number;
  error_message?: string | null;
};

export function ImportReview({ storeId, jobId }: { storeId: string; jobId: string }) {
  const router = useRouter();
  const [job, setJob] = useState<Job | null>(null);
  const [items, setItems] = useState<ImportItem[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

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
    if (!res.ok) return;
    const data = await res.json();
    setItems((prev) => prev.map((it) => (it.id === id ? { ...it, ...data.item } : it)));
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
      const selected = items.filter((i) => i.selected).map((i) => i.id);
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
      setMessage(`Saved ${data.created_count} draft product(s). Publish them from Catalog when ready.`);
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
      if (!res.ok) throw new Error("Retry failed");
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

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <Badge tone={job.status === "failed" ? "error" : "neutral"}>{job.status}</Badge>
        <span className="text-sm text-foreground-secondary">
          Progress {job.progress_pct}% · {job.item_count} items
        </span>
        {job.status === "failed" && (
          <Button type="button" variant="outline" size="sm" onClick={retry} disabled={busy}>
            Retry
          </Button>
        )}
      </div>
      {job.error_message && (
        <p className="text-sm text-error">{job.error_message}</p>
      )}
      {(job.status === "pending" || job.status === "processing") && (
        <p className="text-sm text-foreground-secondary">
          Processing PDF pages and extracting products. This page updates automatically.
        </p>
      )}
      {error && <p className="text-sm text-error">{error}</p>}
      {message && <p className="text-sm text-success">{message}</p>}

      {job.status === "review_required" && (
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="text-sm text-foreground-secondary">
            Review extracted items. Uncertain values are flagged — edit or deselect before confirming.
            Confirmed items become <strong>draft</strong> products only.
          </p>
          <Button type="button" onClick={confirmSelected} disabled={busy}>
            Confirm selected
          </Button>
        </div>
      )}

      <ul className="space-y-4">
        {items.map((it) => (
          <li key={it.id} className="rounded-lg border border-border p-4">
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
                      checked={it.selected}
                      disabled={job.status !== "review_required"}
                      onChange={(e) => patchItem(it.id, { selected: e.target.checked })}
                    />
                    Include
                  </label>
                  {it.uncertain && (
                    <span className="inline-flex items-center gap-1 text-xs text-warning">
                      <AlertTriangle className="h-3.5 w-3.5" strokeWidth={1.75} />
                      Uncertain
                    </span>
                  )}
                  {it.page_number != null && (
                    <span className="text-xs text-foreground-secondary">Page {it.page_number}</span>
                  )}
                  {job.status === "review_required" && (
                    <button
                      type="button"
                      className="ml-auto text-foreground-secondary hover:text-error"
                      aria-label="Delete item"
                      onClick={() => removeItem(it.id)}
                    >
                      <Trash2 className="h-4 w-4" strokeWidth={1.75} />
                    </button>
                  )}
                </div>
                {it.warning && <p className="text-xs text-warning">{it.warning}</p>}
                <div className="grid gap-3 sm:grid-cols-2">
                  <div>
                    <Label>Name</Label>
                    <Input
                      value={it.name || ""}
                      disabled={job.status !== "review_required"}
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
                      disabled={job.status !== "review_required"}
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
                      disabled={job.status !== "review_required"}
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
                      disabled={job.status !== "review_required"}
                      onBlur={(e) => patchItem(it.id, { sku: e.target.value || null })}
                      onChange={(e) =>
                        setItems((prev) =>
                          prev.map((x) => (x.id === it.id ? { ...x, sku: e.target.value } : x)),
                        )
                      }
                    />
                  </div>
                </div>
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
