"use client";

import { useCallback, useEffect, useState } from "react";
import { ApiError, apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

type Relation = {
  id: string;
  product_id: string;
  related_product_id: string;
  relation_type: string;
  related_name?: string;
  related_slug?: string;
  product_name?: string;
  notes?: string | null;
};

type ProductOption = {
  id: string;
  name: string;
  status: string;
};

export function ProductRelations({
  storeId,
  productId,
}: {
  storeId: string;
  productId: string;
}) {
  const [relations, setRelations] = useState<Relation[]>([]);
  const [products, setProducts] = useState<ProductOption[]>([]);
  const [relatedId, setRelatedId] = useState("");
  const [relationType, setRelationType] = useState<"same_item" | "merged_into">("same_item");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const [rel, list] = await Promise.all([
        apiFetch<{ relations: Relation[] }>(
          `/api/v1/stores/${storeId}/products/${productId}/relations`,
        ),
        apiFetch<{ products: ProductOption[] }>(
          `/api/v1/stores/${storeId}/products?limit=100`,
        ),
      ]);
      setRelations(rel.relations || []);
      setProducts((list.products || []).filter((p) => p.id !== productId));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to load relations.");
    }
  }, [storeId, productId]);

  useEffect(() => {
    load();
  }, [load]);

  async function create(e: React.FormEvent) {
    e.preventDefault();
    if (!relatedId) return;
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/products/${productId}/relations`, {
        method: "POST",
        body: { related_product_id: relatedId, relation_type: relationType },
      });
      setRelatedId("");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to create relation.");
    } finally {
      setBusy(false);
    }
  }

  async function remove(id: string) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/products/${productId}/relations/${id}`, {
        method: "DELETE",
      });
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to remove relation.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="space-y-4 rounded-lg border border-border bg-background p-5">
      <div>
        <h2 className="text-base font-semibold text-foreground">Relate / merge across products</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          AI confirm always creates a new product. Use this to manually mark the same physical item
          across catalogs, or mark a merge target.
        </p>
      </div>
      {error && <p className="text-sm text-error">{error}</p>}
      <ul className="space-y-2">
        {relations.length === 0 && (
          <li className="text-sm text-foreground-muted">No relations yet.</li>
        )}
        {relations.map((r) => (
          <li
            key={r.id}
            className="flex flex-wrap items-center justify-between gap-2 rounded-md border border-border px-3 py-2 text-sm"
          >
            <span>
              <span className="font-medium">{r.relation_type}</span>
              {" → "}
              {r.product_id === productId ? r.related_name : r.product_name}
            </span>
            <Button type="button" size="sm" variant="ghost" disabled={busy} onClick={() => remove(r.id)}>
              Remove
            </Button>
          </li>
        ))}
      </ul>
      <form onSubmit={create} className="grid gap-3 sm:grid-cols-[1fr_auto_auto]">
        <div>
          <Label htmlFor="related-product">Existing product</Label>
          <select
            id="related-product"
            className="mt-1 w-full rounded-[10px] border border-border bg-background px-3 py-2 text-sm"
            value={relatedId}
            onChange={(e) => setRelatedId(e.target.value)}
          >
            <option value="">Select product…</option>
            {products.map((p) => (
              <option key={p.id} value={p.id}>
                {p.name} ({p.status})
              </option>
            ))}
          </select>
        </div>
        <div>
          <Label htmlFor="relation-type">Type</Label>
          <select
            id="relation-type"
            className="mt-1 w-full rounded-[10px] border border-border bg-background px-3 py-2 text-sm"
            value={relationType}
            onChange={(e) => setRelationType(e.target.value as "same_item" | "merged_into")}
          >
            <option value="same_item">Same item</option>
            <option value="merged_into">Merged into</option>
          </select>
        </div>
        <div className="flex items-end">
          <Button type="submit" disabled={busy || !relatedId}>
            Relate
          </Button>
        </div>
      </form>
    </section>
  );
}
