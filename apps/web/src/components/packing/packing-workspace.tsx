"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type QueueOrder = {
  order_id: string;
  order_number: string;
  customer_name: string;
  customer_email: string;
  fulfillment_method: string;
  fulfillment_status: string;
  payment_status: string;
  packable_qty: number;
  packed_qty: number;
  remaining_qty: number;
  packing_status?: string | null;
};

type CustomerGroup = {
  customer_key: string;
  customer_name: string;
  customer_email: string;
  order_count: number;
  packable_qty: number;
  remaining_qty: number;
  orders: QueueOrder[];
};

type ChecklistItem = {
  order_item_id: string;
  product_name: string;
  sku?: string | null;
  variant_label?: string | null;
  quantity: number;
  availability_status: string;
  available_qty: number;
  packed_qty: number;
  shipped_qty: number;
  remaining_to_pack: number;
  packable: boolean;
};

type Checklist = {
  order_id: string;
  order_number: string;
  customer_name: string;
  customer_email: string;
  fulfillment_method: string;
  fulfillment_status: string;
  packing_eligible: boolean;
  packing_status?: string | null;
  items: ChecklistItem[];
  all_packed: boolean;
};

export function PackingWorkspace({ storeId }: { storeId: string }) {
  const [groupBy, setGroupBy] = useState<"order" | "customer">("order");
  const [orders, setOrders] = useState<QueueOrder[]>([]);
  const [customers, setCustomers] = useState<CustomerGroup[]>([]);
  const [selectedOrderId, setSelectedOrderId] = useState<string | null>(null);
  const [checklist, setChecklist] = useState<Checklist | null>(null);
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const loadQueue = useCallback(async () => {
    setError(null);
    try {
      const data = await apiFetch<{
        orders?: QueueOrder[];
        customers?: CustomerGroup[];
        group_by: string;
      }>(`/api/v1/stores/${storeId}/packing?group_by=${groupBy}&status=eligible`);
      setOrders(data.orders || []);
      setCustomers(data.customers || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load packing queue.");
    }
  }, [storeId, groupBy]);

  const loadChecklist = useCallback(
    async (orderId: string) => {
      setError(null);
      try {
        const data = await apiFetch<{ packing: Checklist }>(
          `/api/v1/stores/${storeId}/packing/${orderId}`,
        );
        setChecklist(data.packing);
        const next: Record<string, string> = {};
        for (const it of data.packing.items || []) {
          next[it.order_item_id] = String(it.packed_qty);
        }
        setDrafts(next);
      } catch (e) {
        setError(e instanceof ApiError ? e.message : "Failed to load checklist.");
      }
    },
    [storeId],
  );

  useEffect(() => {
    void loadQueue();
  }, [loadQueue]);

  useEffect(() => {
    if (selectedOrderId) void loadChecklist(selectedOrderId);
  }, [selectedOrderId, loadChecklist]);

  async function saveChecklist() {
    if (!checklist) return;
    setBusy(true);
    setError(null);
    try {
      const items = checklist.items.map((it) => ({
        order_item_id: it.order_item_id,
        qty_packed: Number(drafts[it.order_item_id] ?? it.packed_qty),
      }));
      const data = await apiFetch<{ packing: Checklist }>(
        `/api/v1/stores/${storeId}/packing/${checklist.order_id}/checklist`,
        { method: "PUT", body: { items } },
      );
      setChecklist(data.packing);
      await loadQueue();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Save failed.");
    } finally {
      setBusy(false);
    }
  }

  async function completePacking() {
    if (!checklist) return;
    setBusy(true);
    setError(null);
    try {
      // Persist drafts first when present
      const items = checklist.items.map((it) => ({
        order_item_id: it.order_item_id,
        qty_packed: Number(drafts[it.order_item_id] ?? it.available_qty),
      }));
      await apiFetch(`/api/v1/stores/${storeId}/packing/${checklist.order_id}/checklist`, {
        method: "PUT",
        body: { items },
      });
      const data = await apiFetch<{ packing: Checklist }>(
        `/api/v1/stores/${storeId}/packing/${checklist.order_id}/complete`,
        { method: "POST", body: {} },
      );
      setChecklist(data.packing);
      await loadQueue();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Complete packing failed.");
    } finally {
      setBusy(false);
    }
  }

  const flatOrders =
    groupBy === "customer"
      ? customers.flatMap((c) => c.orders)
      : orders;

  return (
    <div className="space-y-6">
      <PageHeader
        title="Packing"
        description="Queue by order or customer. Pack available items, then mark ready to ship or pickup."
        actions={
          <div className="flex gap-2">
            <Button
              type="button"
              variant={groupBy === "order" ? "primary" : "secondary"}
              onClick={() => setGroupBy("order")}
            >
              By order
            </Button>
            <Button
              type="button"
              variant={groupBy === "customer" ? "primary" : "secondary"}
              onClick={() => setGroupBy("customer")}
            >
              By customer
            </Button>
          </div>
        }
      />

      {error ? <p className="text-sm text-error">{error}</p> : null}

      <div className="grid gap-6 lg:grid-cols-[1fr_1.1fr]">
        <section className="rounded-xl border border-border bg-background p-5">
          <h2 className="text-card font-semibold text-foreground">Queue</h2>
          {flatOrders.length === 0 ? (
            <EmptyState
              title="Nothing to pack"
              description="Orders appear here when items are available after receiving or ready-stock payment."
            />
          ) : groupBy === "customer" ? (
            <div className="mt-4 space-y-4">
              {customers.map((c) => (
                <div key={c.customer_key}>
                  <p className="text-sm font-medium text-foreground">
                    {c.customer_name}{" "}
                    <span className="font-normal text-foreground-secondary">
                      · {c.order_count} orders · {c.remaining_qty} left
                    </span>
                  </p>
                  <ul className="mt-2 space-y-1">
                    {c.orders.map((o) => (
                      <li key={o.order_id}>
                        <button
                          type="button"
                          className={`w-full rounded-[10px] px-3 py-2 text-left text-sm hover:bg-surface-hover ${
                            selectedOrderId === o.order_id ? "bg-surface-active" : ""
                          }`}
                          onClick={() => setSelectedOrderId(o.order_id)}
                        >
                          <span className="font-medium">{o.order_number}</span>
                          <span className="ml-2 text-foreground-secondary">
                            {o.remaining_qty} remaining
                          </span>
                        </button>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          ) : (
            <Table className="mt-4">
              <TableHeader>
                <TableRow>
                  <TableHead>Order</TableHead>
                  <TableHead>Customer</TableHead>
                  <TableHead>Left</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {orders.map((o) => (
                  <TableRow
                    key={o.order_id}
                    className={`cursor-pointer ${selectedOrderId === o.order_id ? "bg-surface-active" : ""}`}
                    onClick={() => setSelectedOrderId(o.order_id)}
                  >
                    <TableCell className="font-medium">{o.order_number}</TableCell>
                    <TableCell>
                      <div>{o.customer_name}</div>
                      <div className="text-xs text-foreground-muted">{o.customer_email}</div>
                    </TableCell>
                    <TableCell className="tabular-nums">{o.remaining_qty}</TableCell>
                    <TableCell>
                      <StatusBadge status={o.fulfillment_status} />
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </section>

        <section className="rounded-xl border border-border bg-background p-5">
          <h2 className="text-card font-semibold text-foreground">Pack checklist</h2>
          {!checklist ? (
            <p className="mt-4 text-sm text-foreground-secondary">
              Select an order from the queue to pack items.
            </p>
          ) : (
            <>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                <p className="text-sm font-medium">{checklist.order_number}</p>
                <StatusBadge status={checklist.fulfillment_status} />
                <Link
                  href={urls.storeOrder(storeId, checklist.order_id)}
                  className="text-sm text-foreground-secondary hover:text-foreground"
                >
                  Open order
                </Link>
                {checklist.fulfillment_method !== "pickup" &&
                checklist.fulfillment_status === "ready_to_ship" ? (
                  <Link
                    href={urls.storeShipments(storeId) + `?order=${checklist.order_id}`}
                    className="text-sm font-medium text-foreground hover:underline"
                  >
                    Create shipment →
                  </Link>
                ) : null}
              </div>
              <p className="mt-1 text-sm text-foreground-secondary">
                {checklist.customer_name} · {checklist.customer_email}
              </p>

              <ul className="mt-4 space-y-3">
                {checklist.items.map((it) => (
                  <li
                    key={it.order_item_id}
                    className="flex flex-wrap items-center justify-between gap-3 border-b border-border-subtle pb-3"
                  >
                    <div>
                      <p className="text-sm font-medium">{it.product_name}</p>
                      <p className="text-xs text-foreground-muted">
                        Qty {it.quantity}
                        {it.variant_label ? ` · ${it.variant_label}` : ""}
                        {it.packable
                          ? ` · available ${it.available_qty}`
                          : ` · ${it.availability_status.replaceAll("_", " ")}`}
                        {it.shipped_qty > 0 ? ` · shipped ${it.shipped_qty}` : ""}
                      </p>
                    </div>
                    <div className="flex items-center gap-2">
                      <label className="text-xs text-foreground-secondary">Packed</label>
                      <input
                        type="number"
                        min={0}
                        max={it.available_qty}
                        disabled={!it.packable || busy}
                        className="w-20 rounded-[10px] border border-border bg-background px-2 py-1.5 text-sm tabular-nums"
                        value={drafts[it.order_item_id] ?? String(it.packed_qty)}
                        onChange={(e) =>
                          setDrafts((d) => ({ ...d, [it.order_item_id]: e.target.value }))
                        }
                      />
                    </div>
                  </li>
                ))}
              </ul>

              <div className="mt-5 flex flex-wrap gap-2">
                <Button type="button" variant="secondary" disabled={busy} onClick={() => void saveChecklist()}>
                  Save checklist
                </Button>
                <Button type="button" disabled={busy || !checklist.packing_eligible} onClick={() => void completePacking()}>
                  Mark packed
                </Button>
                <Link
                  href={urls.storeMessages(storeId, {
                    audience: "customer",
                    code: "WA-C10",
                    order_id: checklist.order_id,
                  })}
                  className="inline-flex h-10 items-center rounded-[10px] border border-border bg-background px-4 text-sm font-semibold text-foreground hover:bg-surface-hover"
                >
                  WhatsApp packing update
                </Link>
              </div>
            </>
          )}
        </section>
      </div>
    </div>
  );
}
