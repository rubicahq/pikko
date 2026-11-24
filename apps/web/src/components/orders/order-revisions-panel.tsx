"use client";

import { useEffect, useState } from "react";
import { apiFetch, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";

type Item = {
  id: string;
  product_name: string;
  quantity: number;
  original_quantity: number;
  unit_price_idr: number;
  line_total_idr: number;
  availability_status: string;
  allocated_qty: number;
  packed_qty: number;
  sku?: string | null;
};

type Revision = {
  id: string;
  revision_number: number;
  change_type: string;
  summary: string;
  reason?: string | null;
  created_at: string;
};

export function OrderItemsAndRevisions({
  storeId,
  orderId,
  items,
  canWrite,
}: {
  storeId: string;
  orderId: string;
  items: Item[];
  canWrite: boolean;
}) {
  const [rows, setRows] = useState(items);
  const [revisions, setRevisions] = useState<Revision[]>([]);
  const [busy, setBusy] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [qtyDraft, setQtyDraft] = useState<Record<string, number>>({});

  useEffect(() => {
    setRows(items);
  }, [items]);

  useEffect(() => {
    void apiFetch<{ revisions: Revision[] }>(
      `/api/v1/stores/${storeId}/orders/${orderId}/revisions`,
    )
      .then((res) => setRevisions(res.revisions || []))
      .catch(() => setRevisions([]));
  }, [storeId, orderId]);

  async function adjust(itemId: string) {
    const quantity = qtyDraft[itemId];
    if (!quantity || quantity <= 0) return;
    setBusy(itemId);
    setError(null);
    try {
      const res = await apiFetch<{ order: { items: Item[]; total_idr: number } }>(
        `/api/v1/stores/${storeId}/orders/${orderId}/items/${itemId}`,
        {
          method: "PATCH",
          body: { quantity, reason: "Merchant quantity adjustment" },
        },
      );
      setRows(res.order.items || []);
      const revs = await apiFetch<{ revisions: Revision[] }>(
        `/api/v1/stores/${storeId}/orders/${orderId}/revisions`,
      );
      setRevisions(revs.revisions || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Adjust failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-6">
      <section className="rounded-[10px] border border-border p-5">
        <h2 className="text-base font-medium">Items</h2>
        {error ? <p className="mt-2 text-sm text-error">{error}</p> : null}
        <ul className="mt-3 space-y-3 text-sm">
          {rows.map((it) => (
            <li key={it.id} className="border-b border-border pb-3 last:border-0">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div>
                  <p className="font-medium">
                    {it.product_name} × {it.quantity}
                  </p>
                  <p className="text-caption text-foreground-muted">
                    {formatIDR(it.unit_price_idr)}
                    {it.sku ? ` · ${it.sku}` : ""}
                    {it.original_quantity !== it.quantity
                      ? ` · original ${it.original_quantity}`
                      : ""}
                    {` · ${it.availability_status.replaceAll("_", " ")}`}
                    {` · alloc ${it.allocated_qty} · packed ${it.packed_qty}`}
                  </p>
                </div>
                <span className="tabular-nums">{formatIDR(it.line_total_idr)}</span>
              </div>
              {canWrite ? (
                <div className="mt-2 flex flex-wrap items-center gap-2">
                  <Input
                    type="number"
                    min={1}
                    className="h-9 w-24"
                    defaultValue={it.quantity}
                    onChange={(e) =>
                      setQtyDraft((prev) => ({
                        ...prev,
                        [it.id]: Math.max(1, Number(e.target.value) || 1),
                      }))
                    }
                  />
                  <Button
                    type="button"
                    size="sm"
                    variant="secondary"
                    disabled={busy === it.id}
                    onClick={() => void adjust(it.id)}
                  >
                    Adjust qty
                  </Button>
                </div>
              ) : null}
            </li>
          ))}
        </ul>
      </section>

      <section className="rounded-[10px] border border-border p-5">
        <h2 className="text-base font-medium">Revision history</h2>
        {revisions.length === 0 ? (
          <p className="mt-3 text-sm text-foreground-secondary">No revisions yet.</p>
        ) : (
          <ol className="mt-3 space-y-2 text-sm">
            {revisions.map((r) => (
              <li key={r.id} className="flex gap-3">
                <span className="w-8 tabular-nums text-foreground-muted">#{r.revision_number}</span>
                <div>
                  <p>{r.summary}</p>
                  <p className="text-caption text-foreground-muted">
                    {r.change_type.replaceAll("_", " ")}
                    {r.reason ? ` · ${r.reason}` : ""}
                    {" · "}
                    {new Date(r.created_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })}{" "}
                    (Asia/Jakarta)
                  </p>
                </div>
              </li>
            ))}
          </ol>
        )}
      </section>
    </div>
  );
}
