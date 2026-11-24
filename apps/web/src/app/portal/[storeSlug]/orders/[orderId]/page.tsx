"use client";

import Link from "next/link";
import { FormEvent, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch, formatIDR, ApiError } from "@/lib/api";
import { PortalShell } from "@/components/portal/portal-shell";
import { usePortalSession } from "@/components/portal/use-portal-session";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Detail = {
  order: {
    id: string;
    order_number: string;
    order_status: string;
    payment_status: string;
    fulfillment_status: string;
    fulfillment_method: string;
    total_idr: number;
    subtotal_idr: number;
    shipping_fee_idr: number;
    handling_fee_idr?: number;
    shipping_recipient?: string | null;
    shipping_line1?: string | null;
    shipping_city?: string | null;
    items?: {
      product_name: string;
      quantity: number;
      unit_price_idr: number;
      line_total_idr: number;
      availability_status?: string;
    }[];
  };
  finance?: {
    payable_idr: number;
    paid_idr: number;
    refund_entitled_idr: number;
    remaining_due_idr: number;
    unavailable_delta_idr: number;
  } | null;
  payments: {
    id: string;
    amount_idr: number;
    status: string;
    payment_kind?: string;
    created_at: string;
  }[];
  adjustments: {
    id: string;
    adjustment_type: string;
    amount_idr: number;
    reason?: string | null;
    created_at: string;
  }[];
  refunds: {
    id: string;
    amount_idr: number;
    method: string;
    status: string;
    created_at: string;
  }[];
  can_request_address_change: boolean;
  pending_address_change?: { status: string; line1: string; city: string } | null;
};

