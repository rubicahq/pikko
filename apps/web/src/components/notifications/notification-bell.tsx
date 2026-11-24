"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { Bell } from "lucide-react";
import { urls } from "@/lib/urls";
import { apiFetch } from "@/lib/api";

export function NotificationBell({ storeId }: { storeId: string }) {
  const [count, setCount] = useState(0);

  useEffect(() => {
    let cancelled = false;
    async function load() {
      try {
        const data = await apiFetch<{ unread_count: number }>(
          `/api/v1/stores/${storeId}/notifications/unread-count`,
        );
        if (!cancelled) setCount(data.unread_count || 0);
      } catch {
        /* ignore */
      }
    }
    load();
    const t = setInterval(load, 30000);
    return () => {
      cancelled = true;
      clearInterval(t);
    };
  }, [storeId]);

  return (
    <Link
      href={urls.storeNotifications(storeId)}
      className="relative inline-flex h-10 w-10 items-center justify-center rounded-md text-foreground-secondary hover:bg-surface hover:text-foreground"
      aria-label={count > 0 ? `${count} unread notifications` : "Notifications"}
    >
      <Bell className="h-5 w-5" strokeWidth={1.75} />
      {count > 0 ? (
        <span className="absolute right-1 top-1 flex h-4 min-w-4 items-center justify-center rounded bg-brand-accessible-dark px-1 text-[10px] font-medium text-foreground-inverse">
          {count > 99 ? "99+" : count}
        </span>
      ) : null}
    </Link>
  );
}
