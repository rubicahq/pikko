"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { apiFetch, ApiError, formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";

type LiveVariant = {
  id: string;
  option_key: string;
  effective_price_idr: number;
  availability: string;
};

type LiveProduct = {
  id: string;
  product_id: string;
  product_name: string;
  product_slug: string;
  effective_price_idr: number;
  quantity_limit?: number | null;
  sold_qty: number;
  remaining_qty?: number | null;
  availability: string;
  featured: boolean;
  variants?: LiveVariant[];
};

type LiveSession = {
  id: string;
  name: string;
  slug: string;
  opens_at: string;
  closes_at: string;
  operational_status: string;
  publication_status: string;
  live_quick_add_enabled: boolean;
  live_show_order_feed: boolean;
  is_accepting_orders: boolean;
  order_count: number;
  item_count: number;
  gmv_idr: number;
  products?: LiveProduct[];
};

type FeedEntry = {
  order_id: string;
  order_number: string;
  customer_name: string;
  customer_email: string;
  customer_phone?: string | null;
  order_status: string;
  payment_status: string;
  source: string;
  total_idr: number;
  item_count: number;
  created_at: string;
  updated_at: string;
  latest_product?: string | null;
  latest_qty?: number | null;
};

function wib(iso: string) {
  return new Intl.DateTimeFormat("id-ID", {
    timeZone: "Asia/Jakarta",
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(iso));
}

function newIdempotency() {
  if (typeof crypto !== "undefined" && crypto.randomUUID) return crypto.randomUUID();
  return `live-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export function LiveConsole({
  storeId,
  storeSlug,
  initialSession,
}: {
  storeId: string;
  storeSlug?: string | null;
  initialSession: LiveSession;
}) {
  const [session, setSession] = useState(initialSession);
  const [feed, setFeed] = useState<FeedEntry[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [selectedProductId, setSelectedProductId] = useState(
    () => initialSession.products?.find((p) => p.featured)?.product_id || initialSession.products?.[0]?.product_id || "",
  );
  const [variantId, setVariantId] = useState("");
  const [qty, setQty] = useState("1");
  const [orderQuery, setOrderQuery] = useState("");
  const [orderHits, setOrderHits] = useState<FeedEntry[]>([]);
  const [targetOrderId, setTargetOrderId] = useState<string | null>(null);
  const [custName, setCustName] = useState("");
  const [custEmail, setCustEmail] = useState("");
  const [custPhone, setCustPhone] = useState("");
  const [lastResult, setLastResult] = useState<string | null>(null);

  const selected = (session.products || []).find((p) => p.product_id === selectedProductId);

  const refreshSession = useCallback(async () => {
    const res = await apiFetch<{ session: LiveSession }>(
      `/api/v1/stores/${storeId}/live/${session.id}`,
    );
    setSession(res.session);
  }, [storeId, session.id]);

  const refreshFeed = useCallback(async () => {
    if (!session.live_show_order_feed) {
      setFeed([]);
      return;
    }
    const res = await apiFetch<{ feed: FeedEntry[] }>(
      `/api/v1/stores/${storeId}/live/${session.id}/feed`,
    );
    setFeed(res.feed || []);
  }, [storeId, session.id, session.live_show_order_feed]);

  useEffect(() => {
    let cancelled = false;
    async function tick() {
      try {
        if (!cancelled) {
          await refreshSession();
          await refreshFeed();
        }
      } catch {
        /* keep prior */
      }
    }
    tick();
    const id = setInterval(tick, 4000);
    return () => {
      cancelled = true;
      clearInterval(id);
    };
  }, [refreshSession, refreshFeed]);

  async function searchOrders() {
    setError(null);
    try {
      const res = await apiFetch<{ orders: FeedEntry[] }>(
        `/api/v1/stores/${storeId}/live/${session.id}/orders/search?q=${encodeURIComponent(orderQuery)}`,
      );
      setOrderHits(res.orders || []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Search failed.");
    }
  }

  async function quickAdd(mode: "existing" | "new") {
    if (!selectedProductId) return;
    setBusy(true);
    setError(null);
    setLastResult(null);
    try {
      const body: Record<string, unknown> = {
        product_id: selectedProductId,
        quantity: Number(qty) || 1,
        idempotency_key: newIdempotency(),
      };
      if (variantId) body.variant_id = variantId;
      if (mode === "existing") {
        if (!targetOrderId) {
          setError("Pilih order tujuan dulu.");
          setBusy(false);
          return;
        }
        body.order_id = targetOrderId;
      } else {
        body.customer_name = custName;
        body.customer_email = custEmail;
        if (custPhone) body.customer_phone = custPhone;
        body.fulfillment_method = "pickup";
      }
      const res = await apiFetch<{
        order: { id: string; order_number: string; total_idr: number };
        created_order: boolean;
        session?: LiveSession;
      }>(`/api/v1/stores/${storeId}/live/${session.id}/quick-add`, {
        method: "POST",
        body,
      });
      if (res.session) setSession(res.session);
      setLastResult(
        res.created_order
          ? `Order baru ${res.order.order_number} · ${formatIDR(res.order.total_idr)}`
          : `Ditambahkan ke ${res.order.order_number} · ${formatIDR(res.order.total_idr)}`,
      );
      setTargetOrderId(res.order.id);
      await refreshFeed();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Quick-add gagal.");
    } finally {
      setBusy(false);
    }
  }

  async function patchSession(patch: Record<string, unknown>) {
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/campaigns/${session.id}`, {
        method: "PATCH",
        body: patch,
      });
      await refreshSession();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Update gagal.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-8">
      <PageHeader
        title={session.name}
        description={`Live session · tutup ${wib(session.closes_at)} WIB`}
        actions={
          <div className="flex flex-wrap gap-2">
            {storeSlug ? (
              <Button asChild variant="outline">
                <Link href={urls.storefrontLive(storeSlug, session.slug)} target="_blank">
                  Lihat di storefront
                </Link>
              </Button>
            ) : null}
            <Button asChild variant="outline">
              <Link href={urls.storeCampaign(storeId, session.id)}>Kelola campaign</Link>
            </Button>
            {session.is_accepting_orders ? (
              <Button
                type="button"
                variant="outline"
                disabled={busy}
                onClick={() =>
                  patchSession({
                    operational_status: "closed",
                    closes_at: new Date().toISOString(),
                  })
                }
              >
                Tutup sesi
              </Button>
            ) : (
              <Button
                type="button"
                disabled={busy}
                onClick={() =>
                  patchSession({
                    publication_status: "published",
                    operational_status: "open",
                    closes_at: new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString(),
                  })
                }
              >
                Buka sesi
              </Button>
            )}
          </div>
        }
      />

      <div className="flex flex-wrap items-center gap-3 text-sm">
        <StatusBadge status={session.operational_status} />
        <span className="text-foreground-secondary">
          {session.is_accepting_orders ? "Menerima order" : "Tidak aktif"}
        </span>
        <span className="tabular-nums text-foreground">
          {session.order_count} order · {session.item_count} item · {formatIDR(session.gmv_idr)}
        </span>
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}
      {lastResult ? <p className="text-sm text-success">{lastResult}</p> : null}

      <div className="grid gap-8 lg:grid-cols-[1fr_360px]">
        <section className="space-y-4">
          <h2 className="text-lg font-semibold text-foreground">Quick-add SKU</h2>
          {(session.products || []).length === 0 ? (
            <EmptyState
              title="Belum ada produk"
              description="Assign produk di halaman campaign dulu."
              action={
                <Button asChild>
                  <Link href={urls.storeCampaign(storeId, session.id)}>Assign produk</Link>
                </Button>
              }
            />
          ) : (
            <div className="grid gap-3 sm:grid-cols-2">
              {(session.products || []).map((p) => {
                const active = p.product_id === selectedProductId;
                const exhausted =
                  p.remaining_qty != null ? p.remaining_qty <= 0 : p.availability !== "available";
                return (
                  <button
                    key={p.product_id}
                    type="button"
                    disabled={exhausted}
                    onClick={() => {
                      setSelectedProductId(p.product_id);
                      setVariantId("");
                    }}
                    className={`rounded-[14px] border px-4 py-3 text-left transition-colors ${
                      active
                        ? "border-ink bg-surface-active"
                        : "border-border bg-white hover:bg-surface-hover"
                    } ${exhausted ? "opacity-50" : ""}`}
                  >
                    <div className="flex items-start justify-between gap-2">
                      <p className="font-medium text-foreground">{p.product_name}</p>
                      {p.featured ? (
                        <span className="rounded-[6px] bg-brand-peach px-1.5 py-0.5 text-[11px] font-medium text-ink">
                          Featured
                        </span>
                      ) : null}
                    </div>
                    <p className="mt-1 tabular-nums text-sm text-foreground-secondary">
                      {formatIDR(p.effective_price_idr)}
                    </p>
                    <p className="mt-1 text-xs text-foreground-muted">
                      Terjual {p.sold_qty}
                      {p.quantity_limit != null ? ` / ${p.quantity_limit}` : ""}
                      {p.remaining_qty != null ? ` · sisa ${p.remaining_qty}` : ""}
                    </p>
                  </button>
                );
              })}
            </div>
          )}

          <div className="rounded-[14px] border border-border bg-white p-5 space-y-4">
            <div className="grid gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="qty">Qty</Label>
                <Input
                  id="qty"
                  inputMode="numeric"
                  value={qty}
                  onChange={(e) => setQty(e.target.value.replace(/\D/g, ""))}
                />
              </div>
              {selected?.variants && selected.variants.length > 0 ? (
                <div className="space-y-2">
                  <Label htmlFor="variant">Variant</Label>
                  <select
                    id="variant"
                    className="h-[42px] w-full rounded-[10px] border border-border bg-background px-3 text-sm"
                    value={variantId}
                    onChange={(e) => setVariantId(e.target.value)}
                  >
                    <option value="">Tanpa variant</option>
                    {selected.variants.map((v) => (
                      <option key={v.id} value={v.id}>
                        {v.option_key} · {formatIDR(v.effective_price_idr)}
                      </option>
                    ))}
                  </select>
                </div>
              ) : null}
            </div>

            <div className="space-y-3 border-t border-border pt-4">
              <p className="text-sm font-medium text-foreground">Ke order yang sudah ada</p>
              <div className="flex flex-wrap gap-2">
                <Input
                  placeholder="Cari no order / email / nama"
                  value={orderQuery}
                  onChange={(e) => setOrderQuery(e.target.value)}
                  className="max-w-sm"
                />
                <Button type="button" variant="outline" onClick={searchOrders}>
                  Cari
                </Button>
              </div>
              {orderHits.length > 0 ? (
                <ul className="divide-y divide-border rounded-[10px] border border-border">
                  {orderHits.map((o) => (
                    <li key={o.order_id}>
                      <button
                        type="button"
                        className={`flex w-full items-center justify-between px-3 py-2 text-left text-sm ${
                          targetOrderId === o.order_id ? "bg-surface-active" : "hover:bg-surface-hover"
                        }`}
                        onClick={() => setTargetOrderId(o.order_id)}
                      >
                        <span>
                          <span className="font-medium">{o.order_number}</span>
                          <span className="text-foreground-secondary"> · {o.customer_name}</span>
                        </span>
                        <span className="tabular-nums text-foreground-muted">
                          {formatIDR(o.total_idr)}
                        </span>
                      </button>
                    </li>
                  ))}
                </ul>
              ) : null}
              <Button
                type="button"
                disabled={busy || !session.is_accepting_orders || !selectedProductId}
                onClick={() => quickAdd("existing")}
              >
                Quick-add ke order
              </Button>
            </div>

            <div className="space-y-3 border-t border-border pt-4">
              <p className="text-sm font-medium text-foreground">Atau buat order baru</p>
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="space-y-2">
                  <Label htmlFor="cname">Nama</Label>
                  <Input id="cname" value={custName} onChange={(e) => setCustName(e.target.value)} />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="cemail">Email</Label>
                  <Input
                    id="cemail"
                    type="email"
                    value={custEmail}
                    onChange={(e) => setCustEmail(e.target.value)}
                  />
                </div>
                <div className="space-y-2 sm:col-span-2">
                  <Label htmlFor="cphone">Telepon (opsional)</Label>
                  <Input id="cphone" value={custPhone} onChange={(e) => setCustPhone(e.target.value)} />
                </div>
              </div>
              <Button
                type="button"
                disabled={
                  busy ||
                  !session.is_accepting_orders ||
                  !selectedProductId ||
                  custName.trim().length < 2 ||
                  !custEmail.includes("@")
                }
                onClick={() => quickAdd("new")}
              >
                Buat order + quick-add
              </Button>
            </div>
          </div>
        </section>

        <aside className="space-y-3">
          <h2 className="text-lg font-semibold text-foreground">Order feed</h2>
          {!session.live_show_order_feed ? (
            <p className="text-sm text-foreground-secondary">Feed dimatikan untuk sesi ini.</p>
          ) : feed.length === 0 ? (
            <p className="text-sm text-foreground-secondary">Belum ada order di sesi ini.</p>
          ) : (
            <ul className="space-y-2">
              {feed.map((e) => (
                <li
                  key={`${e.order_id}-${e.updated_at}`}
                  className="rounded-[14px] border border-border bg-white px-4 py-3"
                >
                  <div className="flex items-start justify-between gap-2">
                    <Link
                      href={urls.storeOrder(storeId, e.order_id)}
                      className="font-medium text-foreground hover:underline"
                    >
                      {e.order_number}
                    </Link>
                    <span className="tabular-nums text-sm">{formatIDR(e.total_idr)}</span>
                  </div>
                  <p className="mt-1 text-sm text-foreground-secondary">
                    {e.customer_name}
                    {e.latest_product ? ` · ${e.latest_product}` : ""}
                    {e.latest_qty != null ? ` ×${e.latest_qty}` : ""}
                  </p>
                  <p className="mt-1 text-xs text-foreground-muted">
                    {e.source.replaceAll("_", " ")} · {wib(e.updated_at)} WIB
                  </p>
                </li>
              ))}
            </ul>
          )}
        </aside>
      </div>
    </div>
  );
}
