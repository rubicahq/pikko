"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import { liveNavGroups, type NavGroup } from "@/components/dashboard/nav-config";

export function SidebarNav({
  storeId,
  onNavigate,
  groups,
  permissions,
}: {
  storeId?: string;
  onNavigate?: () => void;
  groups?: NavGroup[];
  permissions?: string[] | null;
}) {
  const pathname = usePathname();
  const sections = liveNavGroups(groups, permissions);

  return (
    <nav className="flex-1 space-y-5 overflow-y-auto px-3 pb-4" aria-label="Store">
      {sections.map((group) => (
        <div key={group.id}>
          <p className="px-3 pb-1.5 text-nav-group uppercase text-foreground-muted">
            {group.label}
          </p>
          <div className="space-y-0.5">
            {group.items.map((item) => {
              const href = storeId ? `/stores/${storeId}/${item.href}` : "#";
              const active = storeId ? pathname.startsWith(href) : false;
              const Icon = item.icon;
              return (
                <Link
                  key={item.href}
                  href={href}
                  onClick={onNavigate}
                  className={cn(
                    "flex h-10 items-center gap-2 rounded-md px-3 text-sm font-medium text-foreground-secondary transition-colors hover:bg-surface-hover hover:text-foreground",
                    active && "bg-surface-active font-semibold text-foreground",
                  )}
                  aria-current={active ? "page" : undefined}
                >
                  <Icon
                    className={cn("h-5 w-5 shrink-0", active && "text-brand")}
                    strokeWidth={1.75}
                  />
                  <span className="truncate">{item.label}</span>
                </Link>
              );
            })}
          </div>
        </div>
      ))}
    </nav>
  );
}
