"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";

export function CustomerNotesForm({
  storeId,
  customerId,
  initialNotes,
}: {
  storeId: string;
  customerId: string;
  initialNotes: string;
}) {
  const router = useRouter();
  const [notes, setNotes] = useState(initialNotes);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  async function onSave() {
    setBusy(true);
    setError(null);
    setSaved(false);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/customers/${customerId}`, {
        method: "PATCH",
        body: { internal_notes: notes },
      });
      setSaved(true);
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to save notes.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-3">
      <textarea
        className="min-h-28 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
        value={notes}
        onChange={(e) => setNotes(e.target.value)}
        placeholder="Add an internal note about this customer"
      />
      {error ? <p className="text-sm text-error">{error}</p> : null}
      {saved ? <p className="text-sm text-success">Notes saved.</p> : null}
      <Button type="button" onClick={onSave} disabled={busy}>
        {busy ? "Saving…" : "Save notes"}
      </Button>
    </div>
  );
}
