"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function ArchiveStoreButton({
  storeId,
  storeName,
}: {
  storeId: string;
  storeName: string;
}) {
  const router = useRouter();
  const [confirming, setConfirming] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function archive() {
    setLoading(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/archive`, { method: "POST", body: {} });
      setConfirming(false);
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to archive.");
    } finally {
      setLoading(false);
    }
  }

  if (!confirming) {
    return (
      <Button size="sm" variant="ghost" onClick={() => setConfirming(true)}>
        Archive
      </Button>
    );
  }

  return (
    <div className="w-full rounded-md border border-warning-border bg-warning-bg p-3">
      <p className="text-sm text-warning">
        Archive <strong>{storeName}</strong>? New orders will be disabled and the storefront will
        leave public browsing. Historical records stay.
      </p>
      {error ? <p className="mt-2 text-sm text-error">{error}</p> : null}
      <div className="mt-3 flex gap-2">
        <Button size="sm" variant="destructive" loading={loading} onClick={archive}>
          Confirm archive
        </Button>
        <Button size="sm" variant="outline" onClick={() => setConfirming(false)}>
          Cancel
        </Button>
      </div>
    </div>
  );
}
