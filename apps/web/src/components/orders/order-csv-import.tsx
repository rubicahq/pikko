"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiUpload, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

type PreviewRow = {
  row_number: number;
  customer_name: string;
  customer_email: string;
  product_sku: string;
  quantity: number;
  product_name?: string;
  unit_price_idr?: number;
  matched_product_id?: string;
  matched_customer_id?: string;
  errors?: string[];
  warnings?: string[];
};

type Preview = {
  headers: string[];
  rows: PreviewRow[];
  valid_count: number;
  invalid_count: number;
};

export function OrderCsvImport({ storeId }: { storeId: string }) {
  const router = useRouter();
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<{ created: number; failed: number } | null>(null);

  async function runPreview() {
    if (!file) return;
    setBusy(true);
    setError(null);
    setResult(null);
    try {
      const form = new FormData();
      form.append("file", file);
      const res = await apiUpload<Preview>(`/api/v1/stores/${storeId}/order-entry/csv-preview`, form);
      setPreview(res);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Preview failed");
    } finally {
      setBusy(false);
    }
  }

  async function runImport() {
    if (!file) return;
    setBusy(true);
    setError(null);
    try {
      const form = new FormData();
      form.append("file", file);
      form.append("idempotency_key", crypto.randomUUID());
      const res = await apiUpload<{ created: number; failed: number; order_ids?: string[] }>(
        `/api/v1/stores/${storeId}/order-entry/csv-import`,
        form,
      );
      setResult({ created: res.created, failed: res.failed });
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Import failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end gap-3">
        <div>
          <Label htmlFor="csv-file">CSV file</Label>
          <input
            id="csv-file"
            type="file"
            accept=".csv,text/csv"
            className="mt-1.5 block text-sm"
            onChange={(e) => {
              setFile(e.target.files?.[0] || null);
              setPreview(null);
              setResult(null);
            }}
          />
        </div>
        <Button type="button" variant="secondary" disabled={!file || busy} onClick={() => void runPreview()}>
          Preview & match
        </Button>
        <a
          className="text-sm text-brand-accessible-dark underline"
          href={`/api/v1/stores/${storeId}/order-entry/csv-template`}
        >
          Download template
        </a>
      </div>

      {preview ? (
        <div className="space-y-3">
          <p className="text-sm text-foreground-secondary">
            {preview.valid_count} valid · {preview.invalid_count} with errors
          </p>
          <div className="overflow-x-auto rounded-md border border-border">
            <table className="w-full min-w-[720px] border-collapse text-sm">
              <thead className="bg-background-subtle text-left text-foreground-secondary">
                <tr>
                  <th className="px-3 py-2 font-medium">Row</th>
                  <th className="px-3 py-2 font-medium">Customer</th>
                  <th className="px-3 py-2 font-medium">SKU</th>
                  <th className="px-3 py-2 font-medium">Match</th>
                  <th className="px-3 py-2 text-right font-medium">Qty</th>
                  <th className="px-3 py-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {preview.rows.map((row) => (
                  <tr key={row.row_number} className="border-t border-border">
                    <td className="px-3 py-2 tabular-nums">{row.row_number}</td>
                    <td className="px-3 py-2">
                      <div>{row.customer_name}</div>
                      <div className="text-caption text-foreground-muted">{row.customer_email}</div>
                    </td>
                    <td className="px-3 py-2 font-mono text-xs">{row.product_sku}</td>
                    <td className="px-3 py-2">
                      {row.product_name || "—"}
                      {row.unit_price_idr ? (
                        <span className="text-foreground-muted"> · {formatIDR(row.unit_price_idr)}</span>
                      ) : null}
                    </td>
                    <td className="px-3 py-2 text-right tabular-nums">{row.quantity}</td>
                    <td className="px-3 py-2">
                      {row.errors?.length ? (
                        <span className="text-error">{row.errors.join("; ")}</span>
                      ) : (
                        <span className="text-success">OK{row.warnings?.length ? ` · ${row.warnings.join("; ")}` : ""}</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Button
            type="button"
            disabled={busy || preview.valid_count === 0}
            onClick={() => void runImport()}
          >
            Import {preview.valid_count} valid row group(s)
          </Button>
        </div>
      ) : null}

      {result ? (
        <p className="text-sm">
          Imported {result.created} order(s)
          {result.failed ? ` · ${result.failed} row(s) skipped` : ""}.{" "}
          <button
            type="button"
            className="text-brand-accessible-dark underline"
            onClick={() => router.push(`/stores/${storeId}/orders`)}
          >
            Back to orders
          </button>
        </p>
      ) : null}
      {error ? <p className="text-sm text-error">{error}</p> : null}
    </div>
  );
}
