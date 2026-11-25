"use client";

import Link from "next/link";
import { MessageCircle } from "lucide-react";
import { urls } from "@/lib/urls";
import { NotificationBell } from "@/components/notifications/notification-bell";
import { HEADER_UTILITIES } from "@/components/dashboard/nav-config";
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

  if (!showMessages && !showNotifications) return null;

  return (
    <div className="flex shrink-0 items-center gap-1">
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
