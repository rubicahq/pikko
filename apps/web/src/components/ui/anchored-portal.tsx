"use client";

import * as React from "react";
import { createPortal } from "react-dom";
import { cn } from "@/lib/utils";

type Props = {
  /** Element the list should sit under (typically the search input). */
  anchor: HTMLElement | null;
  open: boolean;
  children: React.ReactNode;
  className?: string;
  /** Called when the user clicks outside the panel and anchor. */
  onDismiss?: () => void;
  /** Max height in px for viewport flip math (matches max-h-56 by default). */
  maxHeight?: number;
};

/**
 * Renders children in a fixed panel under `anchor`, portaled to document.body
 * so table/overflow ancestors cannot clip the list.
 */
export function AnchoredPortal({
  anchor,
  open,
  children,
  className,
  onDismiss,
  maxHeight = 224,
}: Props) {
  const panelRef = React.useRef<HTMLDivElement | null>(null);
  const [mounted, setMounted] = React.useState(false);
  const [coords, setCoords] = React.useState<{
    top: number;
    left: number;
    width: number;
    maxHeight: number;
  } | null>(null);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  React.useLayoutEffect(() => {
    if (!open || !anchor) {
      setCoords(null);
      return;
    }

    function update() {
      if (!anchor) return;
      const r = anchor.getBoundingClientRect();
      const gap = 4;
      const spaceBelow = window.innerHeight - r.bottom - gap - 8;
      const spaceAbove = r.top - gap - 8;
      const placeAbove = spaceBelow < Math.min(maxHeight, 120) && spaceAbove > spaceBelow;
      const available = Math.max(80, placeAbove ? spaceAbove : spaceBelow);
      const height = Math.min(maxHeight, available);
      setCoords({
        top: placeAbove ? Math.max(8, r.top - gap - height) : r.bottom + gap,
        left: Math.min(r.left, window.innerWidth - r.width - 8),
        width: r.width,
        maxHeight: height,
      });
    }

    update();
    window.addEventListener("resize", update);
    window.addEventListener("scroll", update, true);
    return () => {
      window.removeEventListener("resize", update);
      window.removeEventListener("scroll", update, true);
    };
  }, [open, anchor, maxHeight]);

  React.useEffect(() => {
    if (!open || !onDismiss) return;
    const dismiss = onDismiss;

    function onPointerDown(e: MouseEvent) {
      const target = e.target as Node;
      if (anchor?.contains(target)) return;
      if (panelRef.current?.contains(target)) return;
      dismiss();
    }

    document.addEventListener("mousedown", onPointerDown);
    return () => document.removeEventListener("mousedown", onPointerDown);
  }, [open, onDismiss, anchor]);

  if (!mounted || !open || !anchor || !coords) return null;

  return createPortal(
    <div
      ref={panelRef}
      role="listbox"
      style={{
        position: "fixed",
        top: coords.top,
        left: coords.left,
        width: coords.width,
        maxHeight: coords.maxHeight,
        zIndex: 50,
      }}
      className={cn(
        "overflow-auto rounded-md border border-border bg-background shadow-sm",
        className,
      )}
    >
      {children}
    </div>,
    document.body,
  );
}
