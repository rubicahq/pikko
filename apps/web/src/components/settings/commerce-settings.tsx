"use client";

import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { IdrInput } from "@/components/ui/idr-input";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type PaymentMethod = {
  id: string;
  method_type: string;
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

type DokuStatus = {
  checkout_client_id: string;
  mcp_client_id: string;
  checkout_configured: boolean;
  mcp_configured: boolean;
  environment: "sandbox";
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
  const [doku, setDoku] = useState<DokuStatus | null>(null);
  const [dokuCheckoutClientId, setDokuCheckoutClientId] = useState("");
  const [dokuCheckoutSecret, setDokuCheckoutSecret] = useState("");
  const [dokuMcpClientId, setDokuMcpClientId] = useState("");
  const [dokuMcpApiKey, setDokuMcpApiKey] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function reload() {
    const [pm, ful, dokuRes] = await Promise.all([
      apiFetch<{ payment_methods: PaymentMethod[] }>(`/api/v1/stores/${storeId}/payment-methods`),
      apiFetch<{ fulfillment: Fulfillment }>(`/api/v1/stores/${storeId}/fulfillment`),
      apiFetch<{ doku: DokuStatus }>(`/api/v1/stores/${storeId}/doku-credentials`),
    ]);
    setMethods(pm.payment_methods || []);
    setFulfillment(ful.fulfillment);
    setDoku(dokuRes.doku);
    setDokuCheckoutClientId(dokuRes.doku.checkout_client_id || "");
    setDokuMcpClientId(dokuRes.doku.mcp_client_id || "");
  }

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [pm, ful, dokuRes] = await Promise.all([
          apiFetch<{ payment_methods: PaymentMethod[] }>(`/api/v1/stores/${storeId}/payment-methods`),
          apiFetch<{ fulfillment: Fulfillment }>(`/api/v1/stores/${storeId}/fulfillment`),
          apiFetch<{ doku: DokuStatus }>(`/api/v1/stores/${storeId}/doku-credentials`),
        ]);
        if (cancelled) return;
        setMethods(pm.payment_methods || []);
        setFulfillment(ful.fulfillment);
        setDoku(dokuRes.doku);
        setDokuCheckoutClientId(dokuRes.doku.checkout_client_id || "");
        setDokuMcpClientId(dokuRes.doku.mcp_client_id || "");
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

  async function addDokuCheckout() {
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/payment-methods`, {
        method: "POST",
        body: { method_type: "doku_checkout" },
      });
      setMessage("DOKU Checkout enabled for this sandbox store.");
      await reload();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to enable DOKU Checkout.");
    }
  }

  async function saveDokuCredentials(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    try {
      const res = await apiFetch<{ doku: DokuStatus }>(`/api/v1/stores/${storeId}/doku-credentials`, {
        method: "PUT",
        body: {
          checkout_client_id: dokuCheckoutClientId,
          checkout_secret: dokuCheckoutSecret,
          mcp_client_id: dokuMcpClientId,
          mcp_api_key: dokuMcpApiKey,
        },
      });
      setDoku(res.doku);
      setDokuCheckoutSecret("");
      setDokuMcpApiKey("");
      setMessage("Credential DOKU Sandbox tersimpan untuk toko ini.");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save DOKU credentials.");
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
          Live checkout requires an enabled payment method and one fulfillment method.
        </p>
        {message ? <p className="mt-3 text-sm text-success">{message}</p> : null}
        {error ? <p className="mt-3 text-sm text-error">{error}</p> : null}
      </div>

      <section className="space-y-4">
        <div>
          <h3 className="text-base font-medium">DOKU Sandbox credentials</h3>
          <p className="mt-1 text-sm text-foreground-secondary">
            Credential berlaku hanya untuk toko ini. Secret tersimpan terenkripsi dan tidak pernah ditampilkan kembali.
          </p>
        </div>
        <form onSubmit={saveDokuCredentials} className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-2"><Label htmlFor="doku-checkout-client">Checkout Client ID</Label><Input id="doku-checkout-client" value={dokuCheckoutClientId} onChange={(e) => setDokuCheckoutClientId(e.target.value)} required /></div>
          <div className="space-y-2"><Label htmlFor="doku-checkout-secret">Checkout Secret Key</Label><Input id="doku-checkout-secret" type="password" autoComplete="new-password" value={dokuCheckoutSecret} onChange={(e) => setDokuCheckoutSecret(e.target.value)} placeholder={doku?.checkout_configured ? "Masukkan ulang untuk mengganti" : ""} required /></div>
          <div className="space-y-2"><Label htmlFor="doku-mcp-client">MCP Client ID</Label><Input id="doku-mcp-client" value={dokuMcpClientId} onChange={(e) => setDokuMcpClientId(e.target.value)} required /></div>
          <div className="space-y-2"><Label htmlFor="doku-mcp-key">MCP API Key</Label><Input id="doku-mcp-key" type="password" autoComplete="new-password" value={dokuMcpApiKey} onChange={(e) => setDokuMcpApiKey(e.target.value)} placeholder={doku?.mcp_configured ? "Masukkan ulang untuk mengganti" : ""} required /></div>
          <Button type="submit" className="sm:col-span-2 sm:w-fit">Save DOKU credentials</Button>
        </form>
      </section>

      <section className="space-y-4">
        <div className="flex items-center justify-between gap-3">
          <h3 className="text-base font-medium">Payment methods</h3>
          {doku?.checkout_configured && !methods.some((m) => m.method_type === "doku_checkout") ? (
            <Button type="button" variant="outline" onClick={() => void addDokuCheckout()}>
              Enable DOKU Checkout
            </Button>
          ) : null}
        </div>
        <ul className="space-y-2 text-sm">
          {methods.map((m) => (
            <li key={m.id} className="rounded-md border border-border px-3 py-2">
              <span className="font-medium">
                {m.method_type === "doku_checkout" ? "DOKU Checkout" : `${m.bank_name} · ${m.account_number}`}
              </span>
              <span className="mt-1 block text-foreground-secondary">{m.account_holder}</span>
              {!m.is_enabled ? <span className="text-xs text-warning">Disabled</span> : null}
            </li>
          ))}
          {methods.length === 0 ? (
            <li className="text-foreground-secondary">No payment methods yet.</li>
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
            <IdrInput
              id="ship-fee"
              value={fulfillment.shipping_fee_idr}
              onValueChange={(n) =>
                setFulfillment((f) => ({ ...f, shipping_fee_idr: n ?? 0 }))
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
