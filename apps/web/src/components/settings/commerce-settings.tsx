"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type PaymentMethod = {
  id: string;
  bank_name: string;
  account_holder: string;
  account_number: string;
  instructions?: string | null;
  payment_deadline_hours: number;
  is_enabled: boolean;
};

type Fulfillment = {
  shipping_enabled: boolean;
  shipping_fee_idr: number;
  free_shipping_enabled: boolean;
  pickup_enabled: boolean;
  pickup_instructions?: string | null;
};

export function CommerceSettings({ storeId }: { storeId: string }) {
  const [methods, setMethods] = useState<PaymentMethod[]>([]);
  const [fulfillment, setFulfillment] = useState<Fulfillment>({
    shipping_enabled: false,
    shipping_fee_idr: 0,
    free_shipping_enabled: false,
    pickup_enabled: false,
  });
  const [bankName, setBankName] = useState("");
  const [holder, setHolder] = useState("");
  const [number, setNumber] = useState("");
  const [instructions, setInstructions] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function reload() {
    const [pm, ful] = await Promise.all([
      apiFetch<{ payment_methods: PaymentMethod[] }>(`/api/v1/stores/${storeId}/payment-methods`),
      apiFetch<{ fulfillment: Fulfillment }>(`/api/v1/stores/${storeId}/fulfillment`),
    ]);
    setMethods(pm.payment_methods || []);
    setFulfillment(ful.fulfillment);
  }

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [pm, ful] = await Promise.all([
          apiFetch<{ payment_methods: PaymentMethod[] }>(`/api/v1/stores/${storeId}/payment-methods`),
          apiFetch<{ fulfillment: Fulfillment }>(`/api/v1/stores/${storeId}/fulfillment`),
        ]);
        if (cancelled) return;
        setMethods(pm.payment_methods || []);
        setFulfillment(ful.fulfillment);
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : "Failed to load.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [storeId]);

  async function addMethod(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/payment-methods`, {
        method: "POST",
        body: {
          bank_name: bankName,
          account_holder: holder,
          account_number: number,
          instructions: instructions || undefined,
        },
      });
      setBankName("");
      setHolder("");
      setNumber("");
      setInstructions("");
      setMessage("Bank account saved. Checkout enables when fulfillment is also configured.");
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save.");
    }
  }

  async function saveFulfillment(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/fulfillment`, {
        method: "PATCH",
        body: fulfillment,
      });
      setMessage("Fulfillment settings saved.");
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save.");
    }
  }

  return (
    <div className="mt-10 space-y-10 border-t border-border pt-10">
      <div>
        <h2 className="text-xl font-medium text-foreground">Payments & fulfillment</h2>
        <p className="mt-2 text-sm text-foreground-secondary">
          Live checkout requires at least one enabled bank account and one fulfillment method.
        </p>
        {message ? <p className="mt-3 text-sm text-success">{message}</p> : null}
        {error ? <p className="mt-3 text-sm text-error">{error}</p> : null}
      </div>

      <section className="space-y-4">
        <h3 className="text-base font-medium">Bank transfer accounts</h3>
        <ul className="space-y-2 text-sm">
          {methods.map((m) => (
            <li key={m.id} className="rounded-md border border-border px-3 py-2">
              <span className="font-medium">
                {m.bank_name} · {m.account_number}
              </span>
              <span className="mt-1 block text-foreground-secondary">{m.account_holder}</span>
              {!m.is_enabled ? <span className="text-xs text-warning">Disabled</span> : null}
            </li>
          ))}
          {methods.length === 0 ? (
            <li className="text-foreground-secondary">No bank accounts yet.</li>
          ) : null}
        </ul>
        <form onSubmit={addMethod} className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor="bank">Bank name</Label>
            <Input id="bank" value={bankName} onChange={(e) => setBankName(e.target.value)} required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="holder">Account holder</Label>
            <Input id="holder" value={holder} onChange={(e) => setHolder(e.target.value)} required />
          </div>
          <div className="space-y-2">
            <Label htmlFor="number">Account number</Label>
            <Input id="number" value={number} onChange={(e) => setNumber(e.target.value)} required />
          </div>
          <div className="space-y-2 sm:col-span-2">
            <Label htmlFor="instructions">Instructions (optional)</Label>
            <Input
              id="instructions"
              value={instructions}
              onChange={(e) => setInstructions(e.target.value)}
            />
          </div>
          <Button type="submit">Add bank account</Button>
        </form>
      </section>

      <form onSubmit={saveFulfillment} className="space-y-4">
        <h3 className="text-base font-medium">Fulfillment</h3>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={fulfillment.shipping_enabled}
            onChange={(e) =>
              setFulfillment((f) => ({ ...f, shipping_enabled: e.target.checked }))
            }
          />
          Flat-rate shipping
        </label>
        {fulfillment.shipping_enabled ? (
          <div className="space-y-2">
            <Label htmlFor="ship-fee">Shipping fee (IDR)</Label>
            <Input
              id="ship-fee"
              type="number"
              min={0}
              value={fulfillment.shipping_fee_idr}
              onChange={(e) =>
                setFulfillment((f) => ({ ...f, shipping_fee_idr: Number(e.target.value) || 0 }))
              }
            />
          </div>
        ) : null}
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={fulfillment.free_shipping_enabled}
            onChange={(e) =>
              setFulfillment((f) => ({ ...f, free_shipping_enabled: e.target.checked }))
            }
          />
          Free shipping
        </label>
        <label className="flex items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={fulfillment.pickup_enabled}
            onChange={(e) => setFulfillment((f) => ({ ...f, pickup_enabled: e.target.checked }))}
          />
          Customer pickup
        </label>
        {fulfillment.pickup_enabled ? (
          <div className="space-y-2">
            <Label htmlFor="pickup">Pickup instructions</Label>
            <Input
              id="pickup"
              value={fulfillment.pickup_instructions || ""}
              onChange={(e) =>
                setFulfillment((f) => ({ ...f, pickup_instructions: e.target.value }))
              }
            />
          </div>
        ) : null}
        <Button type="submit">Save fulfillment</Button>
      </form>
    </div>
  );
}
