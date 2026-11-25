"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch, apiUpload } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { urls } from "@/lib/urls";

export function CatalogCoverEditor({
  storeId,
  catalogId,
  coverFileId,
}: {
  storeId: string;
  catalogId: string;
  coverFileId?: string | null;
}) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onUpload(file: File) {
    setBusy(true);
    setError(null);
    try {
      const form = new FormData();
      form.set("purpose", "catalog_cover");
      form.set("file", file);
      const uploaded = await apiUpload<{ file: { id: string } }>(
        `/api/v1/stores/${storeId}/files`,
        form,
      );
      await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}`, {
        method: "PATCH",
        body: { cover_file_id: uploaded.file.id },
      });
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Cover upload failed.");
    } finally {
      setBusy(false);
    }
  }

  async function onClear() {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}`, {
        method: "PATCH",
        body: { clear_cover: true },
      });
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to clear cover.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="rounded-[10px] border border-border p-5">
      <h3 className="text-base font-medium">Storefront cover</h3>
      <p className="mt-1 text-sm text-foreground-secondary">
        Optional 16:9 banner image shown on the storefront catalog cards.
      </p>
      {coverFileId ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={urls.file(storeId, coverFileId)}
          alt="Catalog cover"
          className="mt-4 aspect-video w-full max-w-md rounded-[10px] object-cover"
        />
      ) : (
        <div className="mt-4 flex aspect-video w-full max-w-md items-center justify-center rounded-[10px] border border-dashed border-border bg-background-subtle text-sm text-foreground-muted">
          No cover yet
        </div>
      )}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Input
          type="file"
          accept="image/png,image/jpeg,image/webp"
          disabled={busy}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void onUpload(f);
          }}
        />
        {coverFileId ? (
          <Button type="button" variant="outline" disabled={busy} onClick={onClear}>
            Remove cover
          </Button>
        ) : null}
      </div>
      {error ? <p className="mt-2 text-sm text-error">{error}</p> : null}
    </section>
  );
}
