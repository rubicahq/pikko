"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, ApiError } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type ReceivingLine = {
  id: string;
  product_name?: string | null;
  variant_label?: string | null;
  expected_qty: number;
  received_qty: number;
  discrepancy_notes?: string | null;
};

type Receiving = {
  id: string;
  status: string;
  batch_name?: string | null;
  vendor_name?: string | null;
  notes?: string | null;
  lines?: ReceivingLine[];
};

export function ReceivingDetail({
  storeId,
  receivingId,
}: {
  storeId: string;
  receivingId: string;
}) {
  const [rec, setRec] = useState<Receiving | null>(null);
  const [drafts, setDrafts] = useState<Record<string, string>>({});
  const [notes, setNotes] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await apiFetch<{ receiving: Receiving }>(
        `/api/v1/stores/${storeId}/receiving/${receivingId}`,
      );
      setRec(data.receiving);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load receiving.");
    }
  }, [storeId, receivingId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveLine(line: ReceivingLine) {
    const qty = Number(drafts[line.id] ?? line.received_qty);
    if (!Number.isFinite(qty) || qty < 0) {
      setError("Received qty must be non-negative.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const data = await apiFetch<{ receiving: Receiving }>(
        `/api/v1/stores/${storeId}/receiving/${receivingId}/lines/${line.id}`,
        {
          method: "PATCH",
          body: {
            received_qty: qty,
            discrepancy_notes: notes[line.id] ?? line.discrepancy_notes ?? undefined,
          },
        },
      );
      setRec(data.receiving);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Update failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Receiving"
        description={
          rec
            ? rec.batch_name || "Batch"
            : "Record goods received against purchasing."
        }
        actions={
          <Button asChild variant="outline">
            <Link href={urls.storePurchasing(storeId)}>Back to purchasing</Link>
          </Button>
        }
      />

      {error ? (
        <p className="rounded-lg border border-error-border bg-error-bg px-4 py-3 text-sm text-error">
          {error}
        </p>
      ) : null}

      {rec ? (
        <div className="space-y-4">
          <div className="flex items-center gap-3">
            <StatusBadge status={rec.status} />
            {rec.notes ? (
              <p className="text-sm text-foreground-secondary">{rec.notes}</p>
            ) : null}
          </div>
          <Table stickyHeader>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead sticky>Product</TableHead>
                <TableHead sticky className="text-right">
                  Expected
                </TableHead>
                <TableHead sticky className="text-right">
                  Received
                </TableHead>
                <TableHead sticky>Discrepancy</TableHead>
                <TableHead sticky />
              </TableRow>
            </TableHeader>
            <TableBody>
              {(rec.lines || []).map((line) => (
                <TableRow key={line.id}>
                  <TableCell>
                    <div className="font-medium">{line.product_name || "Product"}</div>
                    <div className="text-xs text-foreground-muted">
                      {line.variant_label || "—"}
                    </div>
                  </TableCell>
                  <TableCell className="text-right tabular-nums">{line.expected_qty}</TableCell>
                  <TableCell className="text-right">
                    <Input
                      className="ml-auto h-9 w-20 text-right"
                      value={drafts[line.id] ?? String(line.received_qty)}
                      onChange={(e) =>
                        setDrafts((d) => ({ ...d, [line.id]: e.target.value }))
                      }
                    />
                  </TableCell>
                  <TableCell>
                    <Input
                      className="h-9"
                      placeholder="Optional note"
                      value={notes[line.id] ?? line.discrepancy_notes ?? ""}
                      onChange={(e) =>
                        setNotes((n) => ({ ...n, [line.id]: e.target.value }))
                      }
                    />
                  </TableCell>
                  <TableCell className="text-right">
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={busy}
                      onClick={() => void saveLine(line)}
                    >
                      Save
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      ) : null}
    </div>
  );
}
