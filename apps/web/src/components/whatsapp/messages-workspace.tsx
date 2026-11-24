"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Copy, ExternalLink, MessageCircle } from "lucide-react";
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

export type WATemplate = {
  code: string;
  audience: "vendor" | "customer";
  category: string;
  title: string;
  description: string;
  body: string;
  variables: string[];
  surfaces: string[];
};

type Preview = {
  template: WATemplate;
  variables: Record<string, string>;
  missing: string[];
  body: string;
  phone?: string;
  phone_e164?: string;
  deep_link: string;
};

type SendLog = {
  id: string;
  template_code: string;
  audience: string;
  action: string;
  phone_e164?: string | null;
  created_at: string;
};

type Props = {
  storeId: string;
  initialAudience?: "customer" | "vendor" | "all";
  initialCode?: string;
  initialOrderId?: string;
  initialVendorId?: string;
  initialCustomerId?: string;
};

export function MessagesWorkspace({
  storeId,
  initialAudience = "all",
  initialCode,
  initialOrderId,
  initialVendorId,
  initialCustomerId,
}: Props) {
  const [audience, setAudience] = useState<"all" | "customer" | "vendor">(
    initialAudience === "vendor" || initialAudience === "customer" ? initialAudience : "all",
  );
  const [templates, setTemplates] = useState<WATemplate[]>([]);
  const [selected, setSelected] = useState<string | null>(initialCode || null);
  const [orderId, setOrderId] = useState(initialOrderId || "");
  const [vendorId, setVendorId] = useState(initialVendorId || "");
  const [customerId, setCustomerId] = useState(initialCustomerId || "");
  const [phone, setPhone] = useState("");
  const [overrides, setOverrides] = useState<Record<string, string>>({});
  const [preview, setPreview] = useState<Preview | null>(null);
  const [sends, setSends] = useState<SendLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);

  const loadTemplates = useCallback(async () => {
    setError(null);
    try {
      const q = audience === "all" ? "" : `?audience=${audience}`;
      const data = await apiFetch<{ templates: WATemplate[] }>(
        `/api/v1/stores/${storeId}/whatsapp/templates${q}`,
      );
      setTemplates(data.templates || []);
      if (!selected && data.templates?.length) {
        setSelected(data.templates[0].code);
      }
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Failed to load templates.");
    }
  }, [storeId, audience, selected]);

  const loadSends = useCallback(async () => {
    try {
      const data = await apiFetch<{ sends: SendLog[] }>(
        `/api/v1/stores/${storeId}/whatsapp/sends?limit=12`,
      );
      setSends(data.sends || []);
    } catch {
      /* non-blocking */
    }
  }, [storeId]);

  useEffect(() => {
    void loadTemplates();
    void loadSends();
  }, [loadTemplates, loadSends]);

  const selectedTemplate = useMemo(
    () => templates.find((t) => t.code === selected) || null,
    [templates, selected],
  );

  async function runPreview() {
    if (!selected) return;
    setBusy(true);
    setError(null);
    setCopied(false);
    try {
      const body: Record<string, unknown> = { overrides: { ...overrides } };
      if (phone.trim()) (body.overrides as Record<string, string>).phone = phone.trim();
      if (orderId.trim()) body.order_id = orderId.trim();
      if (vendorId.trim()) body.vendor_id = vendorId.trim();
      if (customerId.trim()) body.customer_id = customerId.trim();
      const data = await apiFetch<Preview>(
        `/api/v1/stores/${storeId}/whatsapp/templates/${selected}/preview`,
        { method: "POST", body },
      );
      setPreview(data);
      if (data.phone && !phone) setPhone(data.phone);
      const next: Record<string, string> = { ...overrides };
      for (const [k, v] of Object.entries(data.variables || {})) {
        if (next[k] === undefined) next[k] = v;
      }
      setOverrides(next);
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Preview failed.");
    } finally {
      setBusy(false);
    }
  }

  async function openOrCopy(action: "open" | "copy") {
    if (!selected) return;
    setBusy(true);
    setError(null);
    try {
      const body: Record<string, unknown> = {
        action,
        overrides: { ...overrides },
        body: preview?.body,
        phone: phone.trim() || undefined,
      };
      if (orderId.trim()) body.order_id = orderId.trim();
      if (vendorId.trim()) body.vendor_id = vendorId.trim();
      if (customerId.trim()) body.customer_id = customerId.trim();
      const data = await apiFetch<{ deep_link: string; body: string }>(
        `/api/v1/stores/${storeId}/whatsapp/templates/${selected}/open`,
        { method: "POST", body },
      );
      if (action === "copy") {
        await navigator.clipboard.writeText(data.body);
        setCopied(true);
      } else {
        window.open(data.deep_link, "_blank", "noopener,noreferrer");
      }
      await loadSends();
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "Could not open WhatsApp.");
    } finally {
      setBusy(false);
    }
  }

  useEffect(() => {
    if (selected) void runPreview();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- preview on selection / context ids only
  }, [selected, orderId, vendorId, customerId]);

  return (
    <div className="space-y-8">
      <PageHeader
        title="Messages"
        description="WhatsApp templates with order and vendor variables. Opens wa.me with prefilled text — no Meta Business API required."
      />

      {error ? (
        <p className="rounded-[10px] border border-border bg-white px-4 py-3 text-sm text-error">
          {error}
        </p>
      ) : null}

      <div className="flex flex-wrap gap-2">
        {(
          [
            ["all", "All"],
            ["customer", "Customer"],
            ["vendor", "Vendor"],
          ] as const
        ).map(([id, label]) => (
          <Button
            key={id}
            type="button"
            size="sm"
            variant={audience === id ? "primary" : "secondary"}
            onClick={() => {
              setAudience(id);
              setSelected(null);
              setPreview(null);
            }}
          >
            {label}
          </Button>
        ))}
      </div>

      <div className="grid gap-8 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.1fr)]">
        <section className="min-w-0 space-y-3">
          <h2 className="text-section font-bold text-foreground">Templates</h2>
          {templates.length === 0 ? (
            <EmptyState
              title="No templates"
              description="Templates load from the store registry."
            />
          ) : (
            <div className="overflow-hidden rounded-[14px] border border-border bg-white">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Code</TableHead>
                    <TableHead>Title</TableHead>
                    <TableHead>Audience</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {templates.map((t) => (
                    <TableRow
                      key={t.code}
                      className={
                        selected === t.code
                          ? "bg-surface-active cursor-pointer"
                          : "cursor-pointer hover:bg-surface-hover"
                      }
                      onClick={() => setSelected(t.code)}
                    >
                      <TableCell className="font-medium tabular-nums">{t.code}</TableCell>
                      <TableCell>
                        <div className="font-medium text-foreground">{t.title}</div>
                        <div className="text-caption text-foreground-muted">{t.description}</div>
                      </TableCell>
                      <TableCell>
                        <StatusBadge
                          status={t.audience === "vendor" ? "purchasing" : "confirmed"}
                          label={t.audience}
                        />
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          )}
        </section>

        <section className="min-w-0 space-y-4">
          <h2 className="text-section font-bold text-foreground">Preview & send</h2>
          {!selectedTemplate ? (
            <EmptyState title="Select a template" description="Choose one from the list." />
          ) : (
            <div className="space-y-4 rounded-[14px] border border-border bg-white p-6">
              <div>
                <p className="text-caption font-medium uppercase tracking-wide text-foreground-muted">
                  {selectedTemplate.code} · {selectedTemplate.category}
                </p>
                <h3 className="mt-1 text-card font-semibold text-foreground">
                  {selectedTemplate.title}
                </h3>
              </div>

              <div className="grid gap-3 sm:grid-cols-2">
                <label className="block space-y-1.5">
                  <span className="text-caption text-foreground-secondary">Order ID</span>
                  <Input
                    value={orderId}
                    onChange={(e) => setOrderId(e.target.value)}
                    placeholder="UUID (customer templates)"
                  />
                </label>
                <label className="block space-y-1.5">
                  <span className="text-caption text-foreground-secondary">Vendor ID</span>
                  <Input
                    value={vendorId}
                    onChange={(e) => setVendorId(e.target.value)}
                    placeholder="UUID (vendor templates)"
                  />
                </label>
                <label className="block space-y-1.5">
                  <span className="text-caption text-foreground-secondary">Customer ID</span>
                  <Input
                    value={customerId}
                    onChange={(e) => setCustomerId(e.target.value)}
                    placeholder="Optional"
                  />
                </label>
                <label className="block space-y-1.5">
                  <span className="text-caption text-foreground-secondary">WhatsApp phone</span>
                  <Input
                    value={phone}
                    onChange={(e) => setPhone(e.target.value)}
                    placeholder="08… or 62…"
                  />
                </label>
              </div>

              {selectedTemplate.variables.length > 0 ? (
                <div className="space-y-2">
                  <p className="text-caption font-medium text-foreground-secondary">
                    Variables
                  </p>
                  <div className="grid gap-2 sm:grid-cols-2">
                    {selectedTemplate.variables.map((key) => (
                      <label key={key} className="block space-y-1">
                        <span className="text-caption text-foreground-muted">{`{{${key}}}`}</span>
                        <Input
                          value={overrides[key] ?? ""}
                          onChange={(e) =>
                            setOverrides((prev) => ({ ...prev, [key]: e.target.value }))
                          }
                        />
                      </label>
                    ))}
                  </div>
                </div>
              ) : null}

              <div className="rounded-[12px] border border-border-subtle bg-cloud p-4">
                <p className="mb-2 text-caption font-medium text-foreground-secondary">
                  Message preview
                </p>
                <pre className="whitespace-pre-wrap font-sans text-body text-foreground">
                  {preview?.body || selectedTemplate.body}
                </pre>
                {preview?.missing?.length ? (
                  <p className="mt-3 text-caption text-warning">
                    Missing: {preview.missing.join(", ")}
                  </p>
                ) : null}
              </div>

              <div className="flex flex-wrap gap-2">
                <Button type="button" variant="secondary" disabled={busy} onClick={() => void runPreview()}>
                  Refresh preview
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  disabled={busy}
                  onClick={() => void openOrCopy("copy")}
                >
                  <Copy className="size-4" />
                  {copied ? "Copied" : "Copy text"}
                </Button>
                <Button type="button" disabled={busy} onClick={() => void openOrCopy("open")}>
                  <MessageCircle className="size-4" />
                  Open WhatsApp
                </Button>
                {preview?.deep_link ? (
                  <a
                    className="inline-flex items-center gap-1.5 text-sm text-foreground-secondary underline-offset-2 hover:underline"
                    href={preview.deep_link}
                    target="_blank"
                    rel="noreferrer"
                  >
                    <ExternalLink className="size-3.5" />
                    Deep link
                  </a>
                ) : null}
              </div>
            </div>
          )}

          <div className="space-y-3">
            <h3 className="text-card font-semibold text-foreground">Recent opens</h3>
            {sends.length === 0 ? (
              <p className="text-sm text-foreground-secondary">No template opens logged yet.</p>
            ) : (
              <ul className="space-y-2">
                {sends.map((s) => (
                  <li
                    key={s.id}
                    className="flex flex-wrap items-center justify-between gap-2 rounded-[10px] border border-border-subtle bg-white px-3 py-2 text-sm"
                  >
                    <span className="font-medium tabular-nums text-foreground">{s.template_code}</span>
                    <span className="text-foreground-muted">
                      {s.action}
                      {s.phone_e164 ? ` · ${s.phone_e164}` : ""}
                    </span>
                    <span className="text-caption text-foreground-muted">
                      {new Date(s.created_at).toLocaleString("id-ID", {
                        timeZone: "Asia/Jakarta",
                      })}{" "}
                      WIB
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}
