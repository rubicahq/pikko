"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, ApiError, formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";

type LiveSession = {
  id: string;
  name: string;
  slug: string;
  opens_at: string;
  closes_at: string;
  operational_status: string;
  is_accepting_orders: boolean;
  order_count: number;
  item_count: number;
  gmv_idr: number;
  live_quick_add_enabled: boolean;
};

function wib(iso: string) {
  return new Intl.DateTimeFormat("id-ID", {
    timeZone: "Asia/Jakarta",
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(iso));
}

export function LiveSessionsList({ storeId }: { storeId: string }) {
  const [sessions, setSessions] = useState<LiveSession[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await apiFetch<{ sessions: LiveSession[] }>(
          `/api/v1/stores/${storeId}/live`,
        );
        if (!cancelled) setSessions(res.sessions || []);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.message : "Gagal memuat live sessions.");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [storeId]);

  const open = sessions.filter((s) => s.is_accepting_orders);
  const other = sessions.filter((s) => !s.is_accepting_orders);

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Live shopping"
        description="Sesi live tanpa video — quick-add SKU, order feed, dan kuota aman."
        actions={
          <Button asChild>
            <Link href={`${urls.storeCampaignNew(storeId)}?type=live`}>Buat sesi live</Link>
          </Button>
        }
      />

      {error ? <p className="text-sm text-error">{error}</p> : null}
      {loading ? <p className="text-sm text-foreground-secondary">Memuat…</p> : null}

      {!loading && sessions.length === 0 ? (
        <EmptyState
          title="Belum ada sesi live"
          description="Buat campaign bertipe live, assign produk featured, lalu buka console untuk quick-add."
          action={
            <Button asChild>
              <Link href={`${urls.storeCampaignNew(storeId)}?type=live`}>Buat sesi live</Link>
            </Button>
          }
        />
      ) : null}

      {open.length > 0 ? (
        <section className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground">Sedang berlangsung</h2>
          <ul className="divide-y divide-border rounded-[14px] border border-border bg-white">
            {open.map((s) => (
              <li key={s.id} className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-medium text-foreground">{s.name}</p>
                    <StatusBadge status={s.operational_status} />
                  </div>
                  <p className="mt-1 text-sm text-foreground-secondary">
                    Tutup {wib(s.closes_at)} WIB · {s.order_count} order · {formatIDR(s.gmv_idr)}
                  </p>
                </div>
                <Button asChild>
                  <Link href={urls.storeLiveSession(storeId, s.id)}>Buka console</Link>
                </Button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {other.length > 0 ? (
        <section className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground">Semua sesi</h2>
          <ul className="divide-y divide-border rounded-[14px] border border-border bg-white">
            {other.map((s) => (
              <li key={s.id} className="flex flex-wrap items-center justify-between gap-3 px-5 py-4">
                <div>
                  <div className="flex flex-wrap items-center gap-2">
                    <p className="font-medium text-foreground">{s.name}</p>
                    <StatusBadge status={s.operational_status} />
                  </div>
                  <p className="mt-1 text-sm text-foreground-secondary">
                    {wib(s.opens_at)} – {wib(s.closes_at)} WIB
                  </p>
                </div>
                <Button asChild variant="outline">
                  <Link href={urls.storeLiveSession(storeId, s.id)}>Buka</Link>
                </Button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </div>
  );
}
