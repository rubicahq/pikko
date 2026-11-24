"use client";

import { useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

function toLocalInput(d: Date) {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

function suggest(name: string) {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 48);
}

export function CampaignForm({ storeId }: { storeId: string }) {
  const router = useRouter();
  const search = useSearchParams();
  const now = new Date();
  const later = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
  const [type, setType] = useState<"preorder" | "live">(
    search.get("type") === "live" ? "live" : "preorder",
  );
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [opensAt, setOpensAt] = useState(toLocalInput(now));
  const [closesAt, setClosesAt] = useState(toLocalInput(later));
  const [etaStart, setEtaStart] = useState("");
  const [etaEnd, setEtaEnd] = useState("");
  const [publish, setPublish] = useState(true);
  const [quickAdd, setQuickAdd] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  function onName(v: string) {
    setName(v);
    if (!slug || slug === suggest(name)) setSlug(suggest(v));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const body: Record<string, unknown> = {
        campaign_type: type,
        name,
        slug,
        description: description || undefined,
        opens_at: new Date(opensAt).toISOString(),
        closes_at: new Date(closesAt).toISOString(),
        publication_status: publish ? "published" : "draft",
        payment_policy: "full_after_checkout",
      };
      if (type === "preorder") {
        if (etaStart) body.eta_start = etaStart;
        if (etaEnd) body.eta_end = etaEnd;
      } else {
        body.live_quick_add_enabled = quickAdd;
        body.live_show_order_feed = true;
      }
      const res = await apiFetch<{ campaign: { id: string } }>(
        `/api/v1/stores/${storeId}/campaigns`,
        { method: "POST", body },
      );
      router.push(urls.storeCampaign(storeId, res.campaign.id));
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not create campaign.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="mt-8 max-w-xl space-y-5">
      {error ? <p className="text-sm text-error">{error}</p> : null}

      <div className="space-y-2">
        <Label>Campaign type</Label>
        <div className="flex gap-4 text-sm">
          <label className="flex items-center gap-2">
            <input
              type="radio"
              name="type"
              checked={type === "preorder"}
              onChange={() => setType("preorder")}
            />
            Preorder
          </label>
          <label className="flex items-center gap-2">
            <input
              type="radio"
              name="type"
              checked={type === "live"}
              onChange={() => setType("live")}
            />
            Live shopping
          </label>
        </div>
      </div>

      <div className="space-y-2">
        <Label htmlFor="name">Name</Label>
        <Input id="name" value={name} onChange={(e) => onName(e.target.value)} required />
      </div>
      <div className="space-y-2">
        <Label htmlFor="slug">URL slug</Label>
        <Input id="slug" value={slug} onChange={(e) => setSlug(e.target.value)} required />
      </div>
      <div className="space-y-2">
        <Label htmlFor="description">Description</Label>
        <textarea
          id="description"
          className="min-h-[96px] w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="opens">Opens at</Label>
          <Input
            id="opens"
            type="datetime-local"
            value={opensAt}
            onChange={(e) => setOpensAt(e.target.value)}
            required
          />
        </div>
        <div className="space-y-2">
          <Label htmlFor="closes">Closes at</Label>
          <Input
            id="closes"
            type="datetime-local"
            value={closesAt}
            onChange={(e) => setClosesAt(e.target.value)}
            required
          />
        </div>
      </div>

      {type === "preorder" ? (
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="etaStart">ETA start</Label>
            <Input
              id="etaStart"
              type="date"
              value={etaStart}
              onChange={(e) => setEtaStart(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="etaEnd">ETA end</Label>
            <Input
              id="etaEnd"
              type="date"
              value={etaEnd}
              onChange={(e) => setEtaEnd(e.target.value)}
            />
          </div>
        </div>
      ) : (
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={quickAdd}
            onChange={(e) => setQuickAdd(e.target.checked)}
          />
          Enable merchant quick-add during live
        </label>
      )}

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={publish}
          onChange={(e) => setPublish(e.target.checked)}
        />
        Publish immediately
      </label>

      <Button type="submit" disabled={busy}>
        {busy ? "Creating…" : "Create campaign"}
      </Button>
    </form>
  );
}
