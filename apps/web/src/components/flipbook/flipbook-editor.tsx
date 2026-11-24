"use client";

import { useCallback, useEffect, useRef, useState } from "react";
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

function clamp(n: number, min: number, max: number) {
  return Math.max(min, Math.min(max, n));
}

export function FlipbookEditor({ storeId, storeSlug }: { storeId: string; storeSlug: string }) {
  const [pages, setPages] = useState<Page[]>([]);
  const [products, setProducts] = useState<ProductOpt[]>([]);
  const [selectedPage, setSelectedPage] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [productId, setProductId] = useState<string>("");
  const [dragging, setDragging] = useState<string | null>(null);
  const pageRef = useRef<HTMLDivElement | null>(null);

  const load = useCallback(async () => {
    const [pRes, prodRes] = await Promise.all([
      fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages`), { credentials: "include" }),
      fetch(urls.api(`/api/v1/stores/${storeId}/products?limit=100`), { credentials: "include" }),
    ]);
    if (pRes.ok) {
      const data = await pRes.json();
      setPages(data.pages || []);
      setSelectedPage((prev) => prev || data.pages?.[0]?.id || null);
    }
    if (prodRes.ok) {
      const data = await prodRes.json();
      setProducts((data.products || []).map((p: ProductOpt) => ({ id: p.id, name: p.name, status: p.status })));
    }
  }, [storeId]);

  useEffect(() => {
    load();
  }, [load]);

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
    const width = 10;
    const height = 10;
    const xPct = clamp(x, 0, 100 - width);
    const yPct = clamp(y, 0, 100 - height);
    const res = await fetch(
      urls.api(`/api/v1/stores/${storeId}/catalog-pages/${page.id}/hotspots`),
      {
        method: "POST",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          product_id: productId,
          x_pct: xPct,
          y_pct: yPct,
          width_pct: width,
          height_pct: height,
        }),
      },
    );
    if (!res.ok) {
      const data = await res.json().catch(() => null);
      setError(data?.error?.message || "Unable to create hotspot.");
      return;
    }
    setMessage("Hotspot added.");
    setError(null);
    await load();
  }

  async function updateHotspot(hs: Hotspot, x: number, y: number) {
    const xPct = clamp(x, 0, 100 - hs.width_pct);
    const yPct = clamp(y, 0, 100 - hs.height_pct);
    setPages((prev) =>
      prev.map((p) =>
        p.id !== page?.id
          ? p
          : {
              ...p,
              hotspots: (p.hotspots || []).map((h) =>
                h.id === hs.id ? { ...h, x_pct: xPct, y_pct: yPct } : h,
              ),
            },
      ),
    );
    await fetch(urls.api(`/api/v1/stores/${storeId}/catalog-pages/hotspots/${hs.id}`), {
      method: "PATCH",
      credentials: "include",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        product_id: hs.product_id,
        x_pct: xPct,
        y_pct: yPct,
        width_pct: hs.width_pct,
        height_pct: hs.height_pct,
      }),
    });
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
                  Click the page to place a hotspot. Drag an existing hotspot to reposition it.
                </p>
              </div>
              <div
                ref={pageRef}
                className="relative max-w-2xl border border-border bg-background"
                onClick={(e) => {
                  if (dragging) return;
                  const rect = e.currentTarget.getBoundingClientRect();
                  const x = ((e.clientX - rect.left) / rect.width) * 100 - 5;
                  const y = ((e.clientY - rect.top) / rect.height) * 100 - 5;
                  createHotspot(x, y);
                }}
              >
                {/* eslint-disable-next-line @next/next/no-img-element */}
                <img src={page.image_url} alt="" className="w-full select-none" draggable={false} />
                {(page.hotspots || []).map((hs) => (
                  <div
                    key={hs.id}
                    className="absolute cursor-grab border border-brand bg-brand/15 active:cursor-grabbing"
                    style={{
                      left: `${hs.x_pct}%`,
                      top: `${hs.y_pct}%`,
                      width: `${hs.width_pct}%`,
                      height: `${hs.height_pct}%`,
                      minWidth: 24,
                      minHeight: 24,
                    }}
                    onClick={(e) => e.stopPropagation()}
                    onPointerDown={(e) => {
                      e.stopPropagation();
                      e.preventDefault();
                      setDragging(hs.id);
                      const target = pageRef.current;
                      if (!target) return;
                      const onMove = (ev: PointerEvent) => {
                        const rect = target.getBoundingClientRect();
                        const x = ((ev.clientX - rect.left) / rect.width) * 100 - hs.width_pct / 2;
                        const y = ((ev.clientY - rect.top) / rect.height) * 100 - hs.height_pct / 2;
                        setPages((prev) =>
                          prev.map((p) =>
                            p.id !== page.id
                              ? p
                              : {
                                  ...p,
                                  hotspots: (p.hotspots || []).map((h) =>
                                    h.id === hs.id
                                      ? {
                                          ...h,
                                          x_pct: clamp(x, 0, 100 - h.width_pct),
                                          y_pct: clamp(y, 0, 100 - h.height_pct),
                                        }
                                      : h,
                                  ),
                                },
                          ),
                        );
                      };
                      const onUp = (ev: PointerEvent) => {
                        window.removeEventListener("pointermove", onMove);
                        window.removeEventListener("pointerup", onUp);
                        setDragging(null);
                        const rect = target.getBoundingClientRect();
                        const x = ((ev.clientX - rect.left) / rect.width) * 100 - hs.width_pct / 2;
                        const y = ((ev.clientY - rect.top) / rect.height) * 100 - hs.height_pct / 2;
                        updateHotspot(hs, x, y);
                      };
                      window.addEventListener("pointermove", onMove);
                      window.addEventListener("pointerup", onUp);
                    }}
                  >
                    <button
                      type="button"
                      className="absolute right-0 top-0 bg-background px-1 text-xs text-error"
                      onClick={(e) => {
                        e.stopPropagation();
                        deleteHotspot(hs.id);
                      }}
                    >
                      ×
                    </button>
                    <span className="block truncate px-1 text-[10px] text-foreground">
                      {hs.product_name || "Product"}
                    </span>
                  </div>
                ))}
              </div>
              <p className="text-xs text-foreground-secondary">
                {(page.hotspots || []).length} hotspot(s) on this page. Publish products before they
                appear on the public flipbook.
              </p>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
