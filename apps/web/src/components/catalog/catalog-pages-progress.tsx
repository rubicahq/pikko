"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";

type Job = {
  id: string;
  status: string;
  progress_pct: number;
  progress_stage?: string;
  progress_page?: number | null;
  progress_pages?: number | null;
  page_count?: number | null;
  catalog_id?: string | null;
  error_message?: string | null;
  source_type?: string;
};

function stageLabel(job: Job): string {
  const stage = job.progress_stage || "";
  switch (stage) {
    case "queued":
      return "Queued — waiting to start";
    case "uploading":
      return "Upload received — preparing PDF";
    case "splitting_pages":
      return job.progress_pages
        ? `Splitting into ${job.progress_pages} page${job.progress_pages === 1 ? "" : "s"}`
        : "Splitting pages";
    case "completed":
      return "Pages ready";
    case "failed":
      return "Failed";
    default:
      return `Progress ${job.progress_pct}%`;
  }
}

/** Polls PDF page-render job and routes to flipbook when complete. AI review is soft-deprecated. */
export function CatalogPagesProgress({
  storeId,
  jobId,
}: {
  storeId: string;
  jobId: string;
}) {
  const router = useRouter();
  const [job, setJob] = useState<Job | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const res = await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}`), {
      credentials: "include",
    });
    if (!res.ok) {
      setError("Unable to load catalog job.");
      return;
    }
    const data = await res.json();
    const next: Job = data.job;
    setJob(next);
    if (next.status === "completed") {
      if (next.catalog_id) {
        router.replace(urls.storeCatalogDoc(storeId, next.catalog_id));
      } else {
        router.replace(urls.catalogsFlipbook(storeId));
      }
      router.refresh();
    }
  }, [storeId, jobId, router]);

  useEffect(() => {
    load();
    const id = window.setInterval(() => {
      if (job?.status === "completed" || job?.status === "failed") return;
      load();
    }, 1500);
    return () => window.clearInterval(id);
  }, [load, job?.status]);

  async function retry() {
    setError(null);
    const res = await fetch(
      urls.api(`/api/v1/stores/${storeId}/catalog-imports/${jobId}/retry`),
      { method: "POST", credentials: "include" },
    );
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      setError(data?.error?.message || "Unable to retry.");
      return;
    }
    await load();
  }

  return (
    <div className="max-w-xl space-y-4 rounded-lg border border-border bg-background p-5">
      <div>
        <h2 className="text-lg font-semibold text-foreground">Preparing flipbook</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Splitting the PDF into pages for your display catalog. Product AI detection is no longer
          used.
        </p>
      </div>

      {error && (
        <p className="rounded-md border border-error-border bg-error-bg px-3 py-2 text-sm text-error">
          {error}
        </p>
      )}

      {job && (
        <div className="space-y-2">
          <div className="flex items-center justify-between text-sm">
            <span className="text-foreground-secondary">{stageLabel(job)}</span>
            <span className="tabular-nums text-foreground-muted">{job.progress_pct}%</span>
          </div>
          <div className="h-2 overflow-hidden rounded-full bg-surface">
            <div
              className="h-full bg-brand transition-all"
              style={{ width: `${Math.min(100, Math.max(0, job.progress_pct))}%` }}
            />
          </div>
          {job.status === "failed" && (
            <div className="space-y-3 pt-2">
              <p className="text-sm text-error">
                {job.error_message || "Page rendering failed."}
              </p>
              <Button type="button" onClick={retry}>
                Retry
              </Button>
            </div>
          )}
          {job.status === "completed" && (
            <p className="text-sm text-success">Redirecting to catalog…</p>
          )}
        </div>
      )}

      <p className="text-xs text-foreground-muted">
        AI product review for PDF catalogs is retired.{" "}
        <Link href={urls.catalogsFlipbook(storeId)} className="underline-offset-2 hover:underline">
          Open flipbook editor
        </Link>
      </p>
    </div>
  );
}
