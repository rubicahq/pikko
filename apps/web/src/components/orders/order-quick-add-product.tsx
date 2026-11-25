"use client";

import { useEffect, useMemo, useState } from "react";
import {
  ApiError,
  apiFetch,
  ATTRIBUTE_OPTION_COLOR_HEX,
  type AttributeDefinition,
  type AttributeOptionColor,
  type Product,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { IdrInput } from "@/components/ui/idr-input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

type AttrDraft = { text: string; number: string; optionIds: string[] };

type Props = {
  storeId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  initialName?: string;
  onCreated: (product: Product) => void;
};

export function OrderQuickAddProduct({
  storeId,
  open,
  onOpenChange,
  initialName = "",
  onCreated,
}: Props) {
  const [name, setName] = useState(initialName);
  const [price, setPrice] = useState<number | null>(null);
  const [attributes, setAttributes] = useState<AttributeDefinition[]>([]);
  const [attrDraft, setAttrDraft] = useState<Record<string, AttrDraft>>({});
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    setName(initialName);
    setPrice(null);
    setError(null);
    void apiFetch<{ attributes: AttributeDefinition[] }>(
      `/api/v1/stores/${storeId}/attributes`,
    )
      .then((res) => {
        const list = res.attributes || [];
        setAttributes(list);
        const draft: Record<string, AttrDraft> = {};
        for (const a of list) {
          draft[a.id] = { text: "", number: "", optionIds: [] };
        }
        setAttrDraft(draft);
      })
      .catch(() => {
        setAttributes([]);
        setAttrDraft({});
      });
  }, [open, storeId, initialName]);

  const optionAttrs = useMemo(
    () =>
      attributes
        .filter((a) => {
          const t = a.type || a.data_type;
          return t === "single_option" || t === "multiple_option";
        })
        .sort(
          (a, b) =>
            (a.display_order ?? 0) - (b.display_order ?? 0) ||
            (a.name || a.label).localeCompare(b.name || b.label),
        ),
    [attributes],
  );

  function setAttr(id: string, patch: Partial<AttrDraft>) {
    setAttrDraft((prev) => ({
      ...prev,
      [id]: { ...(prev[id] || { text: "", number: "", optionIds: [] }), ...patch },
    }));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const trimmed = name.trim();
    if (!trimmed || price == null || price < 0 || !Number.isInteger(price)) {
      setError("Name and a whole-rupiah IDR price are required.");
      return;
    }
    for (const a of optionAttrs) {
      if (!a.is_required) continue;
      const d = attrDraft[a.id];
      if (!d?.optionIds?.length) {
        setError(`${a.name || a.label} is required.`);
        return;
      }
    }
    setBusy(true);
    try {
      const attrPayload = optionAttrs.map((a) => ({
        attribute_id: a.id,
        option_ids: attrDraft[a.id]?.optionIds || [],
        value_text: null,
      }));
      const created = await apiFetch<{ product: Product }>(
        `/api/v1/stores/${storeId}/products`,
        {
          method: "POST",
          body: {
            product: {
              name: trimmed,
              base_price_idr: price,
              status: "published",
              ready_stock_eligible: true,
              inventory_mode: "unlimited",
            },
            options: [],
            variants: [],
            attributes: attrPayload,
          },
        },
      );
      onCreated(created.product);
      onOpenChange(false);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to create product.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add product</DialogTitle>
          <DialogDescription>
            Creates a published product and adds it to this order line. Prices are whole rupiah.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={onSubmit} className="space-y-4">
          <div className="space-y-1.5">
            <Label htmlFor="quick-product-name">Name</Label>
            <Input
              id="quick-product-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="quick-product-price">Price (IDR)</Label>
            <IdrInput
              id="quick-product-price"
              value={price}
              onValueChange={setPrice}
              required
            />
          </div>
          {optionAttrs.map((a) => {
            const t = a.type || a.data_type;
            const d = attrDraft[a.id] || { text: "", number: "", optionIds: [] };
            const label = `${a.name || a.label}${a.is_required ? " *" : ""}`;
            if (t === "single_option") {
              return (
                <div key={a.id} className="space-y-2">
                  <Label>{label}</Label>
                  <div className="flex flex-wrap gap-2">
                    {(a.options || []).map((o) => {
                      const active = d.optionIds[0] === o.id;
                      const color =
                        ATTRIBUTE_OPTION_COLOR_HEX[
                          (o.color as AttributeOptionColor) in ATTRIBUTE_OPTION_COLOR_HEX
                            ? (o.color as AttributeOptionColor)
                            : "slate"
                        ];
                      return (
                        <button
                          key={o.id}
                          type="button"
                          onClick={() => setAttr(a.id, { optionIds: [o.id] })}
                          className={`inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-sm ${
                            active
                              ? "border-ink bg-surface-active"
                              : "border-border hover:bg-background-subtle"
                          }`}
                        >
                          <span
                            className="inline-block h-2.5 w-2.5 rounded-full"
                            style={{ backgroundColor: color }}
                          />
                          {o.label}
                        </button>
                      );
                    })}
                  </div>
                </div>
              );
            }
            return (
              <div key={a.id} className="space-y-2">
                <Label>{label}</Label>
                <div className="flex flex-wrap gap-2">
                  {(a.options || []).map((o) => {
                    const active = d.optionIds.includes(o.id);
                    const color =
                      ATTRIBUTE_OPTION_COLOR_HEX[
                        (o.color as AttributeOptionColor) in ATTRIBUTE_OPTION_COLOR_HEX
                          ? (o.color as AttributeOptionColor)
                          : "slate"
                      ];
                    return (
                      <button
                        key={o.id}
                        type="button"
                        onClick={() => {
                          const next = active
                            ? d.optionIds.filter((id) => id !== o.id)
                            : [...d.optionIds, o.id];
                          setAttr(a.id, { optionIds: next });
                        }}
                        className={`inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-sm ${
                          active
                            ? "border-ink bg-surface-active"
                            : "border-border hover:bg-background-subtle"
                        }`}
                      >
                        <span
                          className="inline-block h-2.5 w-2.5 rounded-full"
                          style={{ backgroundColor: color }}
                        />
                        {o.label}
                      </button>
                    );
                  })}
                </div>
              </div>
            );
          })}
          {error ? <p className="text-sm text-error">{error}</p> : null}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? "Creating…" : "Create & select"}
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
}
