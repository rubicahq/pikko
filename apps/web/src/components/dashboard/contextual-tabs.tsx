"use client";

import Link from "next/link";
import { useEffect, useRef } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  resolveActiveTab,
  resolveWorkspace,
  storeHref,
} from "@/components/dashboard/nav-config";

export function ContextualTabs({
  storeId,
  permissions,
}: {
  storeId: string;
  permissions?: string[] | null;
}) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const search = searchParams.toString();
  const workspace = resolveWorkspace(pathname, storeId, permissions);
  const scrollerRef = useRef<HTMLElement>(null);

  const tabs = workspace?.tabs ?? [];
  const active = workspace
    ? resolveActiveTab(workspace, pathname, storeId, search)
    : null;

  useEffect(() => {
    const root = scrollerRef.current;
    if (!root || !active) return;
    const el = root.querySelector<HTMLElement>(`[data-tab-id="${active.id}"]`);
    el?.scrollIntoView({ inline: "nearest", block: "nearest", behavior: "smooth" });
  }, [active?.id, pathname, search]);

  if (!workspace || tabs.length === 0) {
    return <div className="min-w-0 flex-1" />;
  }

  // Single-tab workspaces still show the tab for orientation consistency.
  return (
    <nav
      ref={scrollerRef}
      className="flex min-w-0 flex-1 items-stretch gap-0 overflow-x-auto [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      aria-label={`${workspace.label} sections`}
    >
      {tabs.map((tab) => {
        const href = storeHref(storeId, tab.href);
        const isActive = active?.id === tab.id;
        return (
          <Link
            key={tab.id}
            data-tab-id={tab.id}
            href={href}
            className={cn(
              "relative flex h-full shrink-0 items-center px-3.5 text-sm font-medium text-foreground-secondary transition-colors",
              "hover:bg-background-subtle hover:text-foreground",
              isActive && "font-semibold text-foreground",
            )}
            aria-current={isActive ? "page" : undefined}
          >
            {tab.label}
            {isActive ? (
              <span
                aria-hidden
                className="absolute inset-x-0 bottom-0 h-0.5 bg-brand"
              />
            ) : null}
          </Link>
        );
      })}
    </nav>
  );
}
