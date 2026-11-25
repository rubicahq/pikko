"use client";

import { useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { apiFetch, formatIDR, type AttributeDefinition, type Product } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { StatusBadge } from "@/components/ui/status-badge";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";

export function ProductsTable({ storeId, products, attributes }: { storeId: string; products: Product[]; attributes: AttributeDefinition[] }) {
  const router = useRouter();
  const [selected, setSelected] = useState<string[]>([]);
  const [action, setAction] = useState("published");
  const [attributeId, setAttributeId] = useState("");
  const [value, setValue] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const attribute = attributes.find((item) => item.id === attributeId);
  const optionType = attribute?.type === "single_option" || attribute?.type === "multiple_option";

  function toggle(id: string) {
    setSelected((current) => current.includes(id) ? current.filter((item) => item !== id) : [...current, id]);
  }

  async function apply() {
    setBusy(true);
    setError("");
    try {
      for (const id of selected) {
        if (action === "attribute") {
          if (!attribute) throw new Error("Choose an attribute.");
          const entry = { attribute_id: attribute.id, option_ids: optionType ? [value] : [], value_text: optionType || attribute.type === "number" ? null : value, value_number: attribute.type === "number" ? Number(value) : null };
          await apiFetch(`/api/v1/stores/${storeId}/products/${id}/attributes`, { method: "PUT", body: entry });
        } else {
          await apiFetch(`/api/v1/stores/${storeId}/products/${id}/status`, { body: { status: action } });
        }
      }
      setSelected([]);
      router.refresh();
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : "Bulk action failed.");
    } finally {
      setBusy(false);
    }
  }

  return <div className="space-y-4">
    {selected.length > 0 && <div className="flex flex-wrap items-center gap-3 rounded-lg border border-border p-3">
      <span>{selected.length} selected</span>
      <select aria-label="Bulk action" value={action} onChange={(event) => setAction(event.target.value)} className="rounded border border-border bg-background p-2">
        <option value="published">Publish</option><option value="draft">Unpublish</option><option value="archived">Archive</option><option value="attribute">Set attribute</option>
      </select>
      {action === "attribute" && <>
        <select aria-label="Attribute" value={attributeId} onChange={(event) => { setAttributeId(event.target.value); setValue(""); }} className="rounded border border-border bg-background p-2">
          <option value="">Choose attribute</option>{attributes.map((item) => <option key={item.id} value={item.id}>{item.name || item.label}</option>)}
        </select>
        {optionType ? <select aria-label="Value" value={value} onChange={(event) => setValue(event.target.value)} className="rounded border border-border bg-background p-2"><option value="">Choose value</option>{attribute.options.map((option) => <option key={option.id} value={option.id}>{option.label}</option>)}</select> : <input aria-label="Value" type={attribute?.type === "number" ? "number" : "text"} value={value} onChange={(event) => setValue(event.target.value)} className="rounded border border-border bg-background p-2" />}
      </>}
      <Button disabled={busy || (action === "attribute" && (!attributeId || !value))} onClick={apply}>Apply</Button>
      {error && <p role="alert" className="text-error">{error} Selected products remain checked so you can retry.</p>}
    </div>}
    <Table stickyHeader><TableHeader><TableRow className="hover:bg-transparent"><TableHead><input type="checkbox" aria-label="Select all products on this page" checked={selected.length === products.length} onChange={(event) => setSelected(event.target.checked ? products.map((product) => product.id) : [])} /></TableHead><TableHead sticky>Product</TableHead><TableHead sticky>SKU</TableHead><TableHead sticky className="text-right">Price</TableHead><TableHead sticky>Status</TableHead></TableRow></TableHeader>
      <TableBody>{products.map((product) => <TableRow key={product.id}><TableCell><input type="checkbox" aria-label={`Select ${product.name}`} checked={selected.includes(product.id)} onChange={() => toggle(product.id)} /></TableCell><TableCell><Link href={urls.storeProduct(storeId, product.id)} className="font-medium text-foreground hover:underline">{product.name}</Link></TableCell><TableCell className="text-foreground-secondary">{product.sku || "—"}</TableCell><TableCell className="text-right tabular-nums">{formatIDR(product.base_price_idr)}</TableCell><TableCell><StatusBadge status={product.status} /></TableCell></TableRow>)}</TableBody></Table>
  </div>;
}
