"use client";

import { useCallback, useEffect, useState } from "react";
import { apiFetch, ApiError } from "@/lib/api";
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

type FinanceRow = {
  order_id: string;
  order_number: string;
  customer_name: string;
  customer_email: string;
  original_total_idr: number;
  unavailable_delta_idr: number;
  credit_applied_idr: number;
  payable_idr: number;
  paid_idr: number;
  refund_entitled_idr: number;
  remaining_due_idr: number;
  refunded_idr: number;
  payment_status: string;
  settlement_status: string;
};

function formatIdr(n: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(n);
}

export function FinanceWorkspace({ storeId }: { storeId: string }) {
  const [rows, setRows] = useState<FinanceRow[]>([]);
  const [filter, setFilter] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<FinanceRow | null>(null);
  const [refundAmount, setRefundAmount] = useState("");

  const load = useCallback(async () => {
    setError(null);
    try {
      const q = filter ? `?settlement_status=${encodeURIComponent(filter)}` : "";
      const data = await apiFetch<{ orders: FinanceRow[] }>(
        `/api/v1/stores/${storeId}/finance${q}`,
      );
      setRows(data.orders || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load finance workspace");
    }
  }, [storeId, filter]);

  useEffect(() => {
    void load();
  }, [load]);

  async function createAndCompleteRefund() {
    if (!selected) return;
    const amount = Number(refundAmount || selected.refund_entitled_idr);
    if (!Number.isFinite(amount) || amount <= 0) {
      setError("Refund amount must be a positive integer");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const created = await apiFetch<{ refund: { id: string; version: number } }>(
        `/api/v1/stores/${storeId}/finance/refunds`,
        {
          method: "POST",
          body: JSON.stringify({
            order_id: selected.order_id,
            amount_idr: Math.round(amount),
            method: "bank_transfer",
            idempotency_key: `ui-refund:${selected.order_id}:${Math.round(amount)}`,
            reason: "Merchant refund from finance workspace",
          }),
        },
      );
      await apiFetch(`/api/v1/stores/${storeId}/finance/refunds/${created.refund.id}/process`, {
        method: "POST",
        body: JSON.stringify({
          expected_version: created.refund.version,
          status: "completed",
        }),
      });
      setSelected(null);
      setRefundAmount("");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Refund failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Payments"
        description="Reconcile expected payable, received deposits, and refund entitlements."
      />

      {error ? (
        <p className="rounded-lg border border-error-border bg-error-bg px-4 py-3 text-sm text-error">
          {error}
        </p>
      ) : null}

      <div className="flex flex-wrap items-end gap-3 rounded-xl border border-border bg-background-subtle/60 p-4">
        <label className="space-y-1 text-sm">
          <span className="text-foreground-muted">Settlement</span>
          <select
            className="block h-10 min-w-[12rem] rounded-md border border-border bg-background px-3"
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
          >
            <option value="">All</option>
            <option value="open">Open</option>
            <option value="partially_settled">Partially settled</option>
            <option value="settled">Settled</option>
            <option value="refund_due">Refund due</option>
            <option value="refunded">Refunded</option>
          </select>
        </label>
        <Button type="button" variant="outline" onClick={() => void load()} disabled={busy}>
          Refresh
        </Button>
      </div>

      {rows.length === 0 ? (
        <EmptyState
          title="No orders in this view"
          description="Verified payments and unavailable adjustments will appear here for reconciliation."
        />
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Order</TableHead>
              <TableHead>Customer</TableHead>
              <TableHead>Original</TableHead>
              <TableHead>Payable</TableHead>
              <TableHead>Paid</TableHead>
              <TableHead>Refund due</TableHead>
              <TableHead>Settlement</TableHead>
              <TableHead />
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => (
              <TableRow key={row.order_id}>
                <TableCell className="font-medium">{row.order_number}</TableCell>
                <TableCell>
                  <div>{row.customer_name}</div>
                  <div className="text-xs text-foreground-secondary">{row.customer_email}</div>
                </TableCell>
                <TableCell>{formatIdr(row.original_total_idr)}</TableCell>
                <TableCell>{formatIdr(row.payable_idr)}</TableCell>
                <TableCell>{formatIdr(row.paid_idr)}</TableCell>
                <TableCell>{formatIdr(row.refund_entitled_idr)}</TableCell>
                <TableCell>
                  <StatusBadge status={row.settlement_status} />
                </TableCell>
                <TableCell>
                  {row.refund_entitled_idr > 0 ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      onClick={() => {
                        setSelected(row);
                        setRefundAmount(String(row.refund_entitled_idr));
                      }}
                    >
                      Refund
                    </Button>
                  ) : null}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      {selected ? (
        <div className="max-w-md space-y-3 rounded-xl border border-border bg-background p-4 shadow-sm">
          <h3 className="text-base font-semibold text-foreground">
            Process refund — {selected.order_number}
          </h3>
          <p className="text-sm text-foreground-secondary">
            Entitled {formatIdr(selected.refund_entitled_idr)} after unavailable / credit
            adjustments.
          </p>
          <Input
            type="number"
            value={refundAmount}
            onChange={(e) => setRefundAmount(e.target.value)}
            aria-label="Refund amount"
          />
          <div className="flex gap-2">
            <Button type="button" onClick={() => void createAndCompleteRefund()} disabled={busy}>
              Complete bank refund
            </Button>
            <Button type="button" variant="ghost" onClick={() => setSelected(null)} disabled={busy}>
              Cancel
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
