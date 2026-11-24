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
  required_qty: number;
  unpaid_demand: number;
  purchased_qty: number;
  unavailable_qty: number;
  allocated_qty: number;
  received_qty: number;
  outstanding_qty: number;
  version?: number;
};

type Vendor = { id: string; name: string; status: string };

type Tab = "requirements" | "eligibility" | "receiving";

export function PurchasingWorkspace({ storeId }: { storeId: string }) {
  const [tab, setTab] = useState<Tab>("requirements");
  const [eligibility, setEligibility] = useState<Eligibility[]>([]);
  const [lines, setLines] = useState<PurchasingLine[]>([]);
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [batchId, setBatchId] = useState("");
  const [vendorId, setVendorId] = useState("");
  const [groupBy, setGroupBy] = useState("variant");
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [unavailDrafts, setUnavailDrafts] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [receiving, setReceiving] = useState<
    Array<{ id: string; status: string; batch_name?: string | null; vendor_name?: string | null; notes?: string | null }>
  >([]);

  const lineKey = useCallback((l: PurchasingLine) => {
    return `${l.batch_id || ""}:${l.product_id}:${l.variant_id || ""}:${l.vendor_id || ""}`;
  }, []);

  const load = useCallback(async () => {
    setError(null);
    try {
      const [elig, purch, vend, recv] = await Promise.all([
        apiFetch<{ eligibility: Eligibility[] }>(
          `/api/v1/stores/${storeId}/purchasing/eligibility`,
        ),
        apiFetch<{ purchasing: PurchasingLine[] }>(
          `/api/v1/stores/${storeId}/purchasing?${new URLSearchParams({
            ...(batchId ? { batch_id: batchId } : {}),
            ...(vendorId ? { vendor_id: vendorId } : {}),
            group_by: groupBy,
          }).toString()}`,
        ),
        apiFetch<{ vendors: Vendor[] }>(`/api/v1/stores/${storeId}/vendors`),
        apiFetch<{ receiving: typeof receiving }>(
          `/api/v1/stores/${storeId}/receiving`,
        ),
      ]);
      setEligibility(elig.eligibility || []);
      setLines(purch.purchasing || []);
      setVendors((vend.vendors || []).filter((v) => v.status === "active"));
      setReceiving(recv.receiving || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load purchasing.");
    }
  }, [storeId, batchId, vendorId, groupBy]);

  useEffect(() => {
    void load();
  }, [load]);

  const eligibleBatches = useMemo(
    () => eligibility.filter((e) => e.purchasing_eligible),
    [eligibility],
  );

  async function savePurchased(line: PurchasingLine) {
    if (!line.batch_id) return;
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
          vendor_id: vendorId || line.vendor_id || undefined,
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

  async function createReceivingFromBatch() {
    if (!batchId) {
      setError("Select a batch before creating a receiving record.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const bodyLines = lines
        .filter((l) => l.batch_id === batchId && l.purchased_qty > 0)
        .map((l) => ({
          product_id: l.product_id,
          variant_id: l.variant_id || undefined,
          expected_qty: l.purchased_qty,
          received_qty: 0,
        }));
      await apiFetch(`/api/v1/stores/${storeId}/receiving`, {
        method: "POST",
        body: {
          batch_id: batchId,
          vendor_id: vendorId || undefined,
          notes: "Created from purchasing workspace",
          lines: bodyLines,
        },
      });
      setTab("receiving");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Receiving create failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Purchasing"
        description="Aggregate paid demand by exact variant, record purchases per vendor, allocate arrivals, and hand off to receiving."
        actions={
          <Button asChild variant="outline">
            <Link href={urls.storeVendors(storeId)}>Vendors</Link>
          </Button>
        }
      />

      {error ? (
        <p className="rounded-lg border border-error-border bg-error-bg px-4 py-3 text-sm text-error">
          {error}
        </p>
      ) : null}

      <div className="flex flex-wrap gap-2">
        {(["requirements", "eligibility", "receiving"] as const).map((t) => (
          <button
            key={t}
            type="button"
            onClick={() => setTab(t)}
            className={
              tab === t
                ? "rounded-md bg-dark px-4 py-2 text-sm font-medium text-foreground-inverse"
                : "rounded-md border border-border bg-background px-4 py-2 text-sm text-foreground-secondary hover:bg-background-subtle"
            }
          >
            {t === "requirements" ? "Requirements" : t === "eligibility" ? "Eligible batches" : "Receiving"}
          </button>
        ))}
      </div>

      {tab === "requirements" ? (
        <div className="space-y-4">
          <div className="flex flex-wrap items-end gap-3 rounded-xl border border-border bg-background-subtle/60 p-4">
            <label className="space-y-1 text-sm">
              <span className="text-foreground-muted">Batch</span>
              <select
                className="block h-10 min-w-[12rem] rounded-md border border-border bg-background px-3"
                value={batchId}
                onChange={(e) => setBatchId(e.target.value)}
              >
                <option value="">All eligible</option>
                {eligibleBatches.map((b) => (
                  <option key={b.batch_id} value={b.batch_id}>
                    {b.batch_name}
                  </option>
                ))}
              </select>
            </label>
            <label className="space-y-1 text-sm">
              <span className="text-foreground-muted">Vendor</span>
              <select
                className="block h-10 min-w-[12rem] rounded-md border border-border bg-background px-3"
                value={vendorId}
                onChange={(e) => setVendorId(e.target.value)}
              >
                <option value="">All vendors</option>
                {vendors.map((v) => (
                  <option key={v.id} value={v.id}>
                    {v.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="space-y-1 text-sm">
              <span className="text-foreground-muted">Group</span>
              <select
                className="block h-10 min-w-[10rem] rounded-md border border-border bg-background px-3"
                value={groupBy}
                onChange={(e) => setGroupBy(e.target.value)}
              >
                <option value="variant">Exact variant</option>
                <option value="vendor">By vendor</option>
                <option value="brand">By brand</option>
                <option value="campaign">By campaign</option>
              </select>
            </label>
            <Button type="button" variant="outline" disabled={busy || !batchId} onClick={() => void createReceivingFromBatch()}>
              Start receiving
            </Button>
          </div>

          {lines.length === 0 ? (
            <EmptyState
              title="No purchasing demand yet"
              description="Close a preorder batch with paid orders, or create admin orders linked to a campaign batch."
            />
          ) : (
            <Table stickyHeader>
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead sticky>Product / variant</TableHead>
                  <TableHead sticky>Batch</TableHead>
                  <TableHead sticky>Vendor</TableHead>
                  <TableHead sticky className="text-right">
                    Required
                  </TableHead>
                  <TableHead sticky className="text-right">
                    Unpaid
                  </TableHead>
                  <TableHead sticky className="text-right">
                    Purchased
                  </TableHead>
                  <TableHead sticky className="text-right">
                    Unavailable
                  </TableHead>
                  <TableHead sticky className="text-right">
                    Allocated
                  </TableHead>
                  <TableHead sticky className="text-right">
                    Outstanding
                  </TableHead>
                  <TableHead sticky />
                </TableRow>
              </TableHeader>
              <TableBody>
                {lines.map((line) => {
                  const key = lineKey(line);
                  return (
                    <TableRow key={key}>
                      <TableCell>
                        <div className="font-medium text-foreground">{line.product_name}</div>
                        <div className="text-xs text-foreground-muted">
                          {[line.variant_label, line.brand_name, line.sku].filter(Boolean).join(" · ") || "—"}
                        </div>
                      </TableCell>
                      <TableCell className="text-sm text-foreground-secondary">
                        {line.batch_name || "—"}
                      </TableCell>
                      <TableCell className="text-sm text-foreground-secondary">
                        {line.vendor_name || "Unassigned"}
                      </TableCell>
                      <TableCell className="text-right tabular-nums">{line.required_qty}</TableCell>
                      <TableCell className="text-right tabular-nums">{line.unpaid_demand}</TableCell>
                      <TableCell className="text-right">
                        <Input
                          className="ml-auto h-9 w-20 text-right"
                          value={drafts[key] ?? String(line.purchased_qty)}
                          onChange={(e) =>
                            setDrafts((d) => ({ ...d, [key]: e.target.value }))
                          }
                        />
                      </TableCell>
                      <TableCell className="text-right">
                        <Input
                          className="ml-auto h-9 w-20 text-right"
                          value={unavailDrafts[key] ?? String(line.unavailable_qty)}
                          onChange={(e) =>
                            setUnavailDrafts((d) => ({ ...d, [key]: e.target.value }))
                          }
                        />
                      </TableCell>
                      <TableCell className="text-right tabular-nums">{line.allocated_qty}</TableCell>
                      <TableCell className="text-right tabular-nums">{line.outstanding_qty}</TableCell>
                      <TableCell className="text-right">
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          disabled={busy || !line.batch_id}
                          onClick={() => void savePurchased(line)}
                        >
                          Save
                        </Button>
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </div>
      ) : null}

      {tab === "eligibility" ? (
        eligibility.length === 0 ? (
          <EmptyState
            title="No eligible batches"
            description="Batches become eligible after close, or when they enter purchasing / awaiting arrival."
          />
        ) : (
          <Table stickyHeader>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead sticky>Batch</TableHead>
                <TableHead sticky>Campaign</TableHead>
                <TableHead sticky>Status</TableHead>
                <TableHead sticky className="text-right">
                  Paid orders
                </TableHead>
                <TableHead sticky className="text-right">
                  Required qty
                </TableHead>
                <TableHead sticky className="text-right">
                  Outstanding
                </TableHead>
                <TableHead sticky />
              </TableRow>
            </TableHeader>
            <TableBody>
              {eligibility.map((e) => (
                <TableRow key={e.batch_id}>
                  <TableCell className="font-medium">{e.batch_name}</TableCell>
                  <TableCell>{e.campaign_name || "—"}</TableCell>
                  <TableCell>
                    <StatusBadge status={e.operational_status} />
                  </TableCell>
                  <TableCell className="text-right tabular-nums">{e.paid_order_count}</TableCell>
                  <TableCell className="text-right tabular-nums">{e.required_line_qty}</TableCell>
                  <TableCell className="text-right tabular-nums">{e.outstanding_line_qty}</TableCell>
                  <TableCell className="text-right">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setBatchId(e.batch_id);
                        setTab("requirements");
                      }}
                    >
                      Open
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )
      ) : null}

      {tab === "receiving" ? (
        receiving.length === 0 ? (
          <EmptyState
            title="No receiving records"
            description="Record purchases first, then start receiving from the requirements tab to hand off arrivals."
          />
        ) : (
          <Table stickyHeader>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead sticky>Batch</TableHead>
                <TableHead sticky>Vendor</TableHead>
                <TableHead sticky>Status</TableHead>
                <TableHead sticky>Notes</TableHead>
                <TableHead sticky />
              </TableRow>
            </TableHeader>
            <TableBody>
              {receiving.map((r) => (
                <TableRow key={r.id}>
                  <TableCell>{r.batch_name || "—"}</TableCell>
                  <TableCell>{r.vendor_name || "—"}</TableCell>
                  <TableCell>
                    <StatusBadge status={r.status} />
                  </TableCell>
                  <TableCell className="max-w-xs truncate text-sm text-foreground-secondary">
                    {r.notes || "—"}
                  </TableCell>
                  <TableCell className="text-right">
                    <Link
                      href={`/stores/${storeId}/purchasing/receiving/${r.id}`}
                      className="text-sm font-medium text-brand-dark underline"
                    >
                      Open
                    </Link>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )
      ) : null}
    </div>
  );
}
