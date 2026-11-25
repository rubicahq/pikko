"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  apiFetch,
  apiUpload,
  ATTRIBUTE_OPTION_COLOR_HEX,
  formatIDRDigits,
  maskIDRInput,
  parseIDRInput,
  type AttributeDefinition,
  type AttributeOptionColor,
  type Product,
  type ProductAttributeValue,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

type AttrDraft = {
  text: string;
  number: string;
  optionIds: string[];
};

function draftFromValues(
  attributes: AttributeDefinition[],
  values: ProductAttributeValue[],
): Record<string, AttrDraft> {
  const byAttr = new Map(values.map((v) => [v.attribute_id, v]));
  const out: Record<string, AttrDraft> = {};
  for (const a of attributes) {
    const v = byAttr.get(a.id);
    out[a.id] = {
      text: v?.value_text || "",
      number: v?.value_number != null ? String(v.value_number) : "",
      optionIds: v?.option_ids || [],
    };
  }
  return out;
}

export function ProductForm({
  storeId,
  attributes,
  attributeValues,
  product,
}: {
  storeId: string;
  attributes: AttributeDefinition[];
  attributeValues?: ProductAttributeValue[];
  product?: Product | null;
}) {
  const router = useRouter();
  const [name, setName] = useState(product?.name || "");
  const [sku, setSku] = useState(product?.sku || "");
  const [description, setDescription] = useState(product?.description || "");
  const [price, setPrice] = useState(
    product?.base_price_idr != null ? formatIDRDigits(product.base_price_idr) : "",
  );
  const [status, setStatus] = useState(product?.status || "draft");
  const [readyStock, setReadyStock] = useState(product?.ready_stock_eligible ?? true);
  const [preorder, setPreorder] = useState(product?.preorder_eligible ?? false);
  const [imageIds, setImageIds] = useState<string[]>(
    product?.images?.map((i) => i.file_id) || [],
  );
  const [previews, setPreviews] = useState<string[]>(
    product?.images?.map((i) => i.url || "").filter(Boolean) || [],
  );
  const [attrDraft, setAttrDraft] = useState<Record<string, AttrDraft>>(() =>
    draftFromValues(attributes, attributeValues || []),
  );
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  const sortedAttributes = useMemo(
    () =>
      [...attributes].sort(
        (a, b) => (a.display_order ?? 0) - (b.display_order ?? 0) || (a.name || a.label).localeCompare(b.name || b.label),
      ),
    [attributes],
  );

  function setAttr(id: string, patch: Partial<AttrDraft>) {
    setAttrDraft((prev) => ({
      ...prev,
      [id]: { ...(prev[id] || { text: "", number: "", optionIds: [] }), ...patch },
    }));
  }

  function buildAttributePayload() {
    return sortedAttributes.map((a) => {
      const t = (a.type || a.data_type) as string;
      const d = attrDraft[a.id] || { text: "", number: "", optionIds: [] };
      if (t === "short_text") {
        return { attribute_id: a.id, value_text: d.text.trim() || null, option_ids: [] };
      }
      if (t === "number") {
        const n = d.number.trim() === "" ? null : Number(d.number);
        return {
          attribute_id: a.id,
          value_number: n != null && Number.isFinite(n) ? n : null,
          option_ids: [],
        };
      }
      return { attribute_id: a.id, option_ids: d.optionIds, value_text: null };
    });
  }

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
    const base = parseIDRInput(price);
    if (!name.trim() || base == null || base < 0) {
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
        status,
        ready_stock_eligible: readyStock,
        preorder_eligible: preorder,
      },
      options: [],
      variants: [],
      image_file_ids: imageIds,
      attributes: buildAttributePayload(),
    };
    try {
      if (product) {
        await apiFetch(`/api/v1/stores/${storeId}/products/${product.id}`, {
          method: "PATCH",
          body,
        });
        router.push(urls.storeProducts(storeId));
        router.refresh();
      } else {
        const created = await apiFetch<{ product: Product }>(
          `/api/v1/stores/${storeId}/products`,
          { method: "POST", body },
        );
        router.push(urls.storeProduct(storeId, created.product.id));
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
          autoComplete="off"
          value={price}
          onChange={(e) => setPrice(maskIDRInput(e.target.value))}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="description">Description</Label>
        <textarea
          id="description"
          className="min-h-28 w-full rounded-[10px] border border-border bg-background px-3 py-2 text-sm"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>

      {sortedAttributes.length > 0 ? (
        <div className="space-y-4">
          <div>
            <h2 className="text-base font-semibold text-foreground">Attributes</h2>
            <p className="mt-1 text-sm text-foreground-secondary">
              Values from your store&apos;s attribute definitions.
            </p>
          </div>
          {sortedAttributes.map((a) => {
            const t = (a.type || a.data_type) as string;
            const d = attrDraft[a.id] || { text: "", number: "", optionIds: [] };
            const label = a.name || a.label;
            if (t === "short_text") {
              return (
                <div key={a.id} className="space-y-2">
                  <Label htmlFor={`attr-${a.id}`}>{label}</Label>
                  <Input
                    id={`attr-${a.id}`}
                    value={d.text}
                    onChange={(e) => setAttr(a.id, { text: e.target.value })}
                  />
                </div>
              );
            }
            if (t === "number") {
              return (
                <div key={a.id} className="space-y-2">
                  <Label htmlFor={`attr-${a.id}`}>{label}</Label>
                  <Input
                    id={`attr-${a.id}`}
                    inputMode="decimal"
                    value={d.number}
                    onChange={(e) => setAttr(a.id, { number: e.target.value })}
                  />
                </div>
              );
            }
            if (t === "single_option") {
              return (
                <div key={a.id} className="space-y-2">
                  <Label htmlFor={`attr-${a.id}`}>{label}</Label>
                  <select
                    id={`attr-${a.id}`}
                    className="h-11 w-full rounded-[10px] border border-border bg-background px-3 text-sm"
                    value={d.optionIds[0] || ""}
                    onChange={(e) =>
                      setAttr(a.id, { optionIds: e.target.value ? [e.target.value] : [] })
                    }
                  >
                    <option value="">—</option>
                    {(a.options || []).map((o) => (
                      <option key={o.id} value={o.id}>
                        {o.label}
                      </option>
                    ))}
                  </select>
                  {(a.options || []).length > 0 ? (
                    <div className="flex flex-wrap gap-2 pt-1">
                      {(a.options || []).map((o) => (
                        <span key={o.id} className="inline-flex items-center gap-1.5 text-xs text-foreground-muted">
                          <span
                            className="inline-block h-2 w-2 rounded-full"
                            style={{
                              backgroundColor:
                                ATTRIBUTE_OPTION_COLOR_HEX[
                                  (o.color as AttributeOptionColor) in ATTRIBUTE_OPTION_COLOR_HEX
                                    ? (o.color as AttributeOptionColor)
                                    : "slate"
                                ],
                            }}
                          />
                          {o.label}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </div>
              );
            }
            // multiple_option
            return (
              <div key={a.id} className="space-y-2">
                <Label>{label}</Label>
                <div className="space-y-2 rounded-[14px] border border-border px-3 py-3">
                  {(a.options || []).length === 0 ? (
                    <p className="text-sm text-foreground-muted">No options defined.</p>
                  ) : (
                    (a.options || []).map((o) => {
                      const checked = d.optionIds.includes(o.id);
                      return (
                        <label key={o.id} className="flex items-center gap-2 text-sm">
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={(e) => {
                              const next = e.target.checked
                                ? [...d.optionIds, o.id]
                                : d.optionIds.filter((id) => id !== o.id);
                              setAttr(a.id, { optionIds: next });
                            }}
                          />
                          <span
                            className="inline-block h-2.5 w-2.5 rounded-full"
                            style={{
                              backgroundColor:
                                ATTRIBUTE_OPTION_COLOR_HEX[
                                  (o.color as AttributeOptionColor) in ATTRIBUTE_OPTION_COLOR_HEX
                                    ? (o.color as AttributeOptionColor)
                                    : "slate"
                                ],
                            }}
                          />
                          {o.label}
                        </label>
                      );
                    })
                  )}
                </div>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="rounded-[14px] border border-border bg-background-subtle/60 px-4 py-3 text-sm text-foreground-secondary">
          No attributes yet.{" "}
          <Link href={urls.storeAttributes(storeId)} className="font-medium text-foreground underline">
            Add attributes
          </Link>{" "}
          to capture brand, category, and other custom fields on products.
        </p>
      )}

      <div className="space-y-2">
        <Label htmlFor="status">Status</Label>
        <select
          id="status"
          className="h-11 w-full rounded-[10px] border border-border bg-background px-3 text-sm"
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
              className="h-24 w-24 rounded-[10px] border border-border object-cover"
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
        <Button type="button" variant="outline" onClick={() => router.push(urls.storeProducts(storeId))}>
          Cancel
        </Button>
      </div>
    </form>
  );
}
