"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";

type Notification = {
  id: string;
  type: string;
  title: string;
  body?: string | null;
  resource_type?: string | null;
  resource_id?: string | null;
  read_at?: string | null;
  created_at: string;
};

function hrefFor(storeId: string, n: Notification): string | null {
  if (n.resource_type === "order" && n.resource_id) {
    return urls.storeOrder(storeId, n.resource_id);
  }
  if (n.resource_type === "preorder_batch" && n.resource_id) {
    return `/stores/${storeId}/preorders/${n.resource_id}`;
  }
  return null;
}

export function NotificationsClient({
  storeId,
  initial,
  unreadCount,
}: {
  storeId: string;
  initial: Notification[];
  unreadCount: number;
}) {
  const router = useRouter();
  const [items, setItems] = useState(initial);
  const [busy, setBusy] = useState(false);

  async function markRead(id: string) {
    setBusy(true);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/notifications/${id}/read`, { method: "POST" });
      setItems((prev) =>
        prev.map((n) => (n.id === id ? { ...n, read_at: n.read_at || new Date().toISOString() } : n)),
      );
      router.refresh();
    } finally {
      setBusy(false);
    }
  }

  async function markAll() {
    setBusy(true);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/notifications/read-all`, { method: "POST" });
      const now = new Date().toISOString();
      setItems((prev) => prev.map((n) => ({ ...n, read_at: n.read_at || now })));
      router.refresh();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mt-8 space-y-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-foreground-secondary">{unreadCount} unread</p>
        <Button type="button" variant="outline" size="sm" disabled={busy || unreadCount === 0} onClick={markAll}>
          Mark all read
        </Button>
      </div>
      {items.length === 0 ? (
        <div className="rounded-[10px] border border-border p-8">
          <p className="text-sm text-foreground-secondary">No notifications yet.</p>
        </div>
      ) : (
        <ul className="divide-y divide-border overflow-hidden rounded-[10px] border border-border">
          {items.map((n) => {
            const href = hrefFor(storeId, n);
            const unread = !n.read_at;
            return (
              <li
                key={n.id}
                className={`flex flex-wrap items-start justify-between gap-3 px-4 py-4 ${unread ? "bg-brand-soft/40" : "bg-background"}`}
              >
                <div className="min-w-0 flex-1">
                  <p className={`text-sm ${unread ? "font-medium text-foreground" : "text-foreground"}`}>
                    {href ? (
                      <Link href={href} className="hover:underline">
                        {n.title}
                      </Link>
                    ) : (
                      n.title
                    )}
                  </p>
                  {n.body ? <p className="mt-1 text-sm text-foreground-secondary">{n.body}</p> : null}
                  <p className="mt-2 text-xs text-foreground-muted">
                    {new Date(n.created_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })}
                  </p>
                </div>
                {unread ? (
                  <Button type="button" variant="ghost" size="sm" disabled={busy} onClick={() => markRead(n.id)}>
                    Mark read
                  </Button>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
