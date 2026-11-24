"use client";

import { useCallback, useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { urls } from "@/lib/urls";

type Hotspot = {
  id: string;
  product_id?: string | null;
  product_name?: string;
  product_status?: string;
  x_pct: number;
  y_pct: number;
  width_pct: number;
  height_pct: number;
};

type Page = {
  id: string;
  image_url: string;
  display_order: number;
  status: string;
  page_number?: number | null;
  hotspots?: Hotspot[];
};

type ProductOpt = { id: string; name: string; status: string };

export function FlipbookEditor({ storeId, storeSlug }: { storeId: string; storeSlug: string }) {
  const [pages, setPages] = useState<Page[]>([]);
  const [products, setProducts] = useState<ProductOpt[]>([]);
  const [selectedPage, setSelectedPage] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [drag, setDrag] = useState<{ x: number; y: number } | null>(null);
  const [productId, setProductId] = useState<string>("");

  const load = useCallback(async () => {
    const [pRes, prodRes] = await Promise.all([
      fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages`), { credentials: "include" }),
      fetch(urls.api(`/api/v1/stores/${storeId}/products?limit=100`), { credentials: "include" }),
    ]);
    if (pRes.ok) {
      const data = await pRes.json();
      setPages(data.pages || []);
      if (!selectedPage && data.pages?.[0]) setSelectedPage(data.pages[0].id);
    }
    if (prodRes.ok) {
      const data = await prodRes.json();
      setProducts((data.products || []).map((p: ProductOpt) => ({ id: p.id, name: p.name, status: p.status })));
    }
  }, [storeId, selectedPage]);

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [storeId]);

  const page = pages.find((p) => p.id === selectedPage) || null;

  async function movePage(id: string, dir: -1 | 1) {
    const idx = pages.findIndex((p) => p.id === id);
    const j = idx + dir;
    if (idx < 0 || j < 0 || j >= pages.length) return;
    const next = [...pages];
    const [item] = next.splice(idx, 1);
    next.splice(j, 0, item);
    setPages(next);
    await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages/order`), {
      method: "PUT",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ page_ids: next.map((p) => p.id) }),
    });
  }

  async function setStatus(status: "draft" | "published") {
    setError(null);
    const res = await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages/status`), {
      method: "POST",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ status }),
    });
    if (!res.ok) {
      setError("Unable to update publication status.");
      return;
    }
    setMessage(status === "published" ? "Catalog pages published." : "Catalog pages set to draft.");
    await load();
  }

  async function createHotspot(x: number, y: number) {
    if (!page || !productId) {
      setError("Select a product before placing a hotspot.");
      return;
    }
    const res = await fetch(
      urls.api(`/api/v1/stores/${storeId}/catalog-pages/${page.id}/hotspots`),
      {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          product_id: productId,
          x_pct: Math.max(0, Math.min(90, x)),
          y_pct: Math.max(0, Math.min(90, y)),
          width_pct: 10,
          height_pct: 10,
        }),
      },
    );
    if (!res.ok) {
      setError("Unable to create hotspot.");
      return;
    }
    setMessage("Hotspot added.");
    await load();
  }

  async function deleteHotspot(id: string) {
    await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages/hotspots/${id}`), {
      method: "DELETE",
      credentials: "include",
    });
    await load();
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-2">
        <Button type="button" onClick={() => setStatus("published")}>
          Publish catalog
        </Button>
        <Button type="button" variant="outline" onClick={() => setStatus("draft")}>
          Unpublish
        </Button>
        <a
          className="text-sm text-brand-dark underline-offset-4 hover:underline"
          href={urls.storefrontCatalog(storeSlug)}
          target="_blank"
          rel="noreferrer"
        >
          Open public flipbook
        </a>
      </div>
      {message && <p className="text-sm text-success">{message}</p>}
      {error && <p className="text-sm text-error">{error}</p>}

      {pages.length === 0 ? (
        <p className="text-sm text-foreground-secondary">
          No catalog pages yet. Import a PDF to generate page images, then return here to place
          hotspots and publish.
        </p>
      ) : (
        <div className="grid gap-6 lg:grid-cols-[220px_1fr]">
          <ul className="space-y-2">
            {pages.map((p, i) => (
              <li key={p.id}>
                <button
                  type="button"
                  onClick={() => setSelectedPage(p.id)}
                  className={`flex w-full items-center gap-2 rounded-md border px-2 py-2 text-left text-sm ${
                    selectedPage === p.id ? "border-brand bg-brand-soft" : "border-border"
                  }`}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={p.image_url} alt="" className="h-12 w-9 object-cover" />
                  <span className="min-w-0 flex-1">
                    <span className="block truncate">Page {i + 1}</span>
                    <Badge tone={p.status === "published" ? "success" : "neutral"}>{p.status}</Badge>
                  </span>
                </button>
                <div className="mt-1 flex gap-1">
                  <Button type="button" size="sm" variant="ghost" onClick={() => movePage(p.id, -1)}>
                    Up
                  </Button>
                  <Button type="button" size="sm" variant="ghost" onClick={() => movePage(p.id, 1)}>
                    Down
                  </Button>
                </div>
              </li>
            ))}
          </ul>

          {page && (
            <div className="space-y-4">
              <div className="flex flex-wrap items-end gap-3">
                <label className="block text-sm">
                  <span className="mb-1 block text-foreground-secondary">Product for new hotspot</span>
                  <select
                    className="rounded-md border border-border bg-background px-3 py-2 text-sm"
                    value={productId}
                    onChange={(e) => setProductId(e.target.value)}
                  >
                    <option value="">Select product…</option>
                    {products.map((p) => (
                      <option key={p.id} value={p.id}>
                        {p.name} ({p.status})
                      </option>
                    ))}
                  </select>
                </label>
                <p className="text-xs text-foreground-secondary">
                  Click the page to place a 10%×10% hotspot. Drag existing hotspots via delete + re-add
                  for MVP positioning.
                </p>
              </div>
              <div
                className="relative max-w-2xl border border-border bg-background"
                onClick={(e) => {
                  const rect = e.currentTarget.getBoundingClientRect();
                  const x = ((e.clientX - rect.left) / rect.width) * 100 - 5;
                  const y = ((e.clientY - rect.top) / rect.height) * 100 - 5;
                  createHotspot(x, y);
                }}
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={page.image_url} alt="" className="w-full" draggable={false} />
                {(page.hotspots || []).map((hs) => (
                  <div
                    key={hs.id}
                    className="absolute border border-brand bg-brand/15"
                    style={{
                      left: `${hs.x_pct}%`,
                      top: `${hs.y_pct}%`,
                      width: `${hs.width_pct}%`,
                      height: `${hs.height_pct}%`,
                    }}
                    onClick={(e) => e.stopPropagation()}
                    onMouseDown={() => setDrag({ x: hs.x_pct, y: hs.y_pct })}
                  >
                    <button
                      type="button"
                      className="absolute right-0 top-0 bg-background px-1 text-xs text-error"
                      onClick={() => deleteHotspot(hs.id)}
                    >
                      ×
                    </button>
                    <span className="block truncate px-1 text-[10px] text-foreground">
                      {hs.product_name || "Product"}
                    </span>
                  </div>
                ))}
              </div>
              {drag && (
                <p className="text-xs text-foreground-secondary">
                  Hotspot selected at {drag.x.toFixed(0)}%, {drag.y.toFixed(0)}%.
                </p>
              )}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
