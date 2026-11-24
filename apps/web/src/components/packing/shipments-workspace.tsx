"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
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

type Shipment = {
  id: string;
  order_id: string;
  order_number?: string;
  customer_name?: string;
  carrier: string;
  tracking_number: string;
  status: string;
  shipped_at?: string | null;
  created_at: string;
  items?: Array<{
    order_item_id: string;
    product_name?: string;
    quantity: number;
  }>;
};

type PackItem = {
  order_item_id: string;
  product_name: string;
  packed_qty: number;
  shipped_qty: number;
  remaining_to_pack: number;
};

type Packing = {
  order_id: string;
  order_number: string;
  fulfillment_status: string;
  items: PackItem[];
};

export function ShipmentsWorkspace({ storeId }: { storeId: string }) {
  const search = useSearchParams();
  const preselectOrder = search.get("order") || "";

  const [shipments, setShipments] = useState<Shipment[]>([]);
  const [orderId, setOrderId] = useState(preselectOrder);
  const [packing, setPacking] = useState<Packing | null>(null);
  const [carrier, setCarrier] = useState("JNE");
  const [tracking, setTracking] = useState("");
  const [qtyDrafts, setQtyDrafts] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const loadShipments = useCallback(async () => {
    setError(null);
    try {
      const data = await apiFetch<{ shipments: Shipment[] }>(
        `/api/v1/stores/${storeId}/shipments`,
      );
      setShipments(data.shipments || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load shipments.");
    }
  }, [storeId]);

  const loadPacking = useCallback(
    async (oid: string) => {
      if (!oid) {
        setPacking(null);
        return;
      }
      try {
        const data = await apiFetch<{ packing: Packing }>(
          `/api/v1/stores/${storeId}/packing/${oid}`,
        );
        setPacking(data.packing);
        const drafts: Record<string, string> = {};
        for (const it of data.packing.items || []) {
          const remain = Math.max(0, it.packed_qty - it.shipped_qty);
          drafts[it.order_item_id] = remain > 0 ? String(remain) : "0";
        }
        setQtyDrafts(drafts);
      } catch (e) {
        setError(e instanceof ApiError ? e.message : "Failed to load order packing.");
        setPacking(null);
      }
    },
    [storeId],
  );

  useEffect(() => {
    void loadShipments();
  }, [loadShipments]);

  useEffect(() => {
    if (preselectOrder) {
      setOrderId(preselectOrder);
      void loadPacking(preselectOrder);
    }
  }, [preselectOrder, loadPacking]);

  const shippableItems = useMemo(() => {
    if (!packing) return [];
    return packing.items.filter((it) => it.packed_qty - it.shipped_qty > 0);
  }, [packing]);

  async function createShipment() {
    if (!orderId || !packing) {
      setError("Select a packed order first.");
      return;
    }
    const items = shippableItems
      .map((it) => ({
        order_item_id: it.order_item_id,
        quantity: Number(qtyDrafts[it.order_item_id] || 0),
      }))
      .filter((it) => it.quantity > 0);
    if (!carrier.trim() || !tracking.trim() || items.length === 0) {
      setError("Carrier, tracking number, and at least one item qty are required.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/shipments`, {
        method: "POST",
        body: {
          order_id: orderId,
          carrier: carrier.trim(),
          tracking_number: tracking.trim(),
          items,
          notify_customer: true,
        },
      });
      setTracking("");
      await loadShipments();
      await loadPacking(orderId);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Create shipment failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Shipments"
        description="Create carrier parcels with tracking. Partial and multi-parcel ships are supported."
        actions={
          <Link href={urls.storePacking(storeId)} className="text-sm font-medium hover:underline">
            ← Packing
          </Link>
        }
      />

      {error ? <p className="text-sm text-error">{error}</p> : null}

      <section className="rounded-xl border border-border bg-background p-5">
        <h2 className="text-card font-semibold">Create shipment</h2>
        <div className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <label className="text-xs text-foreground-secondary">Order ID</label>
            <Input
              value={orderId}
              onChange={(e) => setOrderId(e.target.value)}
              placeholder="UUID"
              className="mt-1"
            />
          </div>
          <div className="flex items-end">
            <Button
              type="button"
              variant="secondary"
              disabled={!orderId || busy}
              onClick={() => void loadPacking(orderId)}
            >
              Load packed items
            </Button>
          </div>
          <div>
            <label className="text-xs text-foreground-secondary">Carrier</label>
            <Input
              value={carrier}
              onChange={(e) => setCarrier(e.target.value)}
              className="mt-1"
              placeholder="JNE / SiCepat / J&T"
            />
          </div>
          <div>
            <label className="text-xs text-foreground-secondary">Tracking number</label>
            <Input
              value={tracking}
              onChange={(e) => setTracking(e.target.value)}
              className="mt-1"
            />
          </div>
        </div>

        {packing ? (
          <div className="mt-4">
            <p className="text-sm">
              <span className="font-medium">{packing.order_number}</span>{" "}
              <StatusBadge status={packing.fulfillment_status} />
            </p>
            {shippableItems.length === 0 ? (
              <p className="mt-2 text-sm text-foreground-secondary">
                No packed quantity remaining to ship. Pack the order first.
              </p>
            ) : (
              <ul className="mt-3 space-y-2">
                {shippableItems.map((it) => {
                  const remain = it.packed_qty - it.shipped_qty;
                  return (
                    <li
                      key={it.order_item_id}
                      className="flex flex-wrap items-center justify-between gap-3 text-sm"
                    >
                      <span>
                        {it.product_name}{" "}
                        <span className="text-foreground-muted">
                          · packed {it.packed_qty} · already shipped {it.shipped_qty}
                        </span>
                      </span>
                      <div className="flex items-center gap-2">
                        <label className="text-xs text-foreground-secondary">Ship qty</label>
                        <input
                          type="number"
                          min={0}
                          max={remain}
                          className="w-20 rounded-[10px] border border-border px-2 py-1.5 tabular-nums"
                          value={qtyDrafts[it.order_item_id] ?? "0"}
                          onChange={(e) =>
                            setQtyDrafts((d) => ({ ...d, [it.order_item_id]: e.target.value }))
                          }
                        />
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
            <Button
              type="button"
              className="mt-4"
              disabled={busy || shippableItems.length === 0}
              onClick={() => void createShipment()}
            >
              Create shipment & notify
            </Button>
          </div>
        ) : null}
      </section>

      <section className="rounded-xl border border-border bg-background p-5">
        <h2 className="text-card font-semibold">Recent shipments</h2>
        {shipments.length === 0 ? (
          <EmptyState
            title="No shipments yet"
            description="Create a parcel after packing an order that uses shipping."
          />
        ) : (
          <Table className="mt-4">
            <TableHeader>
              <TableRow>
                <TableHead>Order</TableHead>
                <TableHead>Carrier</TableHead>
                <TableHead>Tracking</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Created</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {shipments.map((sh) => (
                <TableRow key={sh.id}>
                  <TableCell>
                    <Link
                      href={urls.storeOrder(storeId, sh.order_id)}
                      className="font-medium hover:underline"
                    >
                      {sh.order_number || sh.order_id.slice(0, 8)}
                    </Link>
                    {sh.customer_name ? (
                      <div className="text-xs text-foreground-muted">{sh.customer_name}</div>
                    ) : null}
                  </TableCell>
                  <TableCell>{sh.carrier}</TableCell>
                  <TableCell className="font-mono text-sm">{sh.tracking_number}</TableCell>
                  <TableCell>
                    <StatusBadge status={sh.status} />
                  </TableCell>
                  <TableCell className="text-sm text-foreground-secondary">
                    {new Date(sh.created_at).toLocaleString("id-ID", {
                      timeZone: "Asia/Jakarta",
                    })}
                  </TableCell>
                  <TableCell>
                    <Link
                      href={urls.storeMessages(storeId, {
                        audience: "customer",
                        code: "WA-C12",
                        order_id: sh.order_id,
                      })}
                      className="text-sm font-medium text-foreground hover:underline"
                    >
                      WhatsApp
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </section>
    </div>
  );
}
