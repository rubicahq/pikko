"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function RotatePortalLinkButton({
  storeId,
  customerId,
}: {
  storeId: string;
  customerId: string;
}) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function rotate() {
    if (
      !window.confirm(
        "Invalidate the current portal link and mint a new one? The old URL will stop working.",
      )
    ) {
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/customers/${customerId}/portal-access/rotate`, {
        method: "POST",
      });
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to rotate portal link.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-1">
      <Button type="button" variant="outline" size="sm" disabled={busy} onClick={() => void rotate()}>
        {busy ? "Rotating…" : "Invalidate & new link"}
      </Button>
      {error ? <p className="text-sm text-error">{error}</p> : null}
    </div>
  );
}
