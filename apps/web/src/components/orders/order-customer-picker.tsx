"use client";

import { useEffect, useState } from "react";
import { ApiError, apiFetch } from "@/lib/api";
import { isValidIndonesianPhone } from "@/lib/phone";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

export type OrderCustomer = {
  id: string;
  name: string;
  email?: string | null;
  phone?: string | null;
};

type Props = {
  storeId: string;
  customer: OrderCustomer | null;
  onSelect: (customer: OrderCustomer | null) => void;
};

export function OrderCustomerPicker({ storeId, customer, onSelect }: Props) {
  const [query, setQuery] = useState("");
  const [hits, setHits] = useState<OrderCustomer[]>([]);
  const [open, setOpen] = useState(false);
  const [addOpen, setAddOpen] = useState(false);
  const [addName, setAddName] = useState("");
  const [addPhone, setAddPhone] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!open) return;
    const q = query.trim();
    if (q.length < 1) {
      setHits([]);
      return;
    }
    const t = setTimeout(() => {
      void apiFetch<{ customers: OrderCustomer[] }>(
        `/api/v1/stores/${storeId}/customers?q=${encodeURIComponent(q)}&limit=12`,
      )
        .then((res) => setHits(res.customers || []))
        .catch(() => setHits([]));
    }, 200);
    return () => clearTimeout(t);
  }, [query, storeId, open]);

  function pick(c: OrderCustomer) {
    onSelect(c);
    setQuery("");
    setHits([]);
    setOpen(false);
    setError(null);
  }

  function clear() {
    onSelect(null);
    setQuery("");
    setHits([]);
    setError(null);
  }

  async function createCustomer(e: React.FormEvent) {
    e.preventDefault();
    setError(null);
    const trimmedName = addName.trim();
    if (!trimmedName) {
      setError("Name is required.");
      return;
    }
    if (!isValidIndonesianPhone(addPhone)) {
      setError("Enter a valid Indonesian WhatsApp/phone (e.g. 0812… or +62…).");
      return;
    }
    setBusy(true);
    try {
      const res = await apiFetch<{ customer: OrderCustomer }>(
        `/api/v1/stores/${storeId}/customers`,
        { method: "POST", body: { name: trimmedName, phone: addPhone.trim() } },
      );
      pick(res.customer);
      setAddOpen(false);
      setAddName("");
      setAddPhone("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to create customer.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-2 sm:col-span-2 lg:col-span-3">
      <Label htmlFor="order-customer-search">Customer</Label>
      {customer ? (
        <div className="flex flex-wrap items-center gap-2 rounded-md border border-border bg-background-subtle/60 px-3 py-2">
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-foreground">{customer.name}</p>
            <p className="truncate text-caption text-foreground-muted">
              {[customer.phone, customer.email].filter(Boolean).join(" · ") || "No phone or email"}
            </p>
          </div>
          <Button type="button" variant="ghost" size="sm" onClick={clear}>
            Change
          </Button>
        </div>
      ) : (
        <div className="relative">
          <div className="flex flex-wrap gap-2">
            <Input
              id="order-customer-search"
              value={query}
              onChange={(e) => {
                setQuery(e.target.value);
                setOpen(true);
              }}
              onFocus={() => setOpen(true)}
              placeholder="Search name, phone, or email…"
              className="min-w-[200px] flex-1"
              autoComplete="off"
            />
            <Button type="button" variant="secondary" onClick={() => setAddOpen(true)}>
              Add customer
            </Button>
          </div>
          {open && query.trim().length > 0 ? (
            <ul className="absolute z-20 mt-1 max-h-56 w-full overflow-auto rounded-md border border-border bg-background shadow-sm">
              {hits.length === 0 ? (
                <li className="px-3 py-2 text-sm text-foreground-muted">
                  No matches.{" "}
                  <button
                    type="button"
                    className="font-medium text-foreground underline underline-offset-2"
                    onClick={() => {
                      setAddName(query.trim());
                      setAddOpen(true);
                    }}
                  >
                    Add new customer
                  </button>
                </li>
              ) : (
                hits.map((c) => (
                  <li key={c.id}>
                    <button
                      type="button"
                      className="flex w-full flex-col gap-0.5 px-3 py-2 text-left hover:bg-background-subtle"
                      onClick={() => pick(c)}
                    >
                      <span className="text-sm font-medium text-foreground">{c.name}</span>
                      <span className="text-caption text-foreground-muted">
                        {[c.phone, c.email].filter(Boolean).join(" · ") || "No phone or email"}
                      </span>
                    </button>
                  </li>
                ))
              )}
            </ul>
          ) : null}
        </div>
      )}

      <Dialog open={addOpen} onOpenChange={setAddOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Add customer</DialogTitle>
            <DialogDescription>
              Name and WhatsApp/phone only. The new customer is selected for this order.
            </DialogDescription>
          </DialogHeader>
          <form onSubmit={createCustomer} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="new-customer-name">Name</Label>
              <Input
                id="new-customer-name"
                value={addName}
                onChange={(e) => setAddName(e.target.value)}
                required
                autoComplete="name"
              />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="new-customer-phone">Phone / WhatsApp</Label>
              <Input
                id="new-customer-phone"
                value={addPhone}
                onChange={(e) => setAddPhone(e.target.value)}
                placeholder="0812…"
                inputMode="tel"
                required
                autoComplete="tel"
              />
            </div>
            {error ? <p className="text-sm text-error">{error}</p> : null}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="secondary" onClick={() => setAddOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={busy}>
                {busy ? "Creating…" : "Create & select"}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
