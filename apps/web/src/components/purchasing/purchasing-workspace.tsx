"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
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

type Eligibility = {
  batch_id: string;
  batch_name: string;
  operational_status: string;
  campaign_id?: string | null;
  campaign_name?: string | null;
  paid_order_count: number;
  unpaid_order_count: number;
  required_line_qty: number;
  outstanding_line_qty: number;
  purchasing_eligible: boolean;
};

type PurchasingLine = {
  product_id: string;
  variant_id?: string | null;
  product_name: string;
  sku?: string | null;
  variant_label?: string | null;
  brand_id?: string | null;
  brand_name?: string | null;
  vendor_id?: string | null;
  vendor_name?: string | null;
  campaign_id?: string | null;
  batch_id?: string | null;
  batch_name?: string | null;
  customer_id?: string | null;
  customer_name?: string | null;
  required_qty: number;
  unpaid_demand: number;
  purchased_qty: number;
  unavailable_qty: number;
  allocated_qty: number;
  received_qty: number;
  outstanding_qty: number;
  line_status: string;
  version?: number;
  group_label?: string;
};

type GroupableAttribute = { id: string; label: string };

type LineStatus = "pending" | "partial" | "purchased" | "unavailable";

const STATUS_OPTIONS: { value: LineStatus; label: string }[] = [
  { value: "pending", label: "Pending" },
  { value: "purchased", label: "Purchased" },
  { value: "unavailable", label: "Unavailable" },
];