export default function PortalOrderDetailPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const orderId = String(params.orderId || "");
  const { session, loading: sessionLoading } = usePortalSession(storeSlug);
  const [detail, setDetail] = useState<Detail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [showChange, setShowChange] = useState(false);
  const [changeForm, setChangeForm] = useState({
    recipient_name: "",
    line1: "",
    city: "",
    province: "",
    postal_code: "",
    phone: "",
    customer_note: "",
  });
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch<{ detail: Detail }>(
        `/api/v1/portal/${storeSlug}/orders/${orderId}`,
      );
      setDetail(res.detail);
      setError(null);
      const o = res.detail.order;
      setChangeForm((f) => ({
        ...f,
        recipient_name: o.shipping_recipient || "",
        line1: o.shipping_line1 || "",
        city: o.shipping_city || "",
      }));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Order not found.");
      setDetail(null);
    }
  }, [storeSlug, orderId]);

  useEffect(() => {
    if (session) load();
  }, [session, load]);

  async function submitAddressChange(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setMessage(null);
    try {
      await apiFetch(`/api/v1/portal/${storeSlug}/orders/${orderId}/address-change`, {
        body: {
          recipient_name: changeForm.recipient_name,
          line1: changeForm.line1,
          city: changeForm.city,
          province: changeForm.province || undefined,
          postal_code: changeForm.postal_code || undefined,
          phone: changeForm.phone || undefined,
          customer_note: changeForm.customer_note || undefined,
          country: "ID",
        },
      });
      setMessage("Address change requested. The merchant will review it.");
      setShowChange(false);
      await load();
    } catch (err) {
      setMessage(err instanceof ApiError ? err.message : "Unable to submit request.");
    } finally {
      setSaving(false);
    }
  }

  if (sessionLoading || !session) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-workspace">
        <p className="text-sm text-foreground-secondary">Loading…</p>
      </div>
    );
  }

  return (
    <PortalShell
      storeSlug={storeSlug}
      storeName={session.store.name}
      customerEmail={session.customer.email}
      primaryColor={session.store.primary_color}
    >
      <Link
        href={`/portal/${storeSlug}/dashboard`}
        className="text-sm font-medium text-foreground-secondary hover:text-foreground"
      >
        ← Orders
      </Link>

      {error ? <p className="mt-6 text-sm text-error">{error}</p> : null}

      {detail ? (
        <>
          <h1 className="mt-4 text-page font-bold tracking-tight text-foreground">
            {detail.order.order_number}
          </h1>
          <p className="mt-1.5 text-body capitalize text-foreground-secondary">
            {detail.order.payment_status.replaceAll("_", " ")} ·{" "}
            {detail.order.fulfillment_status.replaceAll("_", " ")}
          </p>

          <section className="mt-8 rounded-xl border border-border bg-background p-5">
            <h2 className="text-card font-semibold text-foreground">Items</h2>
            <ul className="mt-3 space-y-2 text-sm">
              {(detail.order.items || []).map((it, i) => (
                <li key={i} className="flex justify-between gap-3">
                  <span>
                    {it.product_name} × {it.quantity}
                    {it.availability_status && it.availability_status !== "available" ? (
                      <span className="ml-2 text-warning">
                        ({it.availability_status.replaceAll("_", " ")})
                      </span>
                    ) : null}
                  </span>
                  <span className="tabular-nums">{formatIDR(it.line_total_idr)}</span>
                </li>
              ))}
            </ul>
            <div className="mt-4 space-y-1 border-t border-border pt-3 text-sm">
              <div className="flex justify-between">
                <span>Subtotal</span>
                <span className="tabular-nums">{formatIDR(detail.order.subtotal_idr)}</span>
              </div>
              {(detail.order.shipping_fee_idr || 0) > 0 ? (
                <div className="flex justify-between">
                  <span>Shipping</span>
                  <span className="tabular-nums">{formatIDR(detail.order.shipping_fee_idr)}</span>
                </div>
              ) : null}
              <div className="flex justify-between pt-1 font-semibold">
                <span>Total</span>
                <span className="tabular-nums">{formatIDR(detail.order.total_idr)}</span>
              </div>
            </div>
          </section>

          {detail.finance ? (
            <section className="mt-6 rounded-xl border border-border bg-background p-5">
              <h2 className="text-card font-semibold text-foreground">Settlement</h2>
              <dl className="mt-3 space-y-2 text-sm">
                <div className="flex justify-between">
                  <dt className="text-foreground-secondary">Payable</dt>
                  <dd className="tabular-nums">{formatIDR(detail.finance.payable_idr)}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-foreground-secondary">Paid</dt>
                  <dd className="tabular-nums">{formatIDR(detail.finance.paid_idr)}</dd>
                </div>
                <div className="flex justify-between">
                  <dt className="text-foreground-secondary">Remaining due</dt>
                  <dd className="tabular-nums">{formatIDR(detail.finance.remaining_due_idr)}</dd>
                </div>
                {detail.finance.unavailable_delta_idr > 0 ? (
                  <div className="flex justify-between">
                    <dt className="text-foreground-secondary">Unavailable adjustment</dt>
                    <dd className="tabular-nums">
                      −{formatIDR(detail.finance.unavailable_delta_idr)}
                    </dd>
                  </div>
                ) : null}
                {detail.finance.refund_entitled_idr > 0 ? (
                  <div className="flex justify-between">
                    <dt className="text-foreground-secondary">Refund entitled</dt>
                    <dd className="tabular-nums">{formatIDR(detail.finance.refund_entitled_idr)}</dd>
                  </div>
                ) : null}
              </dl>
            </section>
          ) : null}

          <section className="mt-6 rounded-xl border border-border bg-background p-5">
            <h2 className="text-card font-semibold text-foreground">Payments</h2>
            {(detail.payments || []).length === 0 ? (
              <p className="mt-3 text-sm text-foreground-secondary">No payments yet.</p>
            ) : (
              <ul className="mt-3 space-y-2 text-sm">
                {detail.payments.map((p) => (
                  <li key={p.id} className="flex justify-between gap-3">
                    <span className="capitalize text-foreground-secondary">
                      {(p.payment_kind || "payment").replaceAll("_", " ")} · {p.status}
                    </span>
                    <span className="tabular-nums">{formatIDR(p.amount_idr)}</span>
                  </li>
                ))}
              </ul>
            )}
          </section>

          {(detail.adjustments || []).length > 0 ? (
            <section className="mt-6 rounded-xl border border-border bg-background p-5">
              <h2 className="text-card font-semibold text-foreground">Adjustments</h2>
              <ul className="mt-3 space-y-2 text-sm">
                {detail.adjustments.map((a) => (
                  <li key={a.id} className="flex justify-between gap-3">
                    <span className="capitalize text-foreground-secondary">
                      {a.adjustment_type.replaceAll("_", " ")}
                      {a.reason ? ` — ${a.reason}` : ""}
                    </span>
                    <span className="tabular-nums">{formatIDR(a.amount_idr)}</span>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}

          {(detail.refunds || []).length > 0 ? (
            <section className="mt-6 rounded-xl border border-border bg-background p-5">
              <h2 className="text-card font-semibold text-foreground">Refunds</h2>
              <ul className="mt-3 space-y-2 text-sm">
                {detail.refunds.map((r) => (
                  <li key={r.id} className="flex justify-between gap-3">
                    <span className="capitalize text-foreground-secondary">
                      {r.method.replaceAll("_", " ")} · {r.status}
                    </span>
                    <span className="tabular-nums">{formatIDR(r.amount_idr)}</span>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}

          {(detail.order.fulfillment_method === "shipping" ||
            detail.order.fulfillment_method === "free_shipping") && (
            <section className="mt-6 rounded-xl border border-border bg-background p-5">
              <h2 className="text-card font-semibold text-foreground">Shipping address</h2>
              <p className="mt-3 text-sm text-foreground-secondary">
                {detail.order.shipping_recipient}
                <br />
                {detail.order.shipping_line1}
                <br />
                {detail.order.shipping_city}
              </p>
              {detail.pending_address_change ? (
                <p className="mt-3 text-sm text-warning">
                  Address change pending merchant approval (
                  {detail.pending_address_change.line1}, {detail.pending_address_change.city}).
                </p>
              ) : null}
              {detail.can_request_address_change ? (
                <Button
                  className="mt-4"
                  variant="secondary"
                  onClick={() => setShowChange((v) => !v)}
                >
                  Request address change
                </Button>
              ) : null}
              {showChange ? (
                <form onSubmit={submitAddressChange} className="mt-4 space-y-3">
                  <div className="space-y-1.5">
                    <Label htmlFor="recipient">Recipient</Label>
                    <Input
                      id="recipient"
                      required
                      value={changeForm.recipient_name}
                      onChange={(e) =>
                        setChangeForm({ ...changeForm, recipient_name: e.target.value })
                      }
                    />
                  </div>
                  <div className="space-y-1.5">
                    <Label htmlFor="line1">Address</Label>
                    <Input
                      id="line1"
                      required
                      value={changeForm.line1}
                      onChange={(e) => setChangeForm({ ...changeForm, line1: e.target.value })}
                    />
                  </div>
                  <div className="grid gap-3 sm:grid-cols-2">
                    <div className="space-y-1.5">
                      <Label htmlFor="city">City</Label>
                      <Input
                        id="city"
                        required
                        value={changeForm.city}
                        onChange={(e) => setChangeForm({ ...changeForm, city: e.target.value })}
                      />
                    </div>
                    <div className="space-y-1.5">
                      <Label htmlFor="postal">Postal code</Label>
                      <Input
                        id="postal"
                        value={changeForm.postal_code}
                        onChange={(e) =>
                          setChangeForm({ ...changeForm, postal_code: e.target.value })
                        }
                      />
                    </div>
                  </div>
                  <Button type="submit" loading={saving}>
                    Submit for approval
                  </Button>
                </form>
              ) : null}
              {message ? <p className="mt-3 text-sm text-foreground-secondary">{message}</p> : null}
            </section>
          )}
        </>
      ) : null}
    </PortalShell>
  );
}
