"use client";

import { useCallback, useEffect, useState } from "react";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

type CatalogDoc = {
  id: string;
  title: string;
  status: string;
  page_count: number;
  campaign_ids?: string[];
  preorder_batch_ids?: string[];
};

export function CatalogAttach({
  storeId,
  mode,
  targetId,
}: {
  storeId: string;
  mode: "campaign" | "preorder";
  targetId: string;
}) {
  const [linked, setLinked] = useState<CatalogDoc[]>([]);
  const [all, setAll] = useState<CatalogDoc[]>([]);
  const [catalogId, setCatalogId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const listPath =
    mode === "campaign"
      ? `/api/v1/stores/${storeId}/campaigns/${targetId}/catalogs`
      : `/api/v1/stores/${storeId}/preorders/${targetId}/catalogs`;

  const load = useCallback(async () => {
    try {
      const [linkedRes, allRes] = await Promise.all([
        apiFetch<{ catalogs: CatalogDoc[] }>(listPath),
        apiFetch<{ catalogs: CatalogDoc[] }>(`/api/v1/stores/${storeId}/catalogs`),
      ]);
      setLinked(linkedRes.catalogs || []);
      setAll(allRes.catalogs || []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to load catalogs.");
    }
  }, [listPath, storeId]);

  useEffect(() => {
    load();
  }, [load]);

  const candidates = all.filter((c) => !linked.some((l) => l.id === c.id));

  async function attach(e: React.FormEvent) {
    e.preventDefault();
    if (!catalogId) return;
    setBusy(true);
    setError(null);
    try {
      if (mode === "campaign") {
        await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}/campaigns`, {
          method: "POST",
          body: { campaign_id: targetId },
        });
      } else {
        await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}/preorders`, {
          method: "POST",
          body: { batch_id: targetId },
        });
      }
      setCatalogId("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to attach catalog.");
    } finally {
      setBusy(false);
    }
  }

  async function detach(id: string) {
    setBusy(true);
    setError(null);
    try {
      if (mode === "campaign") {
        await apiFetch(`/api/v1/stores/${storeId}/catalogs/${id}/campaigns/${targetId}`, {
          method: "DELETE",
        });
      } else {
        await apiFetch(`/api/v1/stores/${storeId}/catalogs/${id}/preorders/${targetId}`, {
          method: "DELETE",
        });
      }
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to detach catalog.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="space-y-4 rounded-lg border border-border bg-background p-5">
      <div>
        <h2 className="text-base font-semibold text-foreground">PDF catalog</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Optional — attach a PDF catalog to this {mode === "campaign" ? "campaign" : "preorder"}, or
          leave catalogs standalone.
        </p>
      </div>
      {error && <p className="text-sm text-error">{error}</p>}
      <ul className="space-y-2">
        {linked.length === 0 && (
          <li className="text-sm text-foreground-muted">No catalog attached.</li>
        )}
        {linked.map((c) => (
          <li
            key={c.id}
            className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border px-3 py-2 text-sm"
          >
            <span>
              {c.title} · {c.page_count} pages · {c.status}
            </span>
            <Button type="button" size="sm" variant="ghost" disabled={busy} onClick={() => detach(c.id)}>
              Detach
            </Button>
          </li>
        ))}
      </ul>
      <form onSubmit={attach} className="flex flex-wrap items-end gap-3">
        <div className="min-w-[220px] flex-1">
          <Label htmlFor="attach-catalog">Catalog</Label>
          <select
            id="attach-catalog"
            className="mt-1 w-full rounded-[10px] border border-border bg-background px-3 py-2 text-sm"
            value={catalogId}
            onChange={(e) => setCatalogId(e.target.value)}
          >
            <option value="">Select catalog…</option>
            {candidates.map((c) => (
              <option key={c.id} value={c.id}>
                {c.title} ({c.page_count} pages)
              </option>
            ))}
          </select>
        </div>
        <Button type="submit" disabled={busy || !catalogId}>
          Attach
        </Button>
      </form>
    </section>
  );
}
