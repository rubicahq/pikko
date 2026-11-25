"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ChevronLeft,
  ChevronRight,
  List,
  Minus,
  Plus,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";
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
  list_only?: boolean;
  base_price_idr?: number | null;
  compare_at_price_idr?: number | null;
  contact_for_price?: boolean;
  primary_image_url?: string;
};

export type FlipPageItem = {
  product_id: string;
  product_slug: string;
  product_name: string;
  base_price_idr: number;
  compare_at_price_idr?: number | null;
  contact_for_price?: boolean;
  primary_image_url?: string;
  hotspot_id?: string;
};

export type FlipPage = {
  id: string;
  image_url: string;
  display_order: number;
  hotspots?: FlipHotspot[];
  items?: FlipPageItem[];
};

/** Display-only PDF etalase — page images + navigation. No cart / wishlist / contact CTAs. */
export function FlipbookViewer({
  storeSlug,
  storeName,
  pages,
  liveSlug,
}: {
  storeSlug: string;
  storeName: string;
  pages: FlipPage[];
  storeWhatsapp?: string | null;
  liveSlug?: string;
}) {
  const [index, setIndex] = useState(0);
  const [zoom, setZoom] = useState(1);
  const [showThumbs, setShowThumbs] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(false);
  const [flipping, setFlipping] = useState<"next" | "prev" | null>(null);
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
      if (e.key === "+" || e.key === "=") setZoom((z) => Math.min(2.5, z + 0.25));
      if (e.key === "-") setZoom((z) => Math.max(1, z - 0.25));
    }
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [go]);

  const title = useMemo(
    () => `${storeName} catalog — page ${index + 1} of ${pages.length}`,
    [storeName, index, pages.length],
  );

  if (!pages.length) {
    return (
      <main className="mx-auto flex min-h-[60vh] max-w-reading flex-col items-start justify-center px-4 py-16">
        <p className="text-sm text-foreground-secondary">
          <a href={urls.storefrontPath(storeSlug)} className="hover:text-foreground">
            {storeName}
          </a>
          <span className="mx-1.5">/</span>
          Catalog
        </p>
        <h1 className="mt-3 text-2xl font-semibold text-foreground">Catalog unavailable</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          This catalog has no published pages. Browse the store for available products.
        </p>
        <a
          href={urls.storefrontPath(storeSlug)}
          className="mt-6 text-sm font-medium text-brand-dark underline-offset-2 hover:underline"
        >
          Back to store
        </a>
      </main>
    );
  }

  return (
    <div className="flex min-h-screen flex-col bg-background-subtle">
      <header className="flex items-center justify-between gap-3 border-b border-border bg-background px-4 py-3">
        <div className="min-w-0">
          <a
            href={liveSlug ? urls.storefrontLive(storeSlug, liveSlug) : urls.storefrontPath(storeSlug)}
            className="truncate text-sm font-semibold text-foreground hover:text-brand-dark"
          >
            {storeName}
          </a>
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

      <div className="mx-auto flex w-full max-w-5xl flex-1 flex-col">
        <section className="relative flex flex-1 flex-col items-center justify-center overflow-hidden p-4">
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
              "relative w-fit max-w-full origin-center bg-background shadow-none transition-transform duration-300",
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
            <img src={page.image_url} alt="" className="block h-auto max-h-[78dvh] w-auto max-w-full select-none" draggable={false} />
          </div>

          <div className="mt-3 flex items-center justify-center gap-3 sm:hidden">
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
        </section>
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
    </div>
  );
}
