"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

type CampaignOption = {
  id: string;
  name: string;
  campaign_type: string;
  operational_status?: string;
  preorder_batch_id?: string | null;
};

type BatchOption = {
  id: string;
  name: string;
  operational_status: string;
};

export function OrderRelationsEditor({
  storeId,
  orderId,
  campaignId,
  preorderBatchId,
  batchName,
}: {
  storeId: string;
  orderId: string;
  campaignId?: string | null;
  preorderBatchId?: string | null;
  batchName?: string | null;
}) {
  const router = useRouter();
  const [campaigns, setCampaigns] = useState<CampaignOption[]>([]);
  const [batches, setBatches] = useState<BatchOption[]>([]);
  const [selectedCampaign, setSelectedCampaign] = useState(campaignId || "");
  const [selectedBatch, setSelectedBatch] = useState(preorderBatchId || "");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    setSelectedCampaign(campaignId || "");
    setSelectedBatch(preorderBatchId || "");
  }, [campaignId, preorderBatchId]);

  useEffect(() => {
    void Promise.all([
      apiFetch<{ campaigns: CampaignOption[] }>(`/api/v1/stores/${storeId}/campaigns`),
      apiFetch<{ batches: BatchOption[] }>(`/api/v1/stores/${storeId}/preorders`),
    ])
      .then(([campRes, batchRes]) => {
        setCampaigns(campRes.campaigns || []);
        setBatches(batchRes.batches || []);
      })
      .catch(() => undefined);
  }, [storeId]);

  async function save() {
    setBusy(true);
    setError(null);
    setSaved(false);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/orders/${orderId}`, {
        method: "PATCH",
        body: {
          campaign_id: selectedCampaign,
          preorder_batch_id: selectedBatch,
        },
      });
      setSaved(true);
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update relations.");
    } finally {
      setBusy(false);
    }
  }

  const campaignLabel = campaigns.find((c) => c.id === (campaignId || ""))?.name;
  const batchLabel =
    batchName || batches.find((b) => b.id === (preorderBatchId || ""))?.name;

  return (
    <section className="rounded-[10px] border border-border p-5">
      <h2 className="text-base font-medium">Campaign & preorder</h2>
      <p className="mt-1 text-sm text-foreground-secondary">
        Optional links for ops. Current:{" "}
        {campaignLabel || batchLabel
          ? [campaignLabel, batchLabel].filter(Boolean).join(" · ")
          : "none"}
      </p>
      <div className="mt-4 grid gap-4 sm:grid-cols-2">
        <div>
          <Label htmlFor="order-campaign">Campaign</Label>
          <select
            id="order-campaign"
            className="mt-1.5 flex h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={selectedCampaign}
            onChange={(e) => {
              const next = e.target.value;
              setSelectedCampaign(next);
              const camp = campaigns.find((c) => c.id === next);
              if (camp?.preorder_batch_id) setSelectedBatch(camp.preorder_batch_id);
            }}
          >
            <option value="">No campaign</option>
            {campaigns.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
                {c.campaign_type ? ` · ${c.campaign_type}` : ""}
              </option>
            ))}
          </select>
        </div>
        <div>
          <Label htmlFor="order-batch">Preorder batch</Label>
          <select
            id="order-batch"
            className="mt-1.5 flex h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={selectedBatch}
            onChange={(e) => setSelectedBatch(e.target.value)}
          >
            <option value="">No preorder</option>
            {batches.map((b) => (
              <option key={b.id} value={b.id}>
                {b.name} · {b.operational_status}
              </option>
            ))}
          </select>
        </div>
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <Button type="button" variant="secondary" disabled={busy} onClick={() => void save()}>
          Save relations
        </Button>
        {saved ? (
          <span className="text-caption text-foreground-muted">Saved</span>
        ) : null}
      </div>
      {error ? <p className="mt-2 text-sm text-error">{error}</p> : null}
    </section>
  );
}
