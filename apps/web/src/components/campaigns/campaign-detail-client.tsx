"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatusBadge } from "@/components/ui/status-badge";
import { CatalogAttach } from "@/components/catalog/catalog-attach";
import { urls } from "@/lib/urls";

type CampaignProduct = {
  id: string;
  product_id: string;
  product_name?: string;
  effective_price_idr: number;
  price_override_idr?: number | null;
  quantity_limit?: number | null;
  availability: string;
  selling_phase?: string | null;
  featured?: boolean;
};

type Campaign = {
  id: string;
  name: string;
  slug: string;
  campaign_type: string;
  opens_at: string;
  closes_at: string;
  operational_status: string;
  publication_status: string;
  preorder_batch_id?: string | null;
  live_quick_add_enabled?: boolean;
  products?: CampaignProduct[];
};

type CatalogProduct = {
  id: string;
  name: string;
  status: string;
  preorder_eligible?: boolean;
  base_price_idr: number;
};

export function CampaignDetailClient({
  storeId,
  campaign: initial,
  catalog,
}: {
  storeId: string;
  campaign: Campaign;
  catalog: CatalogProduct[];
}) {
  const router = useRouter();
  const [campaign, setCampaign] = useState(initial);
  const [productId, setProductId] = useState("");
  const [override, setOverride] = useState("");
  const [limit, setLimit] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const candidates = useMemo(() => {
    const assigned = new Set((campaign.products || []).map((p) => p.product_id));
    return catalog.filter((p) => p.status === "published" && !assigned.has(p.id));
  }, [catalog, campaign.products]);

  async function refresh() {
    const res = await apiFetch<{ campaign: Campaign }>(
      `/api/v1/stores/${storeId}/campaigns/${campaign.id}`,
    );
    setCampaign(res.campaign);
    router.refresh();
  }

  async function assign(e: React.FormEvent) {
    e.preventDefault();
    if (!productId) return;
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/campaigns/${campaign.id}/products`, {
        method: "POST",
        body: {
          product_id: productId,
          price_override_idr: override ? Number(override.replace(/\D/g, "")) : null,
          quantity_limit: limit ? Number(limit) : null,
          availability: "available",
          display_order: (campaign.products?.length || 0) + 1,
        },
      });
      setProductId("");
      setOverride("");
      setLimit("");
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not assign product.");
    } finally {
      setBusy(false);
    }
  }

  async function remove(pid: string) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/campaigns/${campaign.id}/products/${pid}`, {
        method: "DELETE",
      });
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not remove product.");
    } finally {
      setBusy(false);
    }
  }

  async function toggleFeatured(p: CampaignProduct) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/campaigns/${campaign.id}/products`, {
        method: "POST",
        body: {
          product_id: p.product_id,
          price_override_idr: p.price_override_idr ?? null,
          quantity_limit: p.quantity_limit ?? null,
          availability: p.availability,
          selling_phase: p.selling_phase ?? null,
          display_order: 0,
          featured: !p.featured,
        },
      });
      await refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Could not update featured.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-xs uppercase tracking-wide text-foreground-muted">
            {campaign.campaign_type}
          </p>
          <h1 className="mt-1 text-[30px] font-medium text-foreground">{campaign.name}</h1>
          <p className="mt-2 text-sm text-foreground-secondary">/{campaign.slug}</p>
          <div className="mt-3 flex flex-wrap items-center gap-2">
            <StatusBadge status={campaign.operational_status} />
            <span className="text-xs text-foreground-muted">{campaign.publication_status}</span>
          </div>
        </div>
        <div className="flex flex-wrap gap-2">
          {campaign.campaign_type === "live" ? (
            <Button asChild>
              <Link href={urls.storeLiveSession(storeId, campaign.id)}>Open live console</Link>
            </Button>
          ) : null}
          {campaign.preorder_batch_id ? (
            <Button asChild variant="outline">
              <Link href={urls.storePreorder(storeId, campaign.preorder_batch_id)}>
                Open linked preorder batch
              </Link>
            </Button>
          ) : null}
        </div>
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}

      <section className="space-y-3">
        <h2 className="text-base font-medium text-foreground">Assigned products</h2>
        {(campaign.products || []).length === 0 ? (
          <p className="text-sm text-foreground-secondary">No products assigned yet.</p>
        ) : (
          <ul className="divide-y divide-border rounded-[10px] border border-border">
            {(campaign.products || []).map((p) => (
              <li key={p.id} className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                <div>
                  <p className="font-medium text-foreground">{p.product_name || p.product_id}</p>
                  <p className="text-foreground-secondary">
                    {formatIDR(p.effective_price_idr)}
                    {p.selling_phase ? ` · ${p.selling_phase}` : ""}
                    {p.quantity_limit != null ? ` · quota ${p.quantity_limit}` : ""}
                  </p>
                </div>
                <div className="flex items-center gap-2">
                  {campaign.campaign_type === "live" ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      disabled={busy}
                      onClick={() => toggleFeatured(p)}
                    >
                      {p.featured ? "Unfeature" : "Feature"}
                    </Button>
                  ) : null}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    disabled={busy}
                    onClick={() => remove(p.product_id)}
                  >
                    Remove
                  </Button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      <CatalogAttach storeId={storeId} mode="campaign" targetId={campaign.id} />

      <form onSubmit={assign} className="max-w-xl space-y-4 rounded-[10px] border border-border p-5">
        <h2 className="text-base font-medium text-foreground">Assign product</h2>
        <div className="space-y-2">
          <Label htmlFor="product">Product</Label>
          <select
            id="product"
            className="h-10 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={productId}
            onChange={(e) => setProductId(e.target.value)}
            required
          >
            <option value="">Select…</option>
            {candidates.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name} ({formatIDR(p.base_price_idr)})
              </option>
            ))}
          </select>
        </div>
        <div className="grid gap-4 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="override">Price override (IDR)</Label>
            <Input id="override" value={override} onChange={(e) => setOverride(e.target.value)} />
          </div>
          <div className="space-y-2">
            <Label htmlFor="limit">Quantity limit</Label>
            <Input id="limit" value={limit} onChange={(e) => setLimit(e.target.value)} />
          </div>
        </div>
        <Button type="submit" disabled={busy || !productId}>
          Assign
        </Button>
      </form>
    </div>
  );
}
