"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { PortalShell } from "@/components/portal/portal-shell";
import { usePortalSession } from "@/components/portal/use-portal-session";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Pref = {
  preferred_method: string;
  bank_name?: string | null;
  account_holder?: string | null;
  account_number?: string | null;
};

export default function PortalRefundsPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const { session, loading: sessionLoading } = usePortalSession(storeSlug);
  const [method, setMethod] = useState("bank_transfer");
  const [bankName, setBankName] = useState("");
  const [holder, setHolder] = useState("");
  const [number, setNumber] = useState("");
  const [creditAvailable, setCreditAvailable] = useState(false);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const [prefRes, dashRes] = await Promise.all([
      apiFetch<{ refund_preference: Pref | null }>(
        `/api/v1/portal/${storeSlug}/refund-preference`,
      ),
      apiFetch<{ dashboard: { store_credit_enabled: boolean } }>(
        `/api/v1/portal/${storeSlug}/dashboard`,
      ),
    ]);
    setCreditAvailable(!!dashRes.dashboard.store_credit_enabled);
    const pref = prefRes.refund_preference;
    if (pref) {
      setMethod(pref.preferred_method);
      setBankName(pref.bank_name || "");
      setHolder(pref.account_holder || "");
      setNumber(pref.account_number || "");
    }
  }, [storeSlug]);

  useEffect(() => {
    if (session) load().catch((err) => setError(err instanceof Error ? err.message : "Failed"));
  }, [session, load]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    setMessage(null);
    try {
      await apiFetch(`/api/v1/portal/${storeSlug}/refund-preference`, {
        method: "PUT",
        body: {
          preferred_method: method,
          bank_name: method === "bank_transfer" ? bankName : undefined,
          account_holder: method === "bank_transfer" ? holder : undefined,
          account_number: method === "bank_transfer" ? number : undefined,
        },
      });
      setMessage("Refund preference saved.");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to save.");
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
      <h1 className="text-page font-bold tracking-tight text-foreground">Refund preferences</h1>
      <p className="mt-1.5 text-body text-foreground-secondary">
        Choose how you want refunds for unavailable items or overpayments. Bank details are only used
        after you verify via magic link.
      </p>

      <form onSubmit={onSubmit} className="mt-8 space-y-5 rounded-xl border border-border bg-background p-5">
        <fieldset className="space-y-3">
          <legend className="text-sm font-semibold text-foreground">Preferred method</legend>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="radio"
              name="method"
              checked={method === "bank_transfer"}
              onChange={() => setMethod("bank_transfer")}
            />
            Bank transfer
          </label>
          {creditAvailable ? (
            <label className="flex items-center gap-2 text-sm">
              <input
                type="radio"
                name="method"
                checked={method === "store_credit"}
                onChange={() => setMethod("store_credit")}
              />
              Store credit
            </label>
          ) : null}
        </fieldset>

        {method === "bank_transfer" ? (
          <div className="space-y-3">
            <div className="space-y-1.5">
              <Label htmlFor="bank">Bank name</Label>
              <Input
                id="bank"
                required
                value={bankName}
                onChange={(e) => setBankName(e.target.value)}
                placeholder="BCA"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="holder">Account holder</Label>
              <Input
                id="holder"
                required
                value={holder}
                onChange={(e) => setHolder(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="number">Account number</Label>
              <Input
                id="number"
                required
                value={number}
                onChange={(e) => setNumber(e.target.value)}
                inputMode="numeric"
              />
            </div>
          </div>
        ) : null}

        {error ? <p className="text-sm text-error">{error}</p> : null}
        {message ? <p className="text-sm text-success">{message}</p> : null}
        <Button type="submit" loading={saving}>
          Save preference
        </Button>
      </form>
    </PortalShell>
  );
}
