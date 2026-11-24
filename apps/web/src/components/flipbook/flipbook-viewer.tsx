"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  List,
  Minus,
  Plus,
  X,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";
import { addToCart } from "@/lib/cart";
import { cn } from "@/lib/utils";

export type FlipHotspot = {
  id: string;
  product_id?: string | null;
  product_slug?: string;
  product_name?: string;
  x_pct: number;
  y_pct: number;
  width_pct: number;
  height_pct: number;
};

export type FlipPage = {
  id: string;
  image_url: string;
  display_order: number;
  hotspots?: FlipHotspot[];
};

type ProductCard = {
  id: string;
  name: string;
  slug: string;
  base_price_idr: number;
  description?: string | null;
  variants?: { id: string; effective_price_idr: number; options?: Record<string, string> }[];
  primary_image_url?: string;
};

function formatIDR(n: number) {
  return new Intl.NumberFormat("id-ID", {
    style: "currency",
    currency: "IDR",
    maximumFractionDigits: 0,
  }).format(n);
}

export function FlipbookViewer({
  storeSlug,
  storeName,
  pages,
}: {
  storeSlug: string;
  storeName: string;
  pages: FlipPage[];
}) {
  const [index, setIndex] = useState(0);
  const [zoom, setZoom] = useState(1);
  const [showThumbs, setShowThumbs] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(false);
  const [flipping, setFlipping] = useState<"next" | "prev" | null>(null);
  const [product, setProduct] = useState<ProductCard | null>(null);
  const [productURL, setProductURL] = useState<string | null>(null);
  const [variantId, setVariantId] = useState<string | null>(null);
  const [cartMsg, setCartMsg] = useState<string | null>(null);
  const touchX = useRef<number | null>(null);

  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    setReducedMotion(mq.matches);
    const onChange = () => setReducedMotion(mq.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);

  const page = pages[index];

  const go = useCallback(
    (dir: "next" | "prev") => {
      setIndex((i) => {
        const next = dir === "next" ? i + 1 : i - 1;
        if (next < 0 || next >= pages.length) return i;
        if (!reducedMotion) {
          setFlipping(dir);
          window.setTimeout(() => setFlipping(null), 380);
        }
        return next;
      });
    },
    [pages.length, reducedMotion],
  );

  useEffect(() => {
    function onKey(e: KeyboardEvent) {
      if (e.key === "ArrowRight") go("next");
      if (e.key === "ArrowLeft") go("prev");
      if (e.key === "Escape") setProduct(null);
      if (e.key === "+" || e.key === "=") setZoom((z) => Math.min(2.5, z + 0.25));
      if (e.key === "-") setZoom((z) => Math.max(1, z - 0.25));
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [go]);

  async function openHotspot(hs: FlipHotspot) {
    if (!hs.product_id) return;
    const res = await fetch(
      urls.api(`/api/v1/public/stores/${storeSlug}/catalog/products/${hs.product_id}`),
    );
    if (!res.ok) return;
    const data = await res.json();
    setProduct(data.product);
    setProductURL(data.link_ref?.product_url || urls.storefrontProduct(storeSlug, data.product.slug));
    setVariantId(data.product.variants?.[0]?.id || null);
    setCartMsg(null);
  }

  function handleAdd() {
    if (!product) return;
    const variant = product.variants?.find((v) => v.id === variantId);
    addToCart({
      storeSlug,
      productId: product.id,
      productSlug: product.slug,
      name: product.name,
      priceIdr: variant?.effective_price_idr ?? product.base_price_idr,
      variantId: variantId,
      quantity: 1,
    });
    setCartMsg("Added to cart (local). Full checkout lands in a later phase — view product for details.");
  }

  const title = useMemo(
    () => `${storeName} catalog — page ${index + 1} of ${pages.length}`,
    [storeName, index, pages.length],
  );

  if (!pages.length) {
    return (
      <p className="p-8 text-center text-sm text-foreground-secondary">
        This catalog has no published pages yet.
      </p>
    );
  }

  return (
    <div className="flex min-h-screen flex-col bg-background-subtle">
      <header className="flex items-center justify-between gap-3 border-b border-border bg-background px-4 py-3">
        <div className="min-w-0">
          <p className="truncate text-sm font-medium text-foreground">{storeName}</p>
          <p className="text-xs text-foreground-secondary">{title}</p>
        </div>
        <div className="flex items-center gap-1">
          <Button type="button" variant="ghost" size="icon" aria-label="Zoom out" onClick={() => setZoom((z) => Math.max(1, z - 0.25))}>
            <Minus className="h-4 w-4" strokeWidth={1.75} />
          </Button>
          <Button type="button" variant="ghost" size="icon" aria-label="Zoom in" onClick={() => setZoom((z) => Math.min(2.5, z + 0.25))}>
            <Plus className="h-4 w-4" strokeWidth={1.75} />
          </Button>
          <Button type="button" variant="ghost" size="icon" aria-label="Page list" onClick={() => setShowThumbs((v) => !v)}>
            <List className="h-4 w-4" strokeWidth={1.75} />
          </Button>
        </div>
      </header>

      <div className="relative flex flex-1 items-center justify-center overflow-hidden p-4">
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="absolute left-3 z-10 hidden sm:inline-flex"
          aria-label="Previous page"
          disabled={index === 0}
          onClick={() => go("prev")}
        >
          <ChevronLeft className="h-5 w-5" strokeWidth={1.75} />
        </Button>
        <Button
          type="button"
          variant="outline"
          size="icon"
          className="absolute right-3 z-10 hidden sm:inline-flex"
          aria-label="Next page"
          disabled={index >= pages.length - 1}
          onClick={() => go("next")}
        >
          <ChevronRight className="h-5 w-5" strokeWidth={1.75} />
        </Button>

        <div
          className={cn(
            "relative max-h-[80vh] w-full max-w-3xl origin-center border border-border bg-background shadow-none transition-transform duration-300",
            !reducedMotion && flipping && "opacity-90",
          )}
          style={{
            transform: `scale(${zoom}) ${
              !reducedMotion && flipping === "next"
                ? "perspective(1200px) rotateY(-6deg)"
                : !reducedMotion && flipping === "prev"
                  ? "perspective(1200px) rotateY(6deg)"
                  : ""
            }`,
          }}
          onTouchStart={(e) => {
            touchX.current = e.changedTouches[0]?.clientX ?? null;
          }}
          onTouchEnd={(e) => {
            if (touchX.current == null) return;
            const dx = (e.changedTouches[0]?.clientX ?? 0) - touchX.current;
            touchX.current = null;
            if (dx < -40) go("next");
            if (dx > 40) go("prev");
          }}
          role="img"
          aria-label={title}
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img src={page.image_url} alt="" className="h-auto w-full select-none" draggable={false} />
          {(page.hotspots || []).map((hs) => (
            <button
              key={hs.id}
              type="button"
              className="absolute border border-brand/70 bg-brand/10 transition-colors hover:bg-brand/20 focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-dark"
              style={{
                left: `${hs.x_pct}%`,
                top: `${hs.y_pct}%`,
                width: `${hs.width_pct}%`,
                height: `${hs.height_pct}%`,
              }}
              aria-label={hs.product_name ? `Open ${hs.product_name}` : "Open product"}
              onClick={() => openHotspot(hs)}
            />
          ))}
        </div>
      </div>

      <div className="flex items-center justify-center gap-3 border-t border-border bg-background px-4 py-3 sm:hidden">
        <Button type="button" variant="outline" size="sm" disabled={index === 0} onClick={() => go("prev")}>
          Previous
        </Button>
        <span className="text-sm text-foreground-secondary">
          {index + 1} / {pages.length}
        </span>
        <Button type="button" variant="outline" size="sm" disabled={index >= pages.length - 1} onClick={() => go("next")}>
          Next
        </Button>
      </div>

      {showThumbs && (
        <div className="border-t border-border bg-background p-3">
          <ul className="flex gap-2 overflow-x-auto" aria-label="Page list">
            {pages.map((p, i) => (
              <li key={p.id}>
                <button
                  type="button"
                  className={cn(
                    "h-20 w-14 overflow-hidden border border-border",
                    i === index && "border-brand",
                  )}
                  onClick={() => setIndex(i)}
                  aria-label={`Go to page ${i + 1}`}
                  aria-current={i === index}
                >
                  {/* eslint-disable-next-line @next/next/no-img-element */}
                  <img src={p.image_url} alt="" className="h-full w-full object-cover" />
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}

      {product && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-dark/40 p-4 sm:items-center"
          role="dialog"
          aria-modal="true"
          aria-label={product.name}
          onClick={() => setProduct(null)}
        >
          <div
            className="w-full max-w-md rounded-lg border border-border bg-background p-5"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between gap-3">
              <div>
                <h2 className="text-lg font-medium text-foreground">{product.name}</h2>
                <p className="mt-1 text-sm text-foreground-secondary">
                  {formatIDR(
                    product.variants?.find((v) => v.id === variantId)?.effective_price_idr ??
                      product.base_price_idr,
                  )}
                </p>
              </div>
              <button type="button" aria-label="Close" onClick={() => setProduct(null)}>
                <X className="h-5 w-5" strokeWidth={1.75} />
              </button>
            </div>
            {product.description && (
              <p className="mt-3 text-sm text-foreground-secondary">{product.description}</p>
            )}
            {product.variants && product.variants.length > 0 && (
              <div className="mt-4 space-y-2">
                <p className="text-xs font-medium text-foreground">Variant</p>
                <select
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
                  value={variantId || ""}
                  onChange={(e) => setVariantId(e.target.value)}
                >
                  {product.variants.map((v) => (
                    <option key={v.id} value={v.id}>
                      {Object.values(v.options || {}).join(" / ") || "Default"} —{" "}
                      {formatIDR(v.effective_price_idr)}
                    </option>
                  ))}
                </select>
              </div>
            )}
            <div className="mt-5 flex flex-col gap-2 sm:flex-row">
              <Button type="button" onClick={handleAdd}>
                Add to cart
              </Button>
              {productURL && (
                <Button type="button" variant="outline" asChild>
                  <a href={productURL}>View product</a>
                </Button>
              )}
            </div>
            {cartMsg && <p className="mt-3 text-xs text-foreground-secondary">{cartMsg}</p>}
          </div>
        </div>
      )}

    </div>
  );
}
