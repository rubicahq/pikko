"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

type Opt = { id: string; name: string };

export function CatalogDetailLinks({
  storeId,
  catalogId,
  campaignIds,
  preorderBatchIds,
}: {
  storeId: string;
  catalogId: string;
  campaignIds: string[];
  preorderBatchIds: string[];
}) {
  const [campaigns, setCampaigns] = useState<Opt[]>([]);
  const [batches, setBatches] = useState<Opt[]>([]);
  const [campaignId, setCampaignId] = useState("");
  const [batchId, setBatchId] = useState("");
  const [linkedCampaigns, setLinkedCampaigns] = useState(campaignIds);
  const [linkedBatches, setLinkedBatches] = useState(preorderBatchIds);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const loadOptions = useCallback(async () => {
    try {
      const [cRes, bRes] = await Promise.all([
        apiFetch<{ campaigns: { id: string; name: string }[] }>(
          `/api/v1/stores/${storeId}/campaigns`,
        ),
        apiFetch<{ batches: { id: string; name: string }[] }>(
          `/api/v1/stores/${storeId}/preorders`,
        ),
      ]);
      setCampaigns((cRes.campaigns || []).map((c) => ({ id: c.id, name: c.name })));
      setBatches((bRes.batches || []).map((b) => ({ id: b.id, name: b.name })));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to load campaigns/preorders.");
    }
  }, [storeId]);

  useEffect(() => {
    loadOptions();
  }, [loadOptions]);

  async function attachCampaign(e: React.FormEvent) {
    e.preventDefault();
    if (!campaignId) return;
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}/campaigns`, {
        method: "POST",
        body: { campaign_id: campaignId },
      });
      setLinkedCampaigns((prev) => (prev.includes(campaignId) ? prev : [...prev, campaignId]));
      setCampaignId("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to attach campaign.");
    } finally {
      setBusy(false);
    }
  }

  async function attachPreorder(e: React.FormEvent) {
    e.preventDefault();
    if (!batchId) return;
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}/preorders`, {
        method: "POST",
        body: { batch_id: batchId },
      });
      setLinkedBatches((prev) => (prev.includes(batchId) ? prev : [...prev, batchId]));
      setBatchId("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to attach preorder.");
    } finally {
      setBusy(false);
    }
  }

  async function detachCampaign(id: string) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}/campaigns/${id}`, {
        method: "DELETE",
      });
      setLinkedCampaigns((prev) => prev.filter((x) => x !== id));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to detach campaign.");
    } finally {
      setBusy(false);
    }
  }

  async function detachPreorder(id: string) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/catalogs/${catalogId}/preorders/${id}`, {
        method: "DELETE",
      });
      setLinkedBatches((prev) => prev.filter((x) => x !== id));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to detach preorder.");
    } finally {
      setBusy(false);
    }
  }

  const campaignName = (id: string) => campaigns.find((c) => c.id === id)?.name || id;
  const batchName = (id: string) => batches.find((b) => b.id === id)?.name || id;
  const freeCampaigns = campaigns.filter((c) => !linkedCampaigns.includes(c.id));
  const freeBatches = batches.filter((b) => !linkedBatches.includes(b.id));

  return (
    <section className="space-y-4 rounded-lg border border-border bg-background p-5">
      <div>
        <h2 className="text-base font-semibold text-foreground">Campaign &amp; preorder links</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Optional — attach this catalog to a campaign, a preorder, both, or neither.
        </p>
      </div>
      {error && <p className="text-sm text-error">{error}</p>}

      <div className="space-y-2">
        <p className="text-sm font-medium text-foreground">Campaigns</p>
        {linkedCampaigns.length === 0 ? (
          <p className="text-sm text-foreground-muted">Not linked to any campaign.</p>
        ) : (
          <ul className="space-y-2">
            {linkedCampaigns.map((id) => (
              <li
                key={id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border px-3 py-2 text-sm"
              >
                <Link
                  href={urls.storeCampaign(storeId, id)}
                  className="text-foreground hover:underline"
                >
                  {campaignName(id)}
                </Link>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  disabled={busy}
                  onClick={() => detachCampaign(id)}
                >
                  Detach
                </Button>
              </li>
            ))}
          </ul>
        )}
        <form onSubmit={attachCampaign} className="flex flex-wrap items-end gap-3">
          <div className="min-w-[200px] flex-1">
            <Label htmlFor="attach-campaign">Add campaign</Label>
            <select
              id="attach-campaign"
              className="mt-1 w-full rounded-[10px] border border-border bg-background px-3 py-2 text-sm"
              value={campaignId}
              onChange={(e) => setCampaignId(e.target.value)}
            >
              <option value="">Select…</option>
              {freeCampaigns.map((c) => (
                <option key={c.id} value={c.id}>
                  {c.name}
                </option>
              ))}
            </select>
          </div>
          <Button type="submit" disabled={busy || !campaignId}>
            Attach
          </Button>
        </form>
      </div>

      <div className="space-y-2">
        <p className="text-sm font-medium text-foreground">Preorders</p>
        {linkedBatches.length === 0 ? (
          <p className="text-sm text-foreground-muted">Not linked to any preorder.</p>
        ) : (
          <ul className="space-y-2">
            {linkedBatches.map((id) => (
              <li
                key={id}
                className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border px-3 py-2 text-sm"
              >
                <Link
                  href={urls.storePreorder(storeId, id)}
                  className="text-foreground hover:underline"
                >
                  {batchName(id)}
                </Link>
                <Button
                  type="button"
                  size="sm"
                  variant="ghost"
                  disabled={busy}
                  onClick={() => detachPreorder(id)}
                >
                  Detach
                </Button>
              </li>
            ))}
          </ul>
        )}
        <form onSubmit={attachPreorder} className="flex flex-wrap items-end gap-3">
          <div className="min-w-[200px] flex-1">
            <Label htmlFor="attach-preorder">Add preorder</Label>
            <select
              id="attach-preorder"
              className="mt-1 w-full rounded-[10px] border border-border bg-background px-3 py-2 text-sm"
              value={batchId}
              onChange={(e) => setBatchId(e.target.value)}
            >
              <option value="">Select…</option>
              {freeBatches.map((b) => (
                <option key={b.id} value={b.id}>
                  {b.name}
                </option>
              ))}
            </select>
          </div>
          <Button type="submit" disabled={busy || !batchId}>
            Attach
          </Button>
        </form>
      </div>
    </section>
  );
}
