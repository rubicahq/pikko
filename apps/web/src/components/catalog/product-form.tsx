"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, apiUpload, formatIDR, type Category, type Product } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

export function ProductForm({
  storeId,
  categories,
  product,
}: {
  storeId: string;
  categories: Category[];
  product?: Product | null;
}) {
  const router = useRouter();
  const [name, setName] = useState(product?.name || "");
  const [sku, setSku] = useState(product?.sku || "");
  const [description, setDescription] = useState(product?.description || "");
  const [price, setPrice] = useState(String(product?.base_price_idr ?? ""));
  const [categoryId, setCategoryID] = useState(product?.category_id || "");
  const [status, setStatus] = useState(product?.status || "draft");
  const [readyStock, setReadyStock] = useState(product?.ready_stock_eligible ?? true);
  const [preorder, setPreorder] = useState(product?.preorder_eligible ?? false);
  const [imageIds, setImageIds] = useState<string[]>(
    product?.images?.map((i) => i.file_id) || [],
  );
  const [previews, setPreviews] = useState<string[]>(
    product?.images?.map((i) => i.url || "").filter(Boolean) || [],
  );
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  async function onUpload(file: File) {
    const form = new FormData();
    form.set("purpose", "product");
    form.set("file", file);
    const res = await apiUpload<{ file: { id: string; url?: string } }>(
      `/api/v1/stores/${storeId}/files`,
      form,
    );
    setImageIds((ids) => [...ids, res.file.id]);
    setPreviews((p) => [...p, res.file.url || urls.file(storeId, res.file.id)]);
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    const base = Number.parseInt(price.replace(/\D/g, ""), 10);
    if (!name.trim() || Number.isNaN(base) || base < 0) {
      setError("Name and a valid IDR price are required.");
      setSaving(false);
      return;
    }
    const body = {
      product: {
        name: name.trim(),
        sku: sku.trim() || null,
        description: description.trim() || null,
        base_price_idr: base,
        category_id: categoryId || null,
        status,
        ready_stock_eligible: readyStock,
        preorder_eligible: preorder,
      },
      options: [],
      variants: [],
      image_file_ids: imageIds,
    };
    try {
      if (product) {
        await apiFetch(`/api/v1/stores/${storeId}/products/${product.id}`, {
          method: "PATCH",
          body,
        });
        router.push(urls.storeCatalog(storeId));
        router.refresh();
      } else {
        const created = await apiFetch<{ product: Product }>(
          `/api/v1/stores/${storeId}/products`,
          { method: "POST", body },
        );
        router.push(urls.storeCatalogProduct(storeId, created.product.id));
        router.refresh();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save product.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="mx-auto max-w-reading space-y-6">
      <div>
        <h1 className="text-[30px] font-medium text-foreground">
          {product ? "Edit product" : "Add product"}
        </h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Prices are whole rupiah integers. Draft products stay private until published.
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="name">Name</Label>
        <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div className="space-y-2">
        <Label htmlFor="sku">SKU (optional)</Label>
        <Input id="sku" value={sku} onChange={(e) => setSku(e.target.value)} />
      </div>
      <div className="space-y-2">
        <Label htmlFor="price">Price (IDR)</Label>
        <Input
          id="price"
          inputMode="numeric"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
          required
        />
        {price && !Number.isNaN(Number.parseInt(price.replace(/\D/g, ""), 10)) && (
          <p className="text-xs text-foreground-muted">
            {formatIDR(Number.parseInt(price.replace(/\D/g, ""), 10))}
          </p>
        )}
      </div>
      <div className="space-y-2">
        <Label htmlFor="description">Description</Label>
        <textarea
          id="description"
          className="min-h-28 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="category">Category</Label>
        <select
          id="category"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
          value={categoryId}
          onChange={(e) => setCategoryID(e.target.value)}
        >
          <option value="">Uncategorized</option>
          {categories.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </select>
      </div>
      <div className="space-y-2">
        <Label htmlFor="status">Status</Label>
        <select
          id="status"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
          value={status}
          onChange={(e) => setStatus(e.target.value as Product["status"])}
        >
          <option value="draft">Draft</option>
          <option value="published">Published</option>
          <option value="archived">Archived</option>
        </select>
      </div>

      <div className="space-y-2">
        <Label>Selling model</Label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={readyStock}
            onChange={(e) => setReadyStock(e.target.checked)}
          />
          Ready stock eligible
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={preorder}
            onChange={(e) => setPreorder(e.target.checked)}
          />
          Preorder eligible
        </label>
      </div>

      <div className="space-y-3">
        <Label htmlFor="images">Images</Label>
        <div className="flex flex-wrap gap-3">
          {previews.map((src) => (
            // eslint-disable-next-line @next/next/no-img-element
            <img
              key={src}
              src={src}
              alt=""
              className="h-24 w-24 rounded-md border border-border object-cover"
            />
          ))}
        </div>
        <Input
          id="images"
          type="file"
          accept="image/png,image/jpeg,image/webp"
          onChange={async (e) => {
            const file = e.target.files?.[0];
            if (!file) return;
            try {
              await onUpload(file);
            } catch (err) {
              setError(err instanceof Error ? err.message : "Upload failed.");
            }
          }}
        />
      </div>

      {error && <p className="text-sm text-error">{error}</p>}

      <div className="flex flex-wrap gap-3">
        <Button type="submit" loading={saving}>
          Save product
        </Button>
        <Button type="button" variant="outline" onClick={() => router.push(urls.storeCatalog(storeId))}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
