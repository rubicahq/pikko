"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { ApiError, apiFetch } from "@/lib/api";
import { isValidIndonesianPhone } from "@/lib/phone";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type CreateResult = {
  customer: { id: string; name: string; phone?: string | null };
  portal_url?: string;
  portal_access_url?: string;
};

export function CreateCustomerForm({ storeId }: { storeId: string }) {
  const router = useRouter();
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [created, setCreated] = useState<CreateResult | null>(null);

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const trimmedName = name.trim();
    if (!trimmedName) {
      setError("Name is required.");
      return;
    }
    if (!isValidIndonesianPhone(phone)) {
      setError("Enter a valid Indonesian WhatsApp/phone (e.g. 0812… or +62…).");
      return;
    }
    setBusy(true);
    try {
      const res = await apiFetch<CreateResult>(`/api/v1/stores/${storeId}/customers`, {
        method: "POST",
        body: { name: trimmedName, phone: phone.trim() },
      });
      setCreated(res);
      setName("");
      setPhone("");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to create customer.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4 rounded-xl border border-border bg-background-subtle/60 p-4">
      <div>
        <h2 className="text-base font-medium text-foreground">Add customer</h2>
        <p className="mt-1 text-sm text-foreground-secondary">
          Name and WhatsApp/phone only — no order required. You&apos;ll get a portal link to open or share.
        </p>
      </div>
      <form onSubmit={onSubmit} className="flex flex-wrap items-end gap-3">
        <div className="min-w-[180px] flex-1 space-y-1.5">
          <Label htmlFor="customer-name">Name</Label>
          <Input
            id="customer-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="Customer name"
            autoComplete="name"
            required
          />
        </div>
        <div className="min-w-[180px] flex-1 space-y-1.5">
          <Label htmlFor="customer-phone">Phone / WhatsApp</Label>
          <Input
            id="customer-phone"
            value={phone}
            onChange={(e) => setPhone(e.target.value)}
            placeholder="0812…"
            autoComplete="tel"
            inputMode="tel"
            required
          />
        </div>
        <Button type="submit" disabled={busy}>
          {busy ? "Creating…" : "Create customer"}
        </Button>
      </form>
      {error ? <p className="text-sm text-error">{error}</p> : null}
      {created ? (
        <div className="space-y-2 rounded-lg border border-border bg-background px-4 py-3 text-sm">
          <p className="font-medium text-foreground">
            Created {created.customer.name}
            {created.customer.phone ? ` · ${created.customer.phone}` : ""}
          </p>
          {created.portal_access_url ? (
            <p>
              <a
                href={created.portal_access_url}
                target="_blank"
                rel="noreferrer"
                className="font-medium text-foreground underline underline-offset-2"
              >
                Open customer portal
              </a>
              <span className="text-foreground-muted"> (one-time link)</span>
            </p>
          ) : created.portal_url ? (
            <p>
              <a
                href={created.portal_url}
                target="_blank"
                rel="noreferrer"
                className="font-medium text-foreground underline underline-offset-2"
              >
                Open portal home
              </a>
            </p>
          ) : null}
          <p>
            <a
              href={urls.storeCustomer(storeId, created.customer.id)}
              className="text-foreground-secondary underline underline-offset-2"
            >
              View customer record
            </a>
          </p>
        </div>
      ) : null}
    </div>
  );
}
