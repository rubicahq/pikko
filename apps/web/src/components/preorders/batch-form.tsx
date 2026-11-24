"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

function toLocalInput(d: Date) {
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

export function BatchForm({ storeId }: { storeId: string }) {
  const router = useRouter();
  const now = new Date();
  const later = new Date(now.getTime() + 7 * 24 * 60 * 60 * 1000);
  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [description, setDescription] = useState("");
  const [opensAt, setOpensAt] = useState(toLocalInput(now));
  const [closesAt, setClosesAt] = useState(toLocalInput(later));
  const [etaStart, setEtaStart] = useState("");
  const [etaEnd, setEtaEnd] = useState("");
  const [publish, setPublish] = useState(true);
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
      const res = await apiFetch<{ batch: { id: string } }>(`/api/v1/stores/${storeId}/preorders`, {
        method: "POST",
        body: {
          name,
          slug,
          description: description || undefined,
          opens_at: new Date(opensAt).toISOString(),
          closes_at: new Date(closesAt).toISOString(),
          eta_start: etaStart || undefined,
          eta_end: etaEnd || undefined,
          publication_status: publish ? "published" : "draft",
          payment_policy: "full_after_checkout",
        },
      });
      router.push(`/stores/${storeId}/preorders/${res.batch.id}`);
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not create batch.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="mt-8 max-w-xl space-y-5">
      {error ? <p className="text-sm text-error">{error}</p> : null}
      <div className="space-y-2">
        <Label htmlFor="name">Batch name</Label>
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
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="etaStart">ETA start (optional)</Label>
          <Input id="etaStart" type="date" value={etaStart} onChange={(e) => setEtaStart(e.target.value)} />
        </div>
        <div className="space-y-2">
          <Label htmlFor="etaEnd">ETA end (optional)</Label>
          <Input id="etaEnd" type="date" value={etaEnd} onChange={(e) => setEtaEnd(e.target.value)} />
        </div>
      </div>
      <label className="flex items-center gap-2 text-sm text-foreground">
        <input type="checkbox" checked={publish} onChange={(e) => setPublish(e.target.checked)} />
        Publish immediately
      </label>
      <Button type="submit" disabled={busy}>
        {busy ? "Creating…" : "Create batch"}
      </Button>
    </form>
  );
}

function suggest(name: string) {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 50);
}
