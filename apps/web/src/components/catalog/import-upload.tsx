"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

export function ImportUpload({ storeId }: { storeId: string }) {
  const router = useRouter();
  const [file, setFile] = useState<File | null>(null);
  const [extractImages, setExtractImages] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!file) {
      setError("Choose a PDF catalog file.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const fd = new FormData();
      fd.append("file", file);
      fd.append("extract_images", String(extractImages));
      const res = await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-imports`), {
        method: "POST",
        credentials: "include",
        headers: {
          "Idempotency-Key": `web-${storeId}-${file.name}-${file.size}-${file.lastModified}`,
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
        <h2 className="text-lg font-medium text-foreground">Import PDF</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          One PDF per job. Max 25 MB, 50 pages, 500 items. Results are drafts — nothing publishes
          automatically.
        </p>
      </div>
      <div className="space-y-2">
        <Label htmlFor="pdf">PDF file</Label>
        <input
          id="pdf"
          type="file"
          accept="application/pdf,.pdf"
          className="block w-full text-sm text-foreground file:mr-3 file:rounded-md file:border file:border-border file:bg-surface file:px-3 file:py-2"
          onChange={(e) => setFile(e.target.files?.[0] || null)}
        />
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
      {error && <p className="text-sm text-error">{error}</p>}
      <Button type="submit" disabled={busy}>
        {busy ? "Uploading…" : "Start import"}
      </Button>
    </form>
  );
}
