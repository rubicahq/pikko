"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import {
  apiFetch,
  ATTRIBUTE_OPTION_COLOR_HEX,
  ATTRIBUTE_OPTION_COLORS,
  type AttributeDefinition,
  type AttributeOptionColor,
  type AttributeType,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { urls } from "@/lib/urls";

type DraftOption = {
  id?: string;
  label: string;
  color: AttributeOptionColor;
};

const TYPE_LABELS: Record<AttributeType, string> = {
  single_option: "Single option",
  multiple_option: "Multiple options",
  short_text: "Short text",
  number: "Number",
};

function isOptionType(t: AttributeType) {
  return t === "single_option" || t === "multiple_option";
}

export function AttributeForm({
  storeId,
  attribute,
}: {
  storeId: string;
  attribute?: AttributeDefinition | null;
}) {
  const router = useRouter();
  const initialType = (attribute?.type || attribute?.data_type || "short_text") as AttributeType;
  const [name, setName] = useState(attribute?.name || attribute?.label || "");
  const [type, setType] = useState<AttributeType>(
    ["single_option", "multiple_option", "short_text", "number"].includes(initialType)
      ? initialType
      : "short_text",
  );
  const [options, setOptions] = useState<DraftOption[]>(
    attribute?.options?.length
      ? attribute.options.map((o) => ({
          id: o.id,
          label: o.label,
          color: (ATTRIBUTE_OPTION_COLORS.includes(o.color as AttributeOptionColor)
            ? o.color
            : "slate") as AttributeOptionColor,
        }))
      : [{ label: "", color: "coral" }],
  );
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  function updateOption(i: number, patch: Partial<DraftOption>) {
    setOptions((prev) => prev.map((o, idx) => (idx === i ? { ...o, ...patch } : o)));
  }

  function addOption() {
    setOptions((prev) => [
      ...prev,
      {
        label: "",
        color: ATTRIBUTE_OPTION_COLORS[prev.length % ATTRIBUTE_OPTION_COLORS.length],
      },
    ]);
  }

  function removeOption(i: number) {
    setOptions((prev) => (prev.length <= 1 ? prev : prev.filter((_, idx) => idx !== i)));
  }

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    const trimmed = name.trim();
    if (!trimmed) {
      setError("Name is required.");
      setSaving(false);
      return;
    }
    const body: Record<string, unknown> = {
      name: trimmed,
      type,
    };
    if (isOptionType(type)) {
      const cleaned = options
        .map((o) => ({
          id: o.id,
          label: o.label.trim(),
          color: o.color,
        }))
        .filter((o) => o.label);
      if (cleaned.length === 0) {
        setError("Add at least one option with a label.");
        setSaving(false);
        return;
      }
      body.options = cleaned;
    }
    try {
      if (attribute) {
        await apiFetch(`/api/v1/stores/${storeId}/attributes/${attribute.id}`, {
          method: "PATCH",
          body,
        });
      } else {
        await apiFetch(`/api/v1/stores/${storeId}/attributes`, {
          method: "POST",
          body,
        });
      }
      router.push(urls.storeAttributes(storeId));
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save attribute.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="mx-auto max-w-reading space-y-6">
      <div>
        <h1 className="text-[30px] font-medium text-foreground">
          {attribute ? "Edit attribute" : "Add attribute"}
        </h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Custom fields for products — brand, category, size, or anything else you need.
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="attr-name">Name</Label>
        <Input
          id="attr-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder="e.g. Brand"
          required
        />
      </div>

      <div className="space-y-2">
        <Label htmlFor="attr-type">Type</Label>
        <select
          id="attr-type"
          className="h-11 w-full rounded-[10px] border border-border bg-background px-3 text-sm"
          value={type}
          disabled={!!attribute}
          onChange={(e) => setType(e.target.value as AttributeType)}
        >
          {(Object.keys(TYPE_LABELS) as AttributeType[]).map((t) => (
            <option key={t} value={t}>
              {TYPE_LABELS[t]}
            </option>
          ))}
        </select>
        {attribute ? (
          <p className="text-xs text-foreground-muted">Type cannot be changed after creation.</p>
        ) : null}
      </div>

      {isOptionType(type) ? (
        <div className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <Label>Options</Label>
            <Button type="button" variant="outline" size="sm" onClick={addOption}>
              Add option
            </Button>
          </div>
          <ul className="space-y-3">
            {options.map((opt, i) => (
              <li
                key={i}
                className="flex flex-wrap items-end gap-3 rounded-[14px] border border-border bg-background px-3 py-3"
              >
                <div className="min-w-[160px] flex-1 space-y-2">
                  <Label htmlFor={`opt-label-${i}`}>Label</Label>
                  <Input
                    id={`opt-label-${i}`}
                    value={opt.label}
                    onChange={(e) => updateOption(i, { label: e.target.value })}
                    placeholder="Option label"
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label>Color</Label>
                  <div className="flex flex-wrap gap-2" role="radiogroup" aria-label="Option color">
                    {ATTRIBUTE_OPTION_COLORS.map((c) => {
                      const selected = opt.color === c;
                      return (
                        <button
                          key={c}
                          type="button"
                          role="radio"
                          aria-checked={selected}
                          title={c}
                          onClick={() => updateOption(i, { color: c })}
                          className={`h-8 w-8 rounded-[10px] border-2 transition ${
                            selected ? "border-foreground scale-105" : "border-transparent"
                          }`}
                          style={{ backgroundColor: ATTRIBUTE_OPTION_COLOR_HEX[c] }}
                        />
                      );
                    })}
                  </div>
                </div>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => removeOption(i)}
                  disabled={options.length <= 1}
                >
                  Remove
                </Button>
              </li>
            ))}
          </ul>
        </div>
      ) : null}

      {error ? <p className="text-sm text-error">{error}</p> : null}

      <div className="flex flex-wrap gap-3">
        <Button type="submit" loading={saving}>
          Save attribute
        </Button>
        <Button
          type="button"
          variant="outline"
          onClick={() => router.push(urls.storeAttributes(storeId))}
        >
          Cancel
        </Button>
      </div>
    </form>
  );
}
