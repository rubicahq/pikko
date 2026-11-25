"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";

export type PortalSession = {
  customer: { id: string; name: string; email?: string | null; phone?: string | null };
  store: { id: string; name: string; slug: string; primary_color?: string };
};

export function usePortalSession(storeSlug: string) {
  const router = useRouter();
  const [session, setSession] = useState<PortalSession | null>(null);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch<PortalSession>(`/api/v1/portal/${storeSlug}/auth/session`);
      setSession(res);
      setLoading(false);
    } catch {
      setSession(null);
      setLoading(false);
      router.replace(`/portal/${storeSlug}`);
    }
  }, [storeSlug, router]);

  useEffect(() => {
    load();
  }, [load]);

  return { session, loading, reload: load };
}
