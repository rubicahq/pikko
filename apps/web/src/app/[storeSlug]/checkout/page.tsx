"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { ApiError, apiFetch, formatIDR } from "@/lib/api";
import { cartGroups, clearCart, onCartChanged, readCart, type CartLine } from "@/lib/cart";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type PaymentMethod = {
  id: string;
  method_type: string;
  bank_name: string;
  account_holder: string;
  account_number: string;
  instructions?: string | null;
};

type Fulfillment = {
  shipping_enabled: boolean;
  shipping_fee_idr: number;
  free_shipping_enabled: boolean;
  pickup_enabled: boolean;
  pickup_instructions?: string | null;
};

type Preview = {
  subtotal_idr: number;
  shipping_fee_idr: number;
  handling_fee_idr?: number;
  total_idr: number;
  selling_model?: string;
  batch_name?: string;
  requires_approval?: boolean;
  minimum_ok?: boolean;
  minimum_message?: string;
  checkout_enabled: boolean;
  payment_methods: PaymentMethod[];
  fulfillment?: Fulfillment | null;
  lines: { product_name: string; quantity: number; line_total_idr: number }[];
};

function CheckoutInner() {
  const params = useParams();
  const search = useSearchParams();
  const router = useRouter();
  const storeSlug = String(params.storeSlug || "");
  const groupKey = search.get("group") || "";
  const [lines, setLines] = useState<CartLine[]>([]);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [phone, setPhone] = useState("");
  const [method, setMethod] = useState("");
  const [paymentMethodId, setPaymentMethodId] = useState("");
  const [notes, setNotes] = useState("");
  const [recipient, setRecipient] = useState("");
  const [line1, setLine1] = useState("");
  const [city, setCity] = useState("");
  const [province, setProvince] = useState("");
  const [postal, setPostal] = useState("");

  const activeLines = useMemo(() => {
    if (!groupKey) {
      const groups = cartGroups(storeSlug);
      return groups[0]?.lines || lines;
    }
    return lines.filter((l) => {
      const key = l.batchId ? `preorder:${l.batchId}` : "ready_stock";
      return key === groupKey;
    });
  }, [lines, groupKey, storeSlug]);

  const activeGroupKey = groupKey || (activeLines[0]?.batchId ? `preorder:${activeLines[0].batchId}` : "ready_stock");

  const itemsPayload = useMemo(
    () =>
      activeLines.map((l) => ({
        product_id: l.productId,
        variant_id: l.variantId || undefined,
        quantity: l.quantity,
        batch_id: l.batchId || undefined,
      })),
    [activeLines],
  );

  useEffect(() => {
    setLines(readCart(storeSlug));
    return onCartChanged(storeSlug, setLines);
  }, [storeSlug]);

  useEffect(() => {
    if (!activeLines.length) {
      setPreview(null);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const res = await apiFetch<{ preview: Preview }>(
          `/api/v1/public/stores/${storeSlug}/checkout/preview`,
          {
            method: "POST",
            body: { items: itemsPayload, fulfillment_method: method || undefined },
          },
        );
        if (cancelled) return;
        setPreview(res.preview);
        setError(null);
        if (!paymentMethodId && res.preview.payment_methods?.[0]) {
          setPaymentMethodId(res.preview.payment_methods[0].id);
        }
        if (!method && res.preview.fulfillment) {
          const f = res.preview.fulfillment;
          if (f.shipping_enabled) setMethod("shipping");
          else if (f.free_shipping_enabled) setMethod("free_shipping");
          else if (f.pickup_enabled) setMethod("pickup");
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Unable to preview checkout.");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [storeSlug, itemsPayload, method, paymentMethodId, activeLines.length]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!preview || !paymentMethodId || !method) return;
    setSubmitting(true);
    setError(null);
    const idempotency = crypto.randomUUID();
    try {
      const needsAddress = method === "shipping" || method === "free_shipping";
      const res = await apiFetch<{
        order: { id: string };
        tracking_token: string;
        tracking_url: string;
		checkout_url?: string;
      }>(`/api/v1/public/stores/${storeSlug}/checkout/submit`, {
        method: "POST",
        body: {
          items: itemsPayload,
          customer_name: name,
          customer_email: email,
          customer_phone: phone || undefined,
          fulfillment_method: method,
          shipping_address: needsAddress
            ? {
                recipient_name: recipient || name,
                phone: phone || undefined,
                line1,
                city,
                province: province || undefined,
                postal_code: postal || undefined,
                country: "ID",
              }
            : undefined,
          customer_notes: notes || undefined,
          payment_method_id: paymentMethodId,
          idempotency_key: idempotency,
          client_expected_total_idr: preview.total_idr,
        },
      });
      clearCart(storeSlug, activeGroupKey);
	  if (res.checkout_url) {
		window.location.assign(res.checkout_url);
		return;
	  }
      router.push(`/track/${res.tracking_token}`);
    } catch (err) {
      if (err instanceof ApiError && err.code === "price_changed") {
        setError("Prices changed. Please review the updated total and try again.");
      } else {
        setError(err instanceof Error ? err.message : "Checkout failed.");
      }
    } finally {
      setSubmitting(false);
    }
  }

  if (!activeLines.length) {
    return (
      <div className="mx-auto max-w-reading px-4 py-16">
        <h1 className="text-[30px] font-medium">Checkout</h1>
        <p className="mt-3 text-sm text-foreground-secondary">Your cart is empty.</p>
        <Link href={`/${storeSlug}`} className="mt-4 inline-block text-sm font-medium text-brand-accessible-dark">
          Continue shopping
        </Link>
      </div>
    );
  }

  const f = preview?.fulfillment;
  const needsAddress = method === "shipping" || method === "free_shipping";

  return (
    <div className="mx-auto min-h-screen max-w-[1120px] px-4 py-8 md:px-8">
      <Link href={`/${storeSlug}/cart`} className="text-sm text-foreground-secondary hover:text-foreground">
        ← Back to cart
      </Link>
      <h1 className="mt-2 text-[30px] font-medium text-foreground">Checkout</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Guest checkout — no account needed.
        {preview?.selling_model === "preorder" && preview.batch_name
          ? ` Preorder batch: ${preview.batch_name}.`
          : ""}
        {preview?.requires_approval ? " This order needs merchant confirmation before payment." : ""}
      </p>

      {error ? (
        <p className="mt-4 rounded-md border border-error/30 bg-error/5 px-3 py-2 text-sm text-error" role="alert">
          {error}
        </p>
      ) : null}

      {!preview?.checkout_enabled ? (
        <p className="mt-6 text-sm text-foreground-secondary">
          This store is not accepting orders yet. Payment or fulfillment settings may still be incomplete.
        </p>
      ) : (
        <form onSubmit={submit} className="mt-8 grid gap-8 lg:grid-cols-[1fr_360px]">
          <div className="space-y-6">
            <section className="space-y-4">
              <h2 className="text-lg font-medium">Customer</h2>
              <div className="space-y-2">
                <Label htmlFor="name">Full name</Label>
                <Input id="name" value={name} onChange={(e) => setName(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="email">Email</Label>
                <Input id="email" type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
              </div>
              <div className="space-y-2">
                <Label htmlFor="phone">Phone / WhatsApp</Label>
                <Input id="phone" value={phone} onChange={(e) => setPhone(e.target.value)} />
              </div>
            </section>

            <section className="space-y-3">
              <h2 className="text-lg font-medium">Fulfillment</h2>
              {f?.shipping_enabled ? (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="radio"
                    name="fulfillment"
                    checked={method === "shipping"}
                    onChange={() => setMethod("shipping")}
                  />
                  Shipping ({formatIDR(f.shipping_fee_idr)})
                </label>
              ) : null}
              {f?.free_shipping_enabled ? (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="radio"
                    name="fulfillment"
                    checked={method === "free_shipping"}
                    onChange={() => setMethod("free_shipping")}
                  />
                  Free shipping
                </label>
              ) : null}
              {f?.pickup_enabled ? (
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="radio"
                    name="fulfillment"
                    checked={method === "pickup"}
                    onChange={() => setMethod("pickup")}
                  />
                  Pickup
                </label>
              ) : null}
            </section>

            {needsAddress ? (
              <section className="space-y-4">
                <h2 className="text-lg font-medium">Shipping address</h2>
                <div className="space-y-2">
                  <Label htmlFor="recipient">Recipient</Label>
                  <Input id="recipient" value={recipient} onChange={(e) => setRecipient(e.target.value)} required />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="line1">Address</Label>
                  <Input id="line1" value={line1} onChange={(e) => setLine1(e.target.value)} required />
                </div>
                <div className="grid gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="city">City</Label>
                    <Input id="city" value={city} onChange={(e) => setCity(e.target.value)} required />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="province">Province</Label>
                    <Input id="province" value={province} onChange={(e) => setProvince(e.target.value)} />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label htmlFor="postal">Postal code</Label>
                  <Input id="postal" value={postal} onChange={(e) => setPostal(e.target.value)} />
                </div>
              </section>
            ) : null}

            <section className="space-y-3">
              <h2 className="text-lg font-medium">Payment</h2>
              <p className="text-sm text-foreground-secondary">Choose DOKU Checkout or manual bank transfer.</p>
              {(preview?.payment_methods || []).map((pm) => (
                <label key={pm.id} className="flex items-start gap-2 rounded-md border border-border p-3 text-sm">
                  <input
                    type="radio"
                    name="payment"
                    className="mt-1"
                    checked={paymentMethodId === pm.id}
                    onChange={() => setPaymentMethodId(pm.id)}
                  />
                  <span>
                    <span className="font-medium">
                      {pm.method_type === "doku_checkout" ? "DOKU Checkout" : `${pm.bank_name} · ${pm.account_number}`}
                    </span>
                    <span className="mt-1 block text-foreground-secondary">{pm.account_holder}</span>
                  </span>
                </label>
              ))}
            </section>

            <div className="space-y-2">
              <Label htmlFor="notes">Order notes (optional)</Label>
              <textarea
                id="notes"
                className="min-h-20 w-full rounded-md border border-border px-3 py-2 text-sm"
                value={notes}
                onChange={(e) => setNotes(e.target.value)}
              />
            </div>
          </div>

          <aside className="h-fit rounded-[10px] border border-border p-5">
            <h2 className="text-base font-medium">Summary</h2>
            <ul className="mt-4 space-y-2 text-sm">
              {(preview?.lines || []).map((l, i) => (
                <li key={i} className="flex justify-between gap-3">
                  <span>
                    {l.product_name} × {l.quantity}
                  </span>
                  <span className="tabular-nums">{formatIDR(l.line_total_idr)}</span>
                </li>
              ))}
            </ul>
            <div className="mt-4 space-y-1 border-t border-border pt-4 text-sm">
              <div className="flex justify-between">
                <span>Subtotal</span>
                <span className="tabular-nums">{formatIDR(preview?.subtotal_idr || 0)}</span>
              </div>
              <div className="flex justify-between">
                <span>Handling fee</span>
                <span className="tabular-nums">{formatIDR(preview?.handling_fee_idr || 0)}</span>
              </div>
              <div className="flex justify-between">
                <span>Shipping</span>
                <span className="tabular-nums">{formatIDR(preview?.shipping_fee_idr || 0)}</span>
              </div>
              <div className="flex justify-between text-base font-medium">
                <span>Total</span>
                <span className="tabular-nums">{formatIDR(preview?.total_idr || 0)}</span>
              </div>
              {preview?.minimum_ok === false ? (
                <p className="pt-2 text-error">{preview.minimum_message || "Minimum not met."}</p>
              ) : null}
            </div>
            <Button
              type="submit"
              className="mt-6 w-full"
              disabled={
                submitting || !method || !paymentMethodId || preview?.minimum_ok === false
              }
            >
              {submitting ? "Placing order…" : "Place order"}
            </Button>
          </aside>
        </form>
      )}
    </div>
  );
}

export default function CheckoutPage() {
  return (
    <Suspense fallback={<div className="px-4 py-16 text-sm text-foreground-secondary">Loading checkout…</div>}>
      <CheckoutInner />
    </Suspense>
  );
}
