"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { BatchCoverEditor, BatchVariantPairing } from "@/components/preorders/batch-cover-variants";

type BatchProduct = {
  id: string;
  product_id: string;
  product_name: string;
  effective_price_idr: number;
  quantity_limit?: number | null;
  reserved_qty?: number;
  availability: string;
};

type Batch = {
  id: string;
  name: string;
  slug: string;
  operational_status: string;
  publication_status: string;
  opens_at: string;
  closes_at: string;
  cover_file_id?: string | null;
  products?: BatchProduct[];
};

type CatalogProduct = {
  id: string;
  name: string;
  preorder_eligible: boolean;
  status: string;
  base_price_idr: number;
};

type PurchasingLine = {
  product_id: string;
  product_name: string;
  required_qty: number;
  unpaid_demand: number;
  purchased_qty: number;
  unavailable_qty: number;
  outstanding_qty: number;
};

type OrderRow = {
  id: string;
  order_number: string;
  customer_name: string;
  total_idr: number;
  order_status: string;
  payment_status: string;
};

export function BatchDetailClient({
  storeId,
  batch,
  catalog,
  purchasing,
  orders,
}: {
  storeId: string;
  batch: Batch;
  catalog: CatalogProduct[];
  purchasing: PurchasingLine[];
  orders: OrderRow[];
}) {
  const router = useRouter();
  const [tab, setTab] = useState<"products" | "orders" | "purchasing">("products");
  const [productId, setProductId] = useState("");
  const [priceOverride, setPriceOverride] = useState("");
  const [qtyLimit, setQtyLimit] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [purchasedDraft, setPurchasedDraft] = useState<Record<string, string>>({});

  const eligible = useMemo(
    () => catalog.filter((p) => p.preorder_eligible && p.status === "published"),
    [catalog],
  );

  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await fn();
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Action failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-8 space-y-6">
      {error ? <p className="text-sm text-error">{error}</p> : null}
      <div className="flex flex-wrap gap-2">
        {batch.operational_status === "open" ? (
          <Button
            type="button"
            variant="outline"
            disabled={busy}
            onClick={() =>
              run(async () => {
                await apiFetch(`/api/v1/stores/${storeId}/preorders/${batch.id}`, {
                  method: "PATCH",
                  body: { operational_status: "closed" },
                });
              })
            }
          >
            Close early
          </Button>
        ) : null}
        {batch.operational_status === "closed" ? (
          <>
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await apiFetch(`/api/v1/stores/${storeId}/preorders/${batch.id}`, {
                    method: "PATCH",
                    body: { operational_status: "purchasing" },
                  });
                })
              }
            >
              Start purchasing
            </Button>
            <Button
              type="button"
              variant="outline"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  const closes = new Date(Date.now() + 3 * 24 * 60 * 60 * 1000).toISOString();
                  await apiFetch(`/api/v1/stores/${storeId}/preorders/${batch.id}`, {
                    method: "PATCH",
                    body: { operational_status: "open", closes_at: closes },
                  });
                })
              }
            >
              Reopen (3 days)
            </Button>
          </>
        ) : null}
        {batch.publication_status === "draft" ? (
          <Button
            type="button"
            disabled={busy}
            onClick={() =>
              run(async () => {
                await apiFetch(`/api/v1/stores/${storeId}/preorders/${batch.id}`, {
                  method: "PATCH",
                  body: { publication_status: "published" },
                });
              })
            }
          >
            Publish
          </Button>
        ) : null}
      </div>

      <div className="flex gap-4 border-b border-border text-sm">
        {(["products", "orders", "purchasing"] as const).map((t) => (
          <button
            key={t}
            type="button"
            className={`pb-2 capitalize ${tab === t ? "border-b-2 border-foreground font-medium" : "text-foreground-secondary"}`}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "products" ? (
        <div className="space-y-6">
          <BatchCoverEditor
            storeId={storeId}
            batchId={batch.id}
            coverFileId={batch.cover_file_id}
          />
          <form
            className="grid gap-3 rounded-[10px] border border-border p-5 sm:grid-cols-4"
            onSubmit={(e) => {
              e.preventDefault();
              run(async () => {
                await apiFetch(`/api/v1/stores/${storeId}/preorders/${batch.id}/products`, {
                  method: "POST",
                  body: {
                    product_id: productId,
                    price_override_idr: priceOverride ? Number(priceOverride) : undefined,
                    quantity_limit: qtyLimit ? Number(qtyLimit) : undefined,
                    availability: "available",
                  },
                });
                setProductId("");
                setPriceOverride("");
                setQtyLimit("");
              });
            }}
          >
            <div className="sm:col-span-2 space-y-2">
              <Label>Product</Label>
              <select
                className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
                value={productId}
                onChange={(e) => setProductId(e.target.value)}
                required
              >
                <option value="">Select product</option>
                {eligible.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} · {formatIDR(p.base_price_idr)}
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-2">
              <Label htmlFor="batch-price">Batch price (optional)</Label>
              <Input
                id="batch-price"
                value={priceOverride}
                onChange={(e) => setPriceOverride(e.target.value)}
                inputMode="numeric"
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="batch-quota">Quota (optional)</Label>
              <Input
                id="batch-quota"
                value={qtyLimit}
                onChange={(e) => setQtyLimit(e.target.value)}
                inputMode="numeric"
              />
            </div>
            <div className="sm:col-span-4">
              <Button type="submit" disabled={busy || !productId}>
                Assign product
              </Button>
            </div>
          </form>
          <div className="overflow-hidden rounded-[10px] border border-border">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-border bg-background-subtle text-xs font-medium text-foreground-muted">
                <tr>
                  <th className="px-4 py-3">Product</th>
                  <th className="px-4 py-3 text-right">Price</th>
                  <th className="px-4 py-3 text-right">Quota</th>
                  <th className="px-4 py-3 text-right">Reserved</th>
                  <th className="px-4 py-3" />
                </tr>
              </thead>
              <tbody>
                {(batch.products || []).map((p) => (
                  <tr key={p.id} className="border-b border-border last:border-0">
                    <td className="px-4 py-3">
                      <div>{p.product_name}</div>
                      <BatchVariantPairing
                        storeId={storeId}
                        batchId={batch.id}
                        productId={p.product_id}
                        productName={p.product_name}
                      />
                    </td>
                    <td className="px-4 py-3 text-right tabular-nums">{formatIDR(p.effective_price_idr)}</td>
                    <td className="px-4 py-3 text-right tabular-nums">{p.quantity_limit ?? "—"}</td>
                    <td className="px-4 py-3 text-right tabular-nums">{p.reserved_qty ?? 0}</td>
                    <td className="px-4 py-3 text-right">
                      <Button
                        type="button"
                        variant="ghost"
                        disabled={busy}
                        onClick={() =>
                          run(async () => {
                            await apiFetch(
                              `/api/v1/stores/${storeId}/preorders/${batch.id}/products/${p.product_id}`,
                              { method: "DELETE" },
                            );
                          })
                        }
                      >
                        Remove
                      </Button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      ) : null}

      {tab === "orders" ? (
        <div className="overflow-hidden rounded-[10px] border border-border">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-border bg-background-subtle text-xs font-medium text-foreground-muted">
              <tr>
                <th className="px-4 py-3">Order</th>
                <th className="px-4 py-3">Customer</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3 text-right">Total</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.id} className="border-b border-border last:border-0">
                  <td className="px-4 py-3">
                    <a className="hover:underline" href={`/stores/${storeId}/orders/${o.id}`}>
                      {o.order_number}
                    </a>
                  </td>
                  <td className="px-4 py-3">{o.customer_name}</td>
                  <td className="px-4 py-3 capitalize">
                    {o.order_status.replaceAll("_", " ")} · {o.payment_status.replaceAll("_", " ")}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums">{formatIDR(o.total_idr)}</td>
                </tr>
              ))}
              {orders.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-4 py-8 text-center text-foreground-secondary">
                    No orders in this batch yet.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      ) : null}

      {tab === "purchasing" ? (
        <div className="overflow-hidden rounded-[10px] border border-border">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-border bg-background-subtle text-xs font-medium text-foreground-muted">
              <tr>
                <th className="px-4 py-3">Product</th>
                <th className="px-4 py-3 text-right">Required (paid)</th>
                <th className="px-4 py-3 text-right">Unpaid demand</th>
                <th className="px-4 py-3 text-right">Purchased</th>
                <th className="px-4 py-3 text-right">Outstanding</th>
                <th className="px-4 py-3" />
              </tr>
            </thead>
            <tbody>
              {purchasing.map((line) => (
                <tr key={line.product_id} className="border-b border-border last:border-0">
                  <td className="px-4 py-3">{line.product_name}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{line.required_qty}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{line.unpaid_demand}</td>
                  <td className="px-4 py-3 text-right">
                    <Input
                      className="ml-auto h-9 w-24 text-right"
                      value={purchasedDraft[line.product_id] ?? String(line.purchased_qty)}
                      onChange={(e) =>
                        setPurchasedDraft((d) => ({ ...d, [line.product_id]: e.target.value }))
                      }
                    />
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums">{line.outstanding_qty}</td>
                  <td className="px-4 py-3 text-right">
                    <Button
                      type="button"
                      variant="outline"
                      disabled={busy}
                      onClick={() =>
                        run(async () => {
                          const qty = Number(purchasedDraft[line.product_id] ?? line.purchased_qty);
                          await apiFetch(`/api/v1/stores/${storeId}/preorders/${batch.id}/purchasing`, {
                            method: "PUT",
                            body: { product_id: line.product_id, purchased_qty: qty },
                          });
                        })
                      }
                    >
                      Save
                    </Button>
                  </td>
                </tr>
              ))}
              {purchasing.length === 0 ? (
                <tr>
                  <td colSpan={6} className="px-4 py-8 text-center text-foreground-secondary">
                    Purchasing list appears once orders exist.
                  </td>
                </tr>
              ) : null}
            </tbody>
          </table>
        </div>
      ) : null}
    </div>
  );
}
