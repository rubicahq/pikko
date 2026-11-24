"use client";

import { useCallback, useEffect, useState } from "react";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type Vendor = {
  id: string;
  name: string;
  slug: string;
  contact_name?: string | null;
  contact_email?: string | null;
  contact_phone?: string | null;
  notes?: string | null;
  status: string;
};

export function VendorsPanel({ storeId }: { storeId: string }) {
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [name, setName] = useState("");
  const [contact, setContact] = useState("");
  const [phone, setPhone] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setError(null);
    try {
      const data = await apiFetch<{ vendors: Vendor[] }>(
        `/api/v1/stores/${storeId}/vendors?include_archived=1`,
      );
      setVendors(data.vendors || []);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load vendors.");
    }
  }, [storeId]);

  useEffect(() => {
    void load();
  }, [load]);

  async function createVendor() {
    if (!name.trim()) {
      setError("Vendor name is required.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${storeId}/vendors`, {
        method: "POST",
        body: {
          name: name.trim(),
          contact_name: contact.trim() || undefined,
          contact_phone: phone.trim() || undefined,
        },
      });
      setName("");
      setContact("");
      setPhone("");
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Create failed.");
    } finally {
      setBusy(false);
    }
  }

  async function archiveVendor(id: string) {
    setBusy(true);
    setError(null);
    try {
      const cur = vendors.find((v) => v.id === id);
      if (!cur) return;
      await apiFetch(`/api/v1/stores/${storeId}/vendors/${id}`, {
        method: "PATCH",
        body: { name: cur.name, status: "archived" },
      });
      await load();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Archive failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Vendors"
        description="Preferred suppliers for purchasing splits and receiving handoff."
      />

      {error ? (
        <p className="rounded-lg border border-error-border bg-error-bg px-4 py-3 text-sm text-error">
          {error}
        </p>
      ) : null}

      <div className="flex flex-wrap items-end gap-3 rounded-xl border border-border bg-background-subtle/60 p-4">
        <label className="space-y-1 text-sm">
          <span className="text-foreground-muted">Name</span>
          <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="Supplier name" />
        </label>
        <label className="space-y-1 text-sm">
          <span className="text-foreground-muted">Contact</span>
          <Input value={contact} onChange={(e) => setContact(e.target.value)} placeholder="Optional" />
        </label>
        <label className="space-y-1 text-sm">
          <span className="text-foreground-muted">Phone</span>
          <Input value={phone} onChange={(e) => setPhone(e.target.value)} placeholder="Optional" />
        </label>
        <Button type="button" disabled={busy} onClick={() => void createVendor()}>
          Add vendor
        </Button>
      </div>

      {vendors.length === 0 ? (
        <EmptyState
          title="No vendors yet"
          description="Add suppliers so purchasing lists can split demand by vendor."
        />
      ) : (
        <Table stickyHeader>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead sticky>Name</TableHead>
              <TableHead sticky>Contact</TableHead>
              <TableHead sticky>Phone</TableHead>
              <TableHead sticky>Status</TableHead>
              <TableHead sticky />
            </TableRow>
          </TableHeader>
          <TableBody>
            {vendors.map((v) => (
              <TableRow key={v.id}>
                <TableCell className="font-medium">{v.name}</TableCell>
                <TableCell>{v.contact_name || "—"}</TableCell>
                <TableCell>{v.contact_phone || "—"}</TableCell>
                <TableCell>
                  <StatusBadge status={v.status} />
                </TableCell>
                <TableCell className="text-right">
                  {v.status === "active" ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      disabled={busy}
                      onClick={() => void archiveVendor(v.id)}
                    >
                      Archive
                    </Button>
                  ) : null}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
