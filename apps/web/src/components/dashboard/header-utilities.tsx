"use client";

import Link from "next/link";
import { MessageCircle, Sparkles } from "lucide-react";
import { urls } from "@/lib/urls";
import { NotificationBell } from "@/components/notifications/notification-bell";
import { CONNIE_NAV, HEADER_UTILITIES } from "@/components/dashboard/nav-config";
import { cn } from "@/lib/utils";

function can(
  permission: string | undefined,
  permissions: string[] | null | undefined,
): boolean {
  if (!permission) return true;
  if (!permissions) return true;
  return permissions.includes(permission);
}

export function HeaderUtilities({
  storeId,
  permissions,
}: {
  storeId: string;
  permissions?: string[] | null;
}) {
  const showMessages =
    HEADER_UTILITIES.messages.implemented &&
    can(HEADER_UTILITIES.messages.permission, permissions);
  const showNotifications =
    HEADER_UTILITIES.notifications.implemented &&
    can(HEADER_UTILITIES.notifications.permission, permissions);

  return (
    <div className="flex shrink-0 items-center gap-1">
      <div
        title="Ask Connie — coming soon"
        className={cn(
          "inline-flex h-9 max-w-[11rem] items-center gap-1.5 rounded-md bg-warm-feature px-2.5",
          "text-sm font-medium text-foreground",
        )}
        aria-disabled
      >
        <Sparkles className="h-4 w-4 shrink-0 text-brand" strokeWidth={1.75} />
        <span className="truncate">{CONNIE_NAV.label}</span>
        <span className="rounded-sm bg-background/70 px-1.5 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-foreground-muted">
          Soon
        </span>
      </div>
      {showMessages ? (
        <Link
          href={urls.storeMessages(storeId)}
          className={cn(
            "inline-flex h-9 w-9 items-center justify-center rounded-md text-foreground-secondary",
            "hover:bg-surface hover:text-foreground",
          )}
          aria-label="Messages"
          title="Messages"
        >
          <MessageCircle className="h-5 w-5" strokeWidth={1.75} />
        </Link>
      ) : null}
      {showNotifications ? <NotificationBell storeId={storeId} /> : null}
    </div>
  );
}
