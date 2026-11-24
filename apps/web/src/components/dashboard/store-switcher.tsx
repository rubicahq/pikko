"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { Check, ChevronsUpDown, Plus, Settings2 } from "lucide-react";
import * as DropdownMenu from "@radix-ui/react-dropdown-menu";
import type { Store } from "@/lib/api";
import { urls } from "@/lib/urls";
import { cn } from "@/lib/utils";

function initial(name: string) {
  return (name.trim()[0] || "S").toUpperCase();
}

export function StoreSwitcher({
  stores,
  current,
}: {
  stores: Store[];
  current?: Store | null;
}) {
  const router = useRouter();
  const active = current || stores[0];

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          className="flex w-full items-center gap-3 rounded-md border border-border bg-background px-3 py-2 text-left hover:bg-background-subtle"
        >
          <span className="flex h-8 w-8 items-center justify-center rounded-md bg-surface text-sm font-medium text-foreground">
            {active ? initial(active.name) : "?"}
          </span>
          <span className="min-w-0 flex-1">
            <span className="block truncate text-sm font-medium text-foreground">
              {active?.name || "Select store"}
            </span>
            {active ? (
              <span className="block truncate text-xs text-foreground-muted">{active.slug}</span>
            ) : null}
          </span>
          <ChevronsUpDown className="h-4 w-4 text-foreground-muted" strokeWidth={1.75} />
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content
          align="start"
          sideOffset={6}
          className="z-50 w-[var(--radix-dropdown-menu-trigger-width)] min-w-[240px] rounded-md border border-border bg-background p-1 shadow-sm"
        >
          {stores.map((store) => (
            <DropdownMenu.Item
              key={store.id}
              className={cn(
                "flex cursor-pointer items-center gap-2 rounded-sm px-2 py-2 text-sm outline-none hover:bg-surface focus:bg-surface",
                active?.id === store.id && "bg-brand-soft",
              )}
              onSelect={() => router.push(urls.storeOverview(store.id))}
            >
              <span className="flex h-7 w-7 items-center justify-center rounded-sm bg-surface text-xs font-medium">
                {initial(store.name)}
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate">{store.name}</span>
                {store.role ? (
                  <span className="block truncate text-xs text-foreground-muted">{store.role}</span>
                ) : null}
              </span>
              {active?.id === store.id ? (
                <Check className="h-4 w-4 text-brand-dark" strokeWidth={1.75} />
              ) : null}
            </DropdownMenu.Item>
          ))}
          <DropdownMenu.Separator className="my-1 h-px bg-border" />
          <DropdownMenu.Item asChild>
            <Link
              href={urls.newStore()}
              className="flex items-center gap-2 rounded-sm px-2 py-2 text-sm outline-none hover:bg-surface"
            >
              <Plus className="h-4 w-4" strokeWidth={1.75} />
              Create new store
            </Link>
          </DropdownMenu.Item>
          <DropdownMenu.Item asChild>
            <Link
              href={urls.stores()}
              className="flex items-center gap-2 rounded-sm px-2 py-2 text-sm outline-none hover:bg-surface"
            >
              <Settings2 className="h-4 w-4" strokeWidth={1.75} />
              Manage stores
            </Link>
          </DropdownMenu.Item>
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
}
