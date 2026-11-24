"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch, apiUpload, formatIDR } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";

type Tracking = {
  store_name: string;
  store_slug: string;
  primary_color?: string;
  contact_email?: string | null;
  whatsapp?: string | null;
  can_submit_evidence: boolean;
  order: {
    order_number: string;
    order_status: string;
    payment_status: string;
    fulfillment_status: string;
    total_idr: number;
    subtotal_idr: number;
    shipping_fee_idr: number;
    handling_fee_idr?: number;
    fulfillment_method: string;
    customer_name: string;
    payment_deadline_at?: string | null;
    items?: {
      product_name: string;
      quantity: number;
      unit_price_idr: number;
      line_total_idr: number;
    }[];
  };
  payment?: {
    id: string;
    status: string;
    bank_name_snapshot?: string | null;
    account_holder_snapshot?: string | null;
    account_number_snapshot?: string | null;
    instructions_snapshot?: string | null;
    amount_idr: number;
    evidence?: { id: string; submitted_at: string; signed_url?: string }[];
  } | null;
  shipments?: {
    id: string;
    carrier: string;
    tracking_number: string;
    status: string;
    items?: { product_name?: string; quantity: number }[];
  }[];
};

export default function TrackingPage() {
  const params = useParams();
  const token = String(params.token || "");
  const [tracking, setTracking] = useState<Tracking | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [uploading, setUploading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch<{ tracking: Tracking }>(`/api/v1/public/tracking/${encodeURIComponent(token)}`);
      setTracking(res.tracking);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Order not found.");
      setTracking(null);
    }
  }, [token]);

  useEffect(() => {
    load();
  }, [load]);

  async function uploadEvidence(file: File) {
    setUploading(true);
    setMessage(null);
    try {
      const form = new FormData();
      form.set("file", file);
      await apiUpload(`/api/v1/public/tracking/${encodeURIComponent(token)}/evidence`, form);
      setMessage("Payment proof uploaded. The merchant will review it shortly.");
      await load();
    } catch (err) {
      setMessage(err instanceof Error ? err.message : "Upload failed.");
    } finally {
      setUploading(false);
    }
  }

  if (error) {
    return (
      <div className="mx-auto max-w-reading px-4 py-16">
        <h1 className="text-[30px] font-medium">Order not found</h1>
        <p className="mt-3 text-sm text-foreground-secondary">{error}</p>
      </div>
    );
  }

  if (!tracking) {
    return (
      <div className="mx-auto max-w-reading px-4 py-16">
        <p className="text-sm text-foreground-secondary">Loading order…</p>
      </div>
    );
  }

  const o = tracking.order;
  const p = tracking.payment;
  const accent = tracking.primary_color || "#FF3D00";

  return (
    <div className="min-h-screen bg-background-subtle">
      <div className="mx-auto max-w-reading px-4 py-10 md:px-0">
        <p className="text-sm text-foreground-secondary">{tracking.store_name}</p>
        <h1 className="mt-1 text-[30px] font-medium text-foreground">Order {o.order_number}</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Hi {o.customer_name}. Track payment and fulfillment below.
        </p>

        <ol className="mt-8 space-y-3">
          {[
            { label: "Order", value: o.order_status },
            { label: "Payment", value: o.payment_status },
            { label: "Fulfillment", value: o.fulfillment_status },
          ].map((step) => (
            <li
              key={step.label}
              className="flex items-center justify-between rounded-[10px] border border-border bg-background px-4 py-3 text-sm"
            >
              <span className="font-medium">{step.label}</span>
              <span className="capitalize text-foreground-secondary">{step.value.replaceAll("_", " ")}</span>
            </li>
          ))}
        </ol>

        <section className="mt-8 rounded-[10px] border border-border bg-background p-5">
          <h2 className="text-base font-medium">Items</h2>
          <ul className="mt-3 space-y-2 text-sm">
            {(o.items || []).map((it, i) => (
              <li key={i} className="flex justify-between gap-3">
                <span>
                  {it.product_name} × {it.quantity}
                  <span className="text-foreground-muted"> · {formatIDR(it.unit_price_idr)}</span>
                </span>
                <span className="tabular-nums">{formatIDR(it.line_total_idr)}</span>
              </li>
            ))}
          </ul>
          <div className="mt-4 space-y-1 border-t border-border pt-3 text-sm">
            <div className="flex justify-between">
              <span>Subtotal</span>
              <span className="tabular-nums">{formatIDR(o.subtotal_idr)}</span>
            </div>
            {(o.handling_fee_idr ?? 0) > 0 ? (
              <div className="flex justify-between">
                <span>Handling fee</span>
                <span className="tabular-nums">{formatIDR(o.handling_fee_idr || 0)}</span>
              </div>
            ) : null}
            {(o.shipping_fee_idr ?? 0) > 0 ? (
              <div className="flex justify-between">
                <span>Shipping</span>
                <span className="tabular-nums">{formatIDR(o.shipping_fee_idr)}</span>
              </div>
            ) : null}
            <div className="flex justify-between pt-1 font-medium">
              <span>Total</span>
              <span className="tabular-nums">{formatIDR(o.total_idr)}</span>
            </div>
          </div>
        </section>

        {(tracking.shipments || []).length > 0 ? (
          <section className="mt-6 rounded-[10px] border border-border bg-background p-5">
            <h2 className="text-base font-medium">Shipments</h2>
            <ul className="mt-3 space-y-3 text-sm">
              {tracking.shipments!.map((sh) => (
                <li key={sh.id}>
                  <div className="flex justify-between gap-3">
                    <span className="font-medium">
                      {sh.carrier} · {sh.tracking_number}
                    </span>
                    <span className="capitalize text-foreground-secondary">
                      {sh.status.replaceAll("_", " ")}
                    </span>
                  </div>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        {p ? (
          <section className="mt-6 rounded-[10px] border border-border bg-background p-5">
            <h2 className="text-base font-medium">Bank transfer</h2>
            <dl className="mt-3 space-y-2 text-sm">
              <div className="flex justify-between gap-3">
                <dt className="text-foreground-secondary">Bank</dt>
                <dd>{p.bank_name_snapshot}</dd>
              </div>
              <div className="flex justify-between gap-3">
                <dt className="text-foreground-secondary">Account name</dt>
                <dd>{p.account_holder_snapshot}</dd>
              </div>
              <div className="flex justify-between gap-3">
                <dt className="text-foreground-secondary">Account number</dt>
                <dd className="font-medium tabular-nums">{p.account_number_snapshot}</dd>
              </div>
              <div className="flex justify-between gap-3">
                <dt className="text-foreground-secondary">Amount</dt>
                <dd className="tabular-nums" style={{ color: accent }}>
                  {formatIDR(p.amount_idr)}
                </dd>
              </div>
            </dl>
            {p.instructions_snapshot ? (
              <p className="mt-3 text-sm text-foreground-secondary whitespace-pre-wrap">{p.instructions_snapshot}</p>
            ) : null}
            {o.payment_deadline_at ? (
              <p className="mt-3 text-xs text-foreground-muted">
                Please transfer before{" "}
                {new Date(o.payment_deadline_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })}{" "}
                (Asia/Jakarta).
              </p>
            ) : null}
          </section>
        ) : null}

        {tracking.can_submit_evidence ? (
          <section className="mt-6 rounded-[10px] border border-border bg-background p-5">
            <h2 className="text-base font-medium">Upload payment proof</h2>
            <p className="mt-2 text-sm text-foreground-secondary">
              Image or PDF, max 8 MB. Stored privately — only you and the merchant can view it.
            </p>
            <div className="mt-4 space-y-2">
              <Label htmlFor="evidence">Payment proof</Label>
              <input
                id="evidence"
                type="file"
                accept="image/png,image/jpeg,image/webp,application/pdf"
                disabled={uploading}
                onChange={(e) => {
                  const f = e.target.files?.[0];
                  if (f) uploadEvidence(f);
                }}
              />
            </div>
            {message ? <p className="mt-3 text-sm text-foreground-secondary">{message}</p> : null}
            {(p?.evidence || []).length ? (
              <ul className="mt-4 space-y-1 text-sm text-foreground-secondary">
                {p!.evidence!.map((ev) => (
                  <li key={ev.id}>
                    Submitted {new Date(ev.submitted_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })}
                    {ev.signed_url ? (
                      <>
                        {" · "}
                        <a className="text-brand-accessible-dark underline" href={ev.signed_url}>
                          View
                        </a>
                      </>
                    ) : null}
                  </li>
                ))}
              </ul>
            ) : null}
          </section>
        ) : (
          <p className="mt-6 text-sm text-foreground-secondary">
            {o.payment_status === "paid"
              ? "Payment verified. The merchant will update fulfillment next."
              : "Evidence upload is not available for this order."}
          </p>
        )}

        {(tracking.contact_email || tracking.whatsapp) && (
          <p className="mt-8 text-sm text-foreground-secondary">
            Need help?{" "}
            {tracking.whatsapp ? (
              <a className="underline" href={`https://wa.me/${tracking.whatsapp.replace(/\D/g, "")}`}>
                WhatsApp
              </a>
            ) : null}
            {tracking.contact_email ? (
              <>
                {tracking.whatsapp ? " · " : null}
                <a className="underline" href={`mailto:${tracking.contact_email}`}>
                  {tracking.contact_email}
                </a>
              </>
            ) : null}
          </p>
        )}
        <div className="mt-6">
          <Button type="button" variant="outline" onClick={load}>
            Refresh status
          </Button>
        </div>
      </div>
    </div>
  );
}
