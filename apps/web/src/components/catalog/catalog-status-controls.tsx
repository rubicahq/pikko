"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function CatalogStatusControls({
  storeId,
  catalogId,
  status,
}: {
  storeId: string;
  catalogId: string;
  status: string;
}) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function setStatus(next: "draft" | "published" | "archived") {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}`, {
        method: "PATCH",
        body: { status: next },
      });
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to update status.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="rounded-[10px] border border-border p-5">
      <h3 className="text-base font-medium">Publication</h3>
      <p className="mt-1 text-sm text-foreground-secondary">
        Published catalogs appear as 16:9 banners on the storefront (max 8 on the homepage).
      </p>
      <div className="mt-4 flex flex-wrap gap-2">
        {status !== "published" ? (
          <Button type="button" disabled={busy} onClick={() => setStatus("published")}>
            Publish
          </Button>
        ) : (
          <Button type="button" variant="outline" disabled={busy} onClick={() => setStatus("draft")}>
            Unpublish
          </Button>
        )}
        {status !== "archived" ? (
          <Button type="button" variant="ghost" disabled={busy} onClick={() => setStatus("archived")}>
            Archive
          </Button>
        ) : null}
      </div>
      {error ? <p className="mt-2 text-sm text-error">{error}</p> : null}
    </section>
  );
}
