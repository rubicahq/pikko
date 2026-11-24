"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

export function ImportUpload({ storeId }: { storeId: string }) {
  const router = useRouter();
  const [mode, setMode] = useState<"pdf" | "csv">("pdf");
  const [file, setFile] = useState<File | null>(null);
  const [extractImages, setExtractImages] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file) {
      setError(mode === "csv" ? "Choose a CSV file." : "Choose a PDF catalog file.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const fd = new FormData();
      fd.append("file", file);
      fd.append("source", mode);
      if (mode === "pdf") {
        fd.append("extract_images", String(extractImages));
      }
      const res = await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-imports`), {
        method: "POST",
        credentials: "include",
        headers: {
          "Idempotency-Key": `web-${mode}-${storeId}-${file.name}-${file.size}-${file.lastModified}`,
        },
        body: fd,
      });
      const data = await res.json();
      if (!res.ok) {
        throw new Error(data?.error?.message || "Upload failed");
      }
      router.push(urls.catalogImportJob(storeId, data.job.id));
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Upload failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="max-w-xl space-y-5 rounded-lg border border-border bg-background p-5">
      <div>
        <h2 className="text-lg font-medium text-foreground">Import catalog</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Import from PDF (AI) or CSV template. Results are drafts — nothing publishes automatically.
        </p>
      </div>

      <div className="flex gap-2 border-b border-border text-sm">
        {(
          [
            { id: "pdf" as const, label: "PDF" },
            { id: "csv" as const, label: "CSV" },
          ] as const
        ).map((m) => (
          <button
            key={m.id}
            type="button"
            className={`pb-2 ${mode === m.id ? "border-b-2 border-foreground font-medium" : "text-foreground-secondary"}`}
            onClick={() => {
              setMode(m.id);
              setFile(null);
              setError(null);
            }}
          >
            {m.label}
          </button>
        ))}
      </div>

      {mode === "pdf" ? (
        <>
          <div className="space-y-2">
            <Label htmlFor="pdf">PDF file</Label>
            <input
              id="pdf"
              type="file"
              accept="application/pdf,.pdf"
              className="block w-full text-sm text-foreground file:mr-3 file:rounded-md file:border file:border-border file:bg-surface file:px-3 file:py-2"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
            />
            <p className="text-xs text-foreground-muted">Max 25 MB, 50 pages, 500 items.</p>
          </div>
          <label className="flex items-center gap-2 text-sm text-foreground">
            <input
              type="checkbox"
              checked={extractImages}
              onChange={(e) => setExtractImages(e.target.checked)}
              className="h-4 w-4 rounded border-border"
            />
            Extract product images when possible
          </label>
        </>
      ) : (
        <div className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="csv">CSV file</Label>
            <input
              id="csv"
              type="file"
              accept=".csv,text/csv"
              className="block w-full text-sm text-foreground file:mr-3 file:rounded-md file:border file:border-border file:bg-surface file:px-3 file:py-2"
              onChange={(e) => setFile(e.target.files?.[0] || null)}
            />
            <p className="text-xs text-foreground-muted">
              Required columns: name, price_idr. Max 5 MB / 500 rows.
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

      {error && <p className="text-sm text-error">{error}</p>}
      <Button type="submit" disabled={busy}>
        {busy ? "Uploading…" : "Start import"}
      </Button>
    </form>
  );
}
