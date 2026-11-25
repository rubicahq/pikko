"use client";

import { useEffect } from "react";
import { usePathname, useSearchParams } from "next/navigation";
import {
  resolveActiveTab,
  resolveWorkspace,
} from "@/components/dashboard/nav-config";

/** Sets `Menu Name | Pikko` from the active merchant workspace (or tab). */
export function DocumentTitle({
  storeId,
  permissions,
}: {
  storeId: string;
  permissions?: string[] | null;
}) {
  const pathname = usePathname();
  const searchParams = useSearchParams();
  const search = searchParams?.toString() ? `?${searchParams.toString()}` : "";

  useEffect(() => {
    const rest = pathname.startsWith(`/stores/${storeId}/`)
      ? pathname.slice(`/stores/${storeId}/`.length)
      : "";
    if (rest === "notifications" || rest.startsWith("notifications/")) {
      document.title = "Notifications | Pikko";
      return;
    }
    const workspace = resolveWorkspace(pathname, storeId, permissions);
    if (!workspace) {
      document.title = "Pikko";
      return;
    }
    // Prefer workspace label (Orders, Fulfillment, …) over tab for the browser title.
    let label = workspace.label;
    if (workspace.id === "settings") {
      const tab = resolveActiveTab(workspace, pathname, storeId, search);
      if (tab) label = tab.label;
    }
    document.title = `${label} | Pikko`;
  }, [pathname, search, storeId, permissions]);

  return null;
}
