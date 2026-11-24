"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch, apiUpload, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

type BatchVariant = {
  id: string;
  variant_id: string;
  variant_label: string;
  effective_price_idr: number;
  quantity_limit?: number | null;
  availability: string;
};

type CatalogVariant = {
  id: string;
  option_key: string;
  effective_price_idr: number;
  sku?: string | null;
};

export function BatchCoverEditor({
  storeId,
  batchId,
  coverFileId,
}: {
  storeId: string;
  batchId: string;
  coverFileId?: string | null;
}) {
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function onUpload(file: File) {
    setBusy(true);
    setError(null);
    try {
      const form = new FormData();
      form.set("purpose", "preorder_cover");
      form.set("file", file);
      const uploaded = await apiUpload<{ file: { id: string } }>(
        `/api/v1/stores/${storeId}/files`,
        form,
      );
      await apiFetch(`/api/v1/stores/${storeId}/preorders/${batchId}`, {
        method: "PATCH",
        body: { cover_file_id: uploaded.file.id },
      });
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Cover upload failed.");
    } finally {
      setBusy(false);
    }
  }

  async function onClear() {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/preorders/${batchId}`, {
        method: "PATCH",
        body: { clear_cover: true },
      });
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to clear cover.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="rounded-[10px] border border-border p-5">
      <h3 className="text-base font-medium">Batch cover</h3>
      <p className="mt-1 text-sm text-foreground-secondary">
        Optional image shown on the storefront preorder list.
      </p>
      {coverFileId ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={urls.file(storeId, coverFileId)}
          alt="Batch cover"
          className="mt-4 h-40 w-full max-w-sm rounded-md object-cover"
        />
      ) : (
        <p className="mt-4 text-sm text-foreground-muted">No cover image yet.</p>
      )}
      <div className="mt-4 flex flex-wrap items-center gap-2">
        <Input
          type="file"
          accept="image/png,image/jpeg,image/webp"
          disabled={busy}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void onUpload(f);
          }}
        />
        {coverFileId ? (
          <Button type="button" variant="outline" disabled={busy} onClick={onClear}>
            Remove cover
          </Button>
        ) : null}
      </div>
      {error ? <p className="mt-2 text-sm text-error">{error}</p> : null}
    </div>
  );
}

export function BatchVariantPairing({
  storeId,
  batchId,
  productId,
  productName,
}: {
  storeId: string;
  batchId: string;
  productId: string;
  productName: string;
}) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [catalogVariants, setCatalogVariants] = useState<CatalogVariant[]>([]);
  const [assigned, setAssigned] = useState<BatchVariant[]>([]);
  const [variantId, setVariantId] = useState("");
  const [priceOverride, setPriceOverride] = useState("");
  const [qtyLimit, setQtyLimit] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    let cancelled = false;
    async function load() {
      try {
        const [product, list] = await Promise.all([
          apiFetch<{ product: { variants?: CatalogVariant[] } }>(
            `/api/v1/stores/${storeId}/products/${productId}`,
          ),
          apiFetch<{ variants: BatchVariant[] }>(
            `/api/v1/stores/${storeId}/preorders/${batchId}/products/${productId}/variants`,
          ),
        ]);
        if (cancelled) return;
        setCatalogVariants(product.product?.variants || []);
        setAssigned(list.variants || []);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.message : "Unable to load variants.");
        }
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [open, storeId, batchId, productId]);

  async function assign() {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(
        `/api/v1/stores/${storeId}/preorders/${batchId}/products/${productId}/variants`,
        {
          method: "POST",
          body: {
            variant_id: variantId,
            price_override_idr: priceOverride ? Number(priceOverride) : undefined,
            quantity_limit: qtyLimit ? Number(qtyLimit) : undefined,
            availability: "available",
          },
        },
      );
      setVariantId("");
      setPriceOverride("");
      setQtyLimit("");
      const list = await apiFetch<{ variants: BatchVariant[] }>(
        `/api/v1/stores/${storeId}/preorders/${batchId}/products/${productId}/variants`,
      );
      setAssigned(list.variants || []);
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Assign failed.");
    } finally {
      setBusy(false);
    }
  }

  async function remove(vid: string) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(
        `/api/v1/stores/${storeId}/preorders/${batchId}/products/${productId}/variants/${vid}`,
        { method: "DELETE" },
      );
      setAssigned((prev) => prev.filter((v) => v.variant_id !== vid));
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Remove failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-2">
      <button
        type="button"
        className="text-sm text-brand-dark hover:underline"
        onClick={() => setOpen((v) => !v)}
      >
        {open ? "Hide variants" : `Pair variants · ${productName}`}
      </button>
      {open ? (
        <div className="mt-3 space-y-3 rounded-md border border-border p-4">
          {error ? <p className="text-sm text-error">{error}</p> : null}
          {catalogVariants.length === 0 ? (
            <p className="text-sm text-foreground-secondary">
              This product has no variants. Add options on the product editor first.
            </p>
          ) : (
            <div className="grid gap-3 sm:grid-cols-4">
              <div className="sm:col-span-2 space-y-1">
                <Label>Variant</Label>
                <select
                  className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
                  value={variantId}
                  onChange={(e) => setVariantId(e.target.value)}
                >
                  <option value="">Select variant</option>
                  {catalogVariants.map((v) => (
                    <option key={v.id} value={v.id}>
                      {v.option_key || "Default"} · {formatIDR(v.effective_price_idr)}
                    </option>
                  ))}
                </select>
              </div>
              <div className="space-y-1">
                <Label>Batch price</Label>
                <Input value={priceOverride} onChange={(e) => setPriceOverride(e.target.value)} />
              </div>
              <div className="space-y-1">
                <Label>Quota</Label>
                <Input value={qtyLimit} onChange={(e) => setQtyLimit(e.target.value)} />
              </div>
              <div className="sm:col-span-4">
                <Button type="button" disabled={busy || !variantId} onClick={assign}>
                  Assign variant
                </Button>
              </div>
            </div>
          )}
          {assigned.length > 0 ? (
            <ul className="space-y-2 text-sm">
              {assigned.map((v) => (
                <li key={v.id} className="flex items-center justify-between gap-2">
                  <span>
                    {v.variant_label} · {formatIDR(v.effective_price_idr)}
                    {v.quantity_limit != null ? ` · quota ${v.quantity_limit}` : ""}
                  </span>
                  <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => remove(v.variant_id)}>
                    Remove
                  </Button>
                </li>
              ))}
            </ul>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
