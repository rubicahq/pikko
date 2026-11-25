"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

export function ImportUpload({
  storeId,
  source,
}: {
  storeId: string;
  /** PDF = Add Catalog flipbook pages; CSV = Products import. */
  source: "pdf" | "csv";
}) {
  const router = useRouter();
  const [catalogName, setCatalogName] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const nameReady = catalogName.trim().length > 0;

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (source === "pdf" && !nameReady) {
      setError("Enter a catalog name before uploading the PDF.");
      return;
    }
    if (!file) {
      setError(source === "csv" ? "Choose a CSV file." : "Choose a PDF catalog file.");
      return;
    }
    if (
      source === "pdf" &&
      file.type &&
      !file.type.includes("pdf") &&
      !file.name.toLowerCase().endsWith(".pdf")
    ) {
      setError("File must be a PDF.");
      return;
    }
    if (
      source === "csv" &&
      !file.name.toLowerCase().endsWith(".csv") &&
      file.type &&
      !file.type.includes("csv") &&
      !file.type.includes("text")
    ) {
      setError("File must be a CSV. Download the template if you are unsure of the format.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const fd = new FormData();
      fd.append("file", file);
      fd.append("source", source);
      if (source === "pdf") {
        fd.append("catalog_name", catalogName.trim());
        fd.append("image_mode", "none");
      }
      const res = await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-imports`), {
        method: "POST",
        credentials: "include",
        headers: {
          "Idempotency-Key": `web-${source}-${storeId}-${file.name}-${file.size}-${file.lastModified}`,
        },
        body: fd,
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data?.error?.message || "Upload failed");
      }
      const jobPath =
        source === "pdf"
          ? urls.catalogsImportJob(storeId, data.job.id)
          : urls.productsImportJob(storeId, data.job.id);
      router.push(jobPath);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form
      onSubmit={onSubmit}
      className="max-w-xl space-y-5 rounded-lg border border-border bg-background p-5"
    >
      <div>
        <h2 className="text-lg font-semibold text-foreground">
          {source === "pdf" ? "Upload PDF catalog" : "Import CSV"}
        </h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          {source === "pdf"
            ? "Name the catalog first, then upload the PDF. We split it into flipbook pages — display only, no AI product detection."
            : "Use the CSV template. Results are drafts — nothing publishes automatically."}
        </p>
      </div>

      {source === "pdf" ? (
        <>
          <div className="space-y-2">
            <Label htmlFor="catalog-name">Catalog name</Label>
            <Input
              id="catalog-name"
              value={catalogName}
              onChange={(e) => {
                setCatalogName(e.target.value);
                if (error) setError(null);
              }}
              placeholder="e.g. Summer Drop Lookbook"
              maxLength={200}
              required
              autoComplete="off"
            />
            <p className="text-xs text-foreground-muted">
              Required. Upload stays locked until you set a name.
            </p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="pdf">PDF file</Label>
            <input
              id="pdf"
              type="file"
              accept="application/pdf,.pdf"
              disabled={!nameReady}
              className="block w-full text-sm text-foreground file:mr-3 file:rounded-md file:border file:border-border file:bg-surface file:px-3 file:py-2 disabled:cursor-not-allowed disabled:opacity-50"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
            />
            <p className="text-xs text-foreground-muted">
              Max 25 MB, 50 pages. Password-protected PDFs are not supported.
            </p>
          </div>
        </>
      ) : (
        <div className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="csv">CSV file</Label>
            <input
              id="csv"
              type="file"
              accept=".csv,text/csv,text/plain"
              className="block w-full text-sm text-foreground file:mr-3 file:rounded-md file:border file:border-border file:bg-surface file:px-3 file:py-2"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
            />
            <p className="text-xs text-foreground-muted">
              Required columns: <code className="text-foreground">name</code>,{" "}
              <code className="text-foreground">price_idr</code> (whole rupiah, no decimals). Max 5
              MB / 500 rows. Use comma separators.
            </p>
          </div>
          <a
            className="inline-block text-sm text-brand-dark underline-offset-2 hover:underline"
            href={urls.api(`/api/v1/stores/${storeId}/catalog-imports/csv-template`)}
          >
            Download CSV template
          </a>
        </div>
      )}

      {error && (
        <p className="rounded-md border border-error-border bg-error-bg px-3 py-2 text-sm text-error">
          {error}
        </p>
      )}
      <Button type="submit" disabled={busy || (source === "pdf" && !nameReady)}>
        {busy ? "Uploading…" : source === "pdf" ? "Create flipbook" : "Start import"}
      </Button>
    </form>
  );
}
