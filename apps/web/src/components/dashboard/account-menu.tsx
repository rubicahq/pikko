"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { ChevronsUpDown, LogOut, UserRound } from "lucide-react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import type { User } from "@/lib/api";
import { apiFetch } from "@/lib/api";
import { urls } from "@/lib/urls";
import { cn } from "@/lib/utils";
import { ACCOUNT_NAV } from "@/components/dashboard/nav-config";

function initials(user: User) {
  const name = (user.display_name || user.email || "?").trim();
  const parts = name.split(/\s+/).filter(Boolean);
  if (parts.length >= 2) {
    return (parts[0][0] + parts[1][0]).toUpperCase();
  }
  return name.slice(0, 2).toUpperCase();
}

export function AccountMenu({
  user,
  collapsed = false,
}: {
  user: User;
  collapsed?: boolean;
}) {
  const router = useRouter();
  const Icon = ACCOUNT_NAV.icon;
  const label = user.display_name || user.email;

  async function signOut() {
    try {
      await apiFetch("/api/v1/auth/logout", { method: "POST", body: {} });
    } catch {
      /* still redirect */
    }
    router.push(urls.auth());
    router.refresh();
  }

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          title={collapsed ? label : undefined}
          className={cn(
            "flex h-10 w-full items-center gap-3 rounded-md px-3 text-sm text-foreground-secondary transition-colors",
            "hover:bg-surface-hover hover:text-foreground",
            collapsed && "justify-center px-0",
          )}
        >
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-surface text-xs font-semibold text-foreground">
            {initials(user)}
          </span>
          {!collapsed ? (
            <>
              <span className="min-w-0 flex-1 truncate text-left font-medium">
                {label}
              </span>
              <ChevronsUpDown className="h-4 w-4 shrink-0 text-foreground-muted" strokeWidth={1.75} />
            </>
          ) : (
            <span className="sr-only">{ACCOUNT_NAV.label}</span>
          )}
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align={collapsed ? "center" : "start"}
          side="top"
          sideOffset={6}
          className="z-50 min-w-[200px] rounded-lg border border-border bg-background p-1 shadow-md"
        >
          <DropdownMenu.Item asChild>
            <Link
              href={urls.accountSettings()}
              className="flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground outline-none hover:bg-surface-hover"
            >
              <UserRound className="h-4 w-4" strokeWidth={1.75} />
              Account Settings
            </Link>
          </DropdownMenu.Item>
          <DropdownMenu.Item asChild>
            <Link
              href={urls.accountSettings()}
              className="flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm text-foreground outline-none hover:bg-surface-hover"
            >
              <Icon className="h-4 w-4" strokeWidth={1.75} />
              My Profile
            </Link>
          </DropdownMenu.Item>
          <DropdownMenu.Separator className="my-1 h-px bg-border" />
          <DropdownMenu.Item
            className="flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 text-sm text-error outline-none hover:bg-error-bg"
            onSelect={(e) => {
              e.preventDefault();
              void signOut();
            }}
          >
            <LogOut className="h-4 w-4" strokeWidth={1.75} />
            Sign Out
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
