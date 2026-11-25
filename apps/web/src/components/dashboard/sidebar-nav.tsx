"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  livePrimaryWorkspaces,
  matchWorkspace,
  storeHref,
  type WorkspaceDef,
} from "@/components/dashboard/nav-config";

export function SidebarNav({
  storeId,
  onNavigate,
  workspaces,
  permissions,
  collapsed = false,
}: {
  storeId?: string;
  onNavigate?: () => void;
  workspaces?: WorkspaceDef[];
  permissions?: string[] | null;
  collapsed?: boolean;
}) {
  const pathname = usePathname();
  const items = workspaces ?? livePrimaryWorkspaces(permissions);

  return (
    <nav
      className="flex-1 space-y-1 overflow-y-auto px-3 pb-4"
      aria-label="Store"
    >
      {items.map((item) => {
        const href = storeId ? storeHref(storeId, item.defaultHref) : "#";
        const active = storeId
          ? matchWorkspace(item, pathname, storeId)
          : false;
        const Icon = item.icon;
        return (
          <Link
            key={item.id}
            href={href}
            onClick={onNavigate}
            title={collapsed ? item.label : undefined}
            className={cn(
              "flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium text-foreground-secondary transition-colors",
              "hover:bg-background-subtle hover:text-foreground",
              active && "bg-surface-active font-semibold text-foreground",
              collapsed && "justify-center px-0",
            )}
            aria-current={active ? "page" : undefined}
          >
            <Icon
              className={cn("h-5 w-5 shrink-0", active && "text-brand")}
              strokeWidth={1.75}
            />
            {!collapsed ? <span className="truncate">{item.label}</span> : null}
            {collapsed ? <span className="sr-only">{item.label}</span> : null}
          </Link>
        );
      })}
    </nav>
  );
}
