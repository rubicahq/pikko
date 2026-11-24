"use client";

import { FormEvent, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { PortalShell } from "@/components/portal/portal-shell";
import { usePortalSession } from "@/components/portal/use-portal-session";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

type Address = {
  id: string;
  label?: string | null;
  recipient_name: string;
  phone?: string | null;
  line1: string;
  line2?: string | null;
  city: string;
  province?: string | null;
  postal_code?: string | null;
  country: string;
  is_default: boolean;
};

const emptyForm = {
  label: "",
  recipient_name: "",
  phone: "",
  line1: "",
  city: "",
  province: "",
  postal_code: "",
  is_default: false,
};

export default function PortalAddressesPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const { session, loading: sessionLoading } = usePortalSession(storeSlug);
  const [addresses, setAddresses] = useState<Address[]>([]);
  const [form, setForm] = useState(emptyForm);
  const [editingId, setEditingId] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    const res = await apiFetch<{ addresses: Address[] }>(
      `/api/v1/portal/${storeSlug}/addresses`,
    );
    setAddresses(res.addresses || []);
  }, [storeSlug]);

  useEffect(() => {
    if (session) load().catch((err) => setError(err instanceof Error ? err.message : "Failed"));
  }, [session, load]);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      const body = {
        label: form.label || undefined,
        recipient_name: form.recipient_name,
        phone: form.phone || undefined,
        line1: form.line1,
        city: form.city,
        province: form.province || undefined,
        postal_code: form.postal_code || undefined,
        country: "ID",
        is_default: form.is_default,
      };
      if (editingId) {
        await apiFetch(`/api/v1/portal/${storeSlug}/addresses/${editingId}`, {
          method: "PATCH",
          body,
        });
      } else {
        await apiFetch(`/api/v1/portal/${storeSlug}/addresses`, { body });
      }
      setForm(emptyForm);
      setEditingId(null);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to save address.");
    } finally {
      setSaving(false);
    }
  }

  async function remove(id: string) {
    setError(null);
    try {
      await apiFetch(`/api/v1/portal/${storeSlug}/addresses/${id}`, { method: "DELETE" });
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to delete.");
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
      <h1 className="text-page font-bold tracking-tight text-foreground">Addresses</h1>
      <p className="mt-1.5 text-body text-foreground-secondary">
        Saved addresses for future checkouts. Changing an existing order requires merchant approval.
      </p>

      <ul className="mt-6 space-y-3">
        {addresses.map((a) => (
          <li key={a.id} className="rounded-xl border border-border bg-background p-4">
            <div className="flex items-start justify-between gap-3">
              <div>
                <p className="text-sm font-semibold text-foreground">
                  {a.recipient_name}
                  {a.is_default ? (
                    <span className="ml-2 rounded-sm bg-surface-active px-1.5 py-0.5 text-caption font-medium text-foreground-secondary">
                      Default
                    </span>
                  ) : null}
                </p>
                <p className="mt-1 text-sm text-foreground-secondary">
                  {a.line1}
                  {a.line2 ? `, ${a.line2}` : ""}
                  <br />
                  {a.city}
                  {a.province ? `, ${a.province}` : ""} {a.postal_code || ""}
                </p>
              </div>
              <div className="flex gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => {
                    setEditingId(a.id);
                    setForm({
                      label: a.label || "",
                      recipient_name: a.recipient_name,
                      phone: a.phone || "",
                      line1: a.line1,
                      city: a.city,
                      province: a.province || "",
                      postal_code: a.postal_code || "",
                      is_default: a.is_default,
                    });
                  }}
                >
                  Edit
                </Button>
                <Button variant="ghost" size="sm" onClick={() => remove(a.id)}>
                  Delete
                </Button>
              </div>
            </div>
          </li>
        ))}
      </ul>

      <form onSubmit={onSubmit} className="mt-8 space-y-3 rounded-xl border border-border bg-background p-5">
        <h2 className="text-card font-semibold text-foreground">
          {editingId ? "Edit address" : "Add address"}
        </h2>
        <div className="space-y-1.5">
          <Label htmlFor="recipient">Recipient</Label>
          <Input
            id="recipient"
            required
            value={form.recipient_name}
            onChange={(e) => setForm({ ...form, recipient_name: e.target.value })}
          />
        </div>
        <div className="space-y-1.5">
          <Label htmlFor="line1">Address line</Label>
          <Input
            id="line1"
            required
            value={form.line1}
            onChange={(e) => setForm({ ...form, line1: e.target.value })}
          />
        </div>
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-1.5">
            <Label htmlFor="city">City</Label>
            <Input
              id="city"
              required
              value={form.city}
              onChange={(e) => setForm({ ...form, city: e.target.value })}
            />
          </div>
          <div className="space-y-1.5">
            <Label htmlFor="postal">Postal code</Label>
            <Input
              id="postal"
              value={form.postal_code}
              onChange={(e) => setForm({ ...form, postal_code: e.target.value })}
            />
          </div>
        </div>
        <label className="flex items-center gap-2 text-sm text-foreground-secondary">
          <input
            type="checkbox"
            checked={form.is_default}
            onChange={(e) => setForm({ ...form, is_default: e.target.checked })}
          />
          Set as default
        </label>
        {error ? <p className="text-sm text-error">{error}</p> : null}
        <div className="flex gap-2">
          <Button type="submit" loading={saving}>
            {editingId ? "Save changes" : "Add address"}
          </Button>
          {editingId ? (
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setEditingId(null);
                setForm(emptyForm);
              }}
            >
              Cancel
            </Button>
          ) : null}
        </div>
      </form>
    </PortalShell>
  );
}
