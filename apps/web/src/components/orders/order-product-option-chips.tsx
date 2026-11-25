"use client";

import {
  ATTRIBUTE_OPTION_COLOR_HEX,
  type AttributeDefinition,
  type AttributeOptionColor,
} from "@/lib/api";
import { formatIDR } from "@/lib/api";
import { Label } from "@/components/ui/label";

export type ProductVariantOption = {
  id: string;
  option_key: string;
  effective_price_idr: number;
  sku?: string | null;
};

type Props = {
  variants: ProductVariantOption[];
  selectedId?: string;
  attributes?: AttributeDefinition[];
  onSelect: (variant: ProductVariantOption) => void;
  error?: string;
};

function colorForLabel(
  label: string,
  attributes: AttributeDefinition[] | undefined,
): string {
  const needle = label.trim().toLowerCase();
  if (!needle || !attributes?.length) {
    return ATTRIBUTE_OPTION_COLOR_HEX.slate;
  }
  for (const a of attributes) {
    const t = a.type || a.data_type;
    if (t !== "single_option" && t !== "multiple_option") continue;
    for (const o of a.options || []) {
      if (o.label.trim().toLowerCase() === needle) {
        const key =
          (o.color as AttributeOptionColor) in ATTRIBUTE_OPTION_COLOR_HEX
            ? (o.color as AttributeOptionColor)
            : "slate";
        return ATTRIBUTE_OPTION_COLOR_HEX[key];
      }
    }
  }
  return ATTRIBUTE_OPTION_COLOR_HEX.slate;
}

/** Required option picker styled like product attribute option chips. */
export function OrderProductOptionChips({
  variants,
  selectedId,
  attributes,
  onSelect,
  error,
}: Props) {
  if (variants.length === 0) return null;

  return (
    <div className="mt-2 space-y-1.5 px-1 pb-1">
      <Label className="text-caption text-foreground-secondary">
        Option <span className="text-error">*</span>
      </Label>
      <div className="flex flex-wrap gap-2">
        {variants.map((v) => {
          const active = selectedId === v.id;
          const color = colorForLabel(v.option_key, attributes);
          return (
            <button
              key={v.id}
              type="button"
              onClick={() => onSelect(v)}
              className={`inline-flex items-center gap-2 rounded-md border px-2.5 py-1.5 text-sm transition-colors ${
                active
                  ? "border-ink bg-surface-active text-foreground"
                  : "border-border bg-background text-foreground-secondary hover:bg-background-subtle"
              }`}
            >
              <span
                className="inline-block h-2.5 w-2.5 shrink-0 rounded-full"
                style={{ backgroundColor: color }}
                aria-hidden
              />
              <span>{v.option_key || "Default"}</span>
              <span className="tabular-nums text-caption text-foreground-muted">
                {formatIDR(v.effective_price_idr)}
              </span>
            </button>
          );
        })}
      </div>
      {error ? <p className="text-caption text-error">{error}</p> : null}
    </div>
  );
}
