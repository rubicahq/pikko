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
import { WishlistBadge } from "@/components/storefront/wishlist-badge";
import { WishlistButton } from "@/components/storefront/wishlist-button";
import { formatIDR } from "@/lib/api";
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

type WishlistPick = {
  product_id: string;
  product_name: string;
  base_price_idr?: number | null;
  compare_at_price_idr?: number | null;
  contact_for_price?: boolean;
};

function PriceBlock({
  contact,
  base,
  compareAt,
}: {
  contact?: boolean;
  base: number;
  compareAt?: number | null;
}) {
  if (contact) {
    return <span className="font-medium text-foreground">Hubungi Penjual</span>;
  }
  if (compareAt != null && compareAt > base) {
    return (
      <span className="inline-flex flex-wrap items-baseline gap-2">
        <span className="font-medium text-foreground">{formatIDR(base)}</span>
        <span className="text-xs text-foreground-muted line-through">{formatIDR(compareAt)}</span>
      </span>
    );
  }
  return <span className="font-medium text-foreground">{formatIDR(base)}</span>;
}

/**
 * Catalog flipbook etalase. With `liveSlug`, restores hotspot hover outlines +
 * scheduled-live wishlist affordances. Without live, stays display-only (no cart/AI).
 */
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
  const wishlistMode = Boolean(liveSlug);
  const [index, setIndex] = useState(0);
  const [zoom, setZoom] = useState(1);
  const [showThumbs, setShowThumbs] = useState(false);
  const [reducedMotion, setReducedMotion] = useState(false);
  const [flipping, setFlipping] = useState<"next" | "prev" | null>(null);
  const [picked, setPicked] = useState<WishlistPick | null>(null);
  const touchX = useRef<number | null>(null);

  useEffect(() => {
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");
    setReducedMotion(mq.matches);
    const onChange = () => setReducedMotion(mq.matches);
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, []);

  const page = pages[index];

  const pageItems = useMemo((): FlipPageItem[] => {
    if (!wishlistMode || !page) return [];
    if (page.items && page.items.length > 0) return page.items;
    const fromHs: FlipPageItem[] = [];
    const seen = new Set<string>();
    for (const hs of page.hotspots || []) {
      if (!hs.product_id || !hs.product_name || seen.has(hs.product_id)) continue;
      seen.add(hs.product_id);
      fromHs.push({
        product_id: hs.product_id,
        product_slug: hs.product_slug || "",
        product_name: hs.product_name,
        base_price_idr: hs.base_price_idr ?? 0,
        compare_at_price_idr: hs.compare_at_price_idr,
        contact_for_price: hs.contact_for_price,
        primary_image_url: hs.primary_image_url,
        hotspot_id: hs.id,
      });
    }
    return fromHs;
  }, [page, wishlistMode]);

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
      if (e.key === "Escape") setPicked(null);
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

  function openWishlistPick(hs: FlipHotspot | FlipPageItem) {
    if (!wishlistMode || !("product_id" in hs) || !hs.product_id) return;
    const name =
      "product_name" in hs && hs.product_name
        ? hs.product_name
        : "Product";
    setPicked({
      product_id: hs.product_id,
      product_name: name,
      base_price_idr: hs.base_price_idr,
      compare_at_price_idr: hs.compare_at_price_idr,
      contact_for_price: hs.contact_for_price,
    });
  }

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
          {liveSlug ? <WishlistBadge storeSlug={storeSlug} liveSlug={liveSlug} color="#15191E" /> : null}
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

      <div
        className={cn(
          "mx-auto w-full flex-1",
          wishlistMode
            ? "grid max-w-6xl grid-cols-1 gap-0 lg:h-[calc(100dvh-65px)] lg:min-h-0 lg:flex-none lg:grid-cols-[minmax(0,1.4fr)_minmax(280px,0.7fr)]"
            : "flex max-w-5xl flex-col",
        )}
      >
        <section
          className={cn(
            "relative flex flex-col items-center justify-center overflow-hidden p-4",
            wishlistMode ? "lg:border-r lg:border-border" : "flex-1",
          )}
        >
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
            className={cn(
              "absolute right-3 z-10 hidden sm:inline-flex",
              wishlistMode && "lg:right-[calc(0%+12px)]",
            )}
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
            <img
              src={page.image_url}
              alt=""
              className={cn(
                "block h-auto w-auto max-w-full select-none",
                wishlistMode ? "max-h-[70dvh] lg:max-h-[calc(100dvh-120px)]" : "max-h-[78dvh]",
              )}
              draggable={false}
            />
            {wishlistMode &&
              (page.hotspots || [])
                .filter((hs) => !hs.list_only)
                .map((hs) => (
                  <button
                    key={hs.id}
                    type="button"
                    className="group absolute min-h-[28px] min-w-[28px] border border-transparent bg-transparent transition-colors hover:border-brand hover:bg-brand/10 focus-visible:border-brand focus-visible:outline focus-visible:outline-2 focus-visible:outline-brand-dark"
                    style={{
                      left: `${hs.x_pct}%`,
                      top: `${hs.y_pct}%`,
                      width: `${Math.max(hs.width_pct, 4)}%`,
                      height: `${Math.max(hs.height_pct, 4)}%`,
                    }}
                    aria-label={hs.product_name ? `Wishlist ${hs.product_name}` : "Add to wishlist"}
                    onClick={() => openWishlistPick(hs)}
                  >
                    {hs.product_name && (
                      <span className="pointer-events-none absolute bottom-full left-1/2 z-20 mb-1 hidden w-max max-w-[min(16rem,80vw)] -translate-x-1/2 rounded-md bg-foreground px-2 py-1 text-xs text-background shadow-md group-hover:block group-focus-visible:block">
                        {hs.product_name} · Wishlist
                      </span>
                    )}
                  </button>
                ))}
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

        {wishlistMode && liveSlug ? (
          <aside className="flex min-h-0 flex-col border-t border-border bg-background lg:overflow-hidden lg:border-t-0">
            <div className="shrink-0 border-b border-border px-4 py-3">
              <h2 className="text-sm font-semibold text-foreground">Items on this page</h2>
              <p className="text-xs text-foreground-secondary">
                Page {index + 1} of {pages.length}
              </p>
            </div>
            <ul className="min-h-0 flex-1 space-y-0 overflow-y-auto overscroll-contain p-2" aria-live="polite">
              {pageItems.length === 0 && (
                <li className="px-3 py-8 text-center text-sm text-foreground-secondary">
                  Tidak ada item di halaman ini
                </li>
              )}
              {pageItems.map((it) => (
                <li
                  key={it.product_id}
                  className="rounded-md border border-transparent p-3 hover:border-border hover:bg-surface-hover"
                >
                  <div className="flex gap-3">
                    <div className="h-14 w-14 shrink-0 overflow-hidden rounded-md border border-border bg-surface">
                      {it.primary_image_url ? (
                        // eslint-disable-next-line @next/next/no-img-element
                        <img src={urls.api(it.primary_image_url)} alt="" className="h-full w-full object-cover" />
                      ) : null}
                    </div>
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-sm font-medium text-foreground">{it.product_name}</p>
                      <div className="mt-0.5 text-sm">
                        <PriceBlock
                          contact={it.contact_for_price}
                          base={it.base_price_idr}
                          compareAt={it.compare_at_price_idr}
                        />
                      </div>
                      <div className="mt-2">
                        <WishlistButton
                          storeSlug={storeSlug}
                          liveSlug={liveSlug}
                          productId={it.product_id}
                          productName={it.product_name}
                          primaryColor="#C83200"
                          buttonFg="#FFFFFF"
                          enabled
                        />
                      </div>
                    </div>
                  </div>
                </li>
              ))}
            </ul>
          </aside>
        ) : null}
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

      {picked && liveSlug && (
        <div
          className="fixed inset-0 z-50 flex items-end justify-center bg-dark/40 p-4 sm:items-center"
          role="dialog"
          aria-modal="true"
          aria-label={picked.product_name}
          onClick={() => setPicked(null)}
        >
          <div
            className="w-full max-w-md rounded-lg border border-border bg-background p-5"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between gap-3">
              <div>
                <h2 className="text-lg font-medium text-foreground">{picked.product_name}</h2>
                <p className="mt-1 text-sm text-foreground-secondary">
                  <PriceBlock
                    contact={picked.contact_for_price}
                    base={picked.base_price_idr ?? 0}
                    compareAt={picked.compare_at_price_idr}
                  />
                </p>
              </div>
              <button type="button" aria-label="Close" onClick={() => setPicked(null)}>
                <X className="h-5 w-5" strokeWidth={1.75} />
              </button>
            </div>
            <div className="mt-5">
              <WishlistButton
                storeSlug={storeSlug}
                liveSlug={liveSlug}
                productId={picked.product_id}
                productName={picked.product_name}
                primaryColor="#C83200"
                buttonFg="#FFFFFF"
                enabled
              />
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