export function PurchasingWorkspace({ storeId }: { storeId: string }) {
  const [eligibility, setEligibility] = useState<Eligibility[]>([]);
  const [lines, setLines] = useState<PurchasingLine[]>([]);
  const [attributes, setAttributes] = useState<GroupableAttribute[]>([]);
  const [source, setSource] = useState<"orders" | "wishlist">("orders");
  const [batchId, setBatchId] = useState("");
  const [groupBy, setGroupBy] = useState("variant");
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [unavailDrafts, setUnavailDrafts] = useState<Record<string, string>>({});
  const [selected, setSelected] = useState<Record<string, boolean>>({});
  const [bulkStatus, setBulkStatus] = useState<LineStatus>("purchased");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const lineKey = useCallback((l: PurchasingLine) => {
    return `${l.batch_id || ""}:${l.customer_id || ""}:${l.product_id}:${l.variant_id || ""}`;
  }, []);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [elig, purch] = await Promise.all([
        apiFetch<{ eligibility: Eligibility[] }>(
          `/api/v1/stores/${storeId}/purchasing/eligibility`,
        ),
        apiFetch<{ purchasing: PurchasingLine[]; groupable_attributes: GroupableAttribute[] }>(
          `/api/v1/stores/${storeId}/purchasing?${new URLSearchParams({
            ...(source === "orders" && batchId ? { batch_id: batchId } : {}),
            source,
            group_by: groupBy,
          }).toString()}`,
        ),
      ]);
      setEligibility(elig.eligibility || []);
      setLines(purch.purchasing || []);
      setAttributes(purch.groupable_attributes || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load purchasing.");
    }
  }, [storeId, batchId, groupBy, source]);

  const displayLines = useMemo(() => {
    if (groupBy === "variant") return lines;
    return [...lines].sort(
      (a, b) =>
        (a.group_label || "").localeCompare(b.group_label || "") ||
        a.product_name.localeCompare(b.product_name),
    );
  }, [lines, groupBy]);

  useEffect(() => {
    void load();
  }, [load]);

  const eligibleBatches = useMemo(
    () => eligibility.filter((e) => e.purchasing_eligible || e.paid_order_count > 0),
    [eligibility],
  );

  const selectedKeys = useMemo(
    () => Object.keys(selected).filter((k) => selected[k]),
    [selected],
  );

  const allVisibleSelected =
    displayLines.length > 0 && displayLines.every((l) => selected[lineKey(l)]);

  function toggleAllVisible() {
    if (allVisibleSelected) {
      setSelected({});
      return;
    }
    const next: Record<string, boolean> = {};
    for (const line of displayLines) next[lineKey(line)] = true;
    setSelected(next);
  }

  async function applyStatus(targets: PurchasingLine[], status: LineStatus) {
    if (targets.length === 0) return;
    if (status === "partial") {
      setError("Pick Pending, Purchased, or Unavailable for status updates.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/purchasing/status`, {
        method: "POST",
        body: {
          status,
          lines: targets.map((l) => ({
            product_id: l.product_id,
            variant_id: l.variant_id || undefined,
            batch_id: l.batch_id || undefined,
          })),
        },
      });
      setSelected({});
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Status update failed.");
    } finally {
      setBusy(false);
    }
  }

  async function savePurchased(line: PurchasingLine) {
    if (!line.batch_id) {
      setError("Ready-stock lines use Status — pick Purchased or Unavailable.");
      return;
    }
    const key = lineKey(line);
    const purchased = Number(drafts[key] ?? line.purchased_qty);
    const unavailable = Number(unavailDrafts[key] ?? line.unavailable_qty);
    if (!Number.isFinite(purchased) || purchased < 0) {
      setError("Purchased qty must be a non-negative number.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/preorders/${line.batch_id}/purchasing`, {
        method: "PUT",
        body: {
          product_id: line.product_id,
          variant_id: line.variant_id || undefined,
          purchased_qty: purchased,
          unavailable_qty: Number.isFinite(unavailable) ? unavailable : 0,
          expected_version: line.version || undefined,
        },
      });
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Save failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Purchasing"
        description="Paid order demand (including Paid by admin) and wishlist interest. Update status per line or in bulk; group by customer or attributes."
        actions={
          <Button asChild variant="outline">
            <Link href={urls.storeAttributes(storeId)}>Attributes</Link>
          </Button>
        }
      />

      {error ? (
        <p className="rounded-lg border border-error-border bg-error-bg px-4 py-3 text-sm text-error">
          {error}
        </p>
      ) : null}

      <div className="space-y-4">
        <div className="flex gap-2" aria-label="Demand source">
          {(["orders", "wishlist"] as const).map((value) => (
            <button
              key={value}
              type="button"
              aria-pressed={source === value}
              onClick={() => {
                setSource(value);
                setGroupBy("variant");
                setSelected({});
              }}
              className={
                source === value
                  ? "rounded-md bg-ink px-4 py-2 text-sm font-semibold text-foreground-inverse"
                  : "rounded-md border border-border bg-background px-4 py-2 text-sm text-foreground-secondary"
              }
            >
              {value === "orders" ? "Orders" : "Wishlist"}
            </button>
          ))}
        </div>

        <div className="flex flex-wrap items-end gap-3 rounded-xl border border-border bg-background-subtle/60 p-4">
          {source === "orders" ? (
            <label className="space-y-1 text-sm">
              <span className="text-foreground-muted">Batch</span>
              <select
                className="block h-10 min-w-[12rem] rounded-md border border-border bg-background px-3"
                value={batchId}
                onChange={(e) => setBatchId(e.target.value)}
              >
                <option value="">All batches + ready stock</option>
                {eligibleBatches.map((b) => (
                  <option key={b.batch_id} value={b.batch_id}>
                    {b.batch_name}
                  </option>
                ))}
              </select>
            </label>
          ) : null}
          <label className="space-y-1 text-sm">
            <span className="text-foreground-muted">Group</span>
            <select
              className="block h-10 min-w-[10rem] rounded-md border border-border bg-background px-3"
              value={groupBy}
              onChange={(e) => {
                setGroupBy(e.target.value);
                setSelected({});
              }}
            >
              <option value="variant">Exact variant</option>
              <option value="brand">By native brand</option>
              <option value="campaign">By campaign</option>
              {source === "orders" ? <option value="customer">By customer</option> : null}
              {attributes.map((attribute) => (
                <option key={attribute.id} value={`attribute:${attribute.id}`}>
                  By {attribute.label}
                </option>
              ))}
            </select>
          </label>
        </div>

        {source === "orders" && selectedKeys.length > 0 ? (
          <div className="flex flex-wrap items-center gap-3 rounded-xl border border-border bg-background px-4 py-3">
            <span className="text-sm text-foreground-secondary">
              {selectedKeys.length} selected
            </span>
            <select
              className="h-9 rounded-md border border-border bg-background px-3 text-sm"
              value={bulkStatus}
              onChange={(e) => setBulkStatus(e.target.value as LineStatus)}
            >
              {STATUS_OPTIONS.map((o) => (
                <option key={o.value} value={o.value}>
                  {o.label}
                </option>
              ))}
            </select>
            <Button
              type="button"
              size="sm"
              disabled={busy}
              onClick={() =>
                void applyStatus(
                  displayLines.filter((l) => selected[lineKey(l)]),
                  bulkStatus,
                )
              }
            >
              Apply status
            </Button>
            <Button
              type="button"
              size="sm"
              variant="ghost"
              disabled={busy}
              onClick={() => setSelected({})}
            >
              Clear
            </Button>
          </div>
        ) : null}

        {lines.length === 0 ? (
          <EmptyState
            title={source === "wishlist" ? "No wishlist interest yet" : "No purchasing demand yet"}
            description={
              source === "wishlist"
                ? "Products saved by customers in live shopping will appear here."
                : "Paid orders (including Paid by admin) and unpaid preorder demand appear here."
            }
          />
        ) : (
          <Table stickyHeader>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                {source === "orders" ? (
                  <TableHead sticky className="w-10">
                    <input
                      type="checkbox"
                      aria-label="Select all"
                      checked={allVisibleSelected}
                      onChange={toggleAllVisible}
                    />
                  </TableHead>
                ) : null}
                <TableHead sticky>Product / variant</TableHead>
                {groupBy !== "variant" ? <TableHead sticky>Group</TableHead> : null}
                <TableHead sticky>
                  {source === "wishlist" ? "Live shopping" : "Batch"}
                </TableHead>
                {source === "orders" && groupBy === "customer" ? (
                  <TableHead sticky>Customer</TableHead>
                ) : null}
                <TableHead sticky className="text-right">
                  {source === "wishlist" ? "Wishlisted" : "Required"}
                </TableHead>
                {source === "orders" ? (
                  <>
                    <TableHead sticky className="text-right">
                      Unpaid
                    </TableHead>
                    <TableHead sticky>Status</TableHead>
                    <TableHead sticky className="text-right">
                      Purchased
                    </TableHead>
                    <TableHead sticky className="text-right">
                      Unavailable
                    </TableHead>
                    <TableHead sticky className="text-right">
                      Outstanding
                    </TableHead>
                    <TableHead sticky />
                  </>
                ) : null}
              </TableRow>
            </TableHeader>
            <TableBody>
              {displayLines.map((line) => {
                const key = lineKey(line);
                return (
                  <TableRow key={key}>
                    {source === "orders" ? (
                      <TableCell>
                        <input
                          type="checkbox"
                          aria-label={`Select ${line.product_name}`}
                          checked={Boolean(selected[key])}
                          onChange={(e) =>
                            setSelected((s) => ({ ...s, [key]: e.target.checked }))
                          }
                        />
                      </TableCell>
                    ) : null}
                    <TableCell>
                      <div className="font-medium text-foreground">{line.product_name}</div>
                      <div className="text-xs text-foreground-muted">
                        {[line.variant_label, line.brand_name, line.sku]
                          .filter(Boolean)
                          .join(" · ") || "—"}
                      </div>
                    </TableCell>
                    {groupBy !== "variant" ? (
                      <TableCell className="font-medium">
                        {line.group_label || "Unspecified"}
                      </TableCell>
                    ) : null}
                    <TableCell className="text-sm text-foreground-secondary">
                      {line.batch_name || (source === "orders" ? "Ready stock" : "—")}
                    </TableCell>
                    {source === "orders" && groupBy === "customer" ? (
                      <TableCell className="text-sm">
                        {line.customer_name || "—"}
                      </TableCell>
                    ) : null}
                    <TableCell className="text-right tabular-nums">
                      {line.required_qty}
                    </TableCell>
                    {source === "orders" ? (
                      <>
                        <TableCell className="text-right tabular-nums">
                          {line.unpaid_demand}
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <StatusBadge status={line.line_status || "pending"} />
                            <select
                              className="h-8 rounded-md border border-border bg-background px-2 text-xs"
                              value={
                                line.line_status === "partial"
                                  ? "pending"
                                  : line.line_status || "pending"
                              }
                              disabled={busy}
                              onChange={(e) =>
                                void applyStatus([line], e.target.value as LineStatus)
                              }
                            >
                              {STATUS_OPTIONS.map((o) => (
                                <option key={o.value} value={o.value}>
                                  {o.label}
                                </option>
                              ))}
                            </select>
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          {line.batch_id ? (
                            <Input
                              className="ml-auto h-9 w-20 text-right"
                              value={drafts[key] ?? String(line.purchased_qty)}
                              onChange={(e) =>
                                setDrafts((d) => ({ ...d, [key]: e.target.value }))
                              }
                            />
                          ) : (
                            <span className="tabular-nums">{line.purchased_qty}</span>
                          )}
                        </TableCell>
                        <TableCell className="text-right">
                          {line.batch_id ? (
                            <Input
                              className="ml-auto h-9 w-20 text-right"
                              value={unavailDrafts[key] ?? String(line.unavailable_qty)}
                              onChange={(e) =>
                                setUnavailDrafts((d) => ({
                                  ...d,
                                  [key]: e.target.value,
                                }))
                              }
                            />
                          ) : (
                            <span className="tabular-nums">{line.unavailable_qty}</span>
                          )}
                        </TableCell>
                        <TableCell className="text-right tabular-nums">
                          {line.outstanding_qty}
                        </TableCell>
                        <TableCell className="text-right">
                          {line.batch_id ? (
                            <Button
                              type="button"
                              size="sm"
                              variant="outline"
                              disabled={busy}
                              onClick={() => void savePurchased(line)}
                            >
                              Save
                            </Button>
                          ) : null}
                        </TableCell>
                      </>
                    ) : null}
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  );
}
