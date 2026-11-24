"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, apiUpload, type Store, type StorefrontConfig } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function StoreSettingsForm({
  store,
  storefront,
}: {
  store: Store;
  storefront: StorefrontConfig;
}) {
  const router = useRouter();
  const [name, setName] = useState(store.name);
  const [cfg, setCfg] = useState(storefront);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  async function saveIdentity(e: React.FormEvent) {
    e.preventDefault();
    setSaving(true);
    setError(null);
    try {
      await apiFetch(`/api/v1/stores/${store.id}`, {
        method: "PATCH",
        body: { name },
      });
      await apiFetch(`/api/v1/stores/${store.id}/storefront`, {
        method: "PATCH",
        body: {
          description: cfg.description,
          contact_email: cfg.contact_email,
          whatsapp: cfg.whatsapp,
        },
      });
      setMessage("Store identity saved to draft storefront.");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save.");
    } finally {
      setSaving(false);
    }
  }

  async function upload(purpose: "logo" | "favicon" | "store_icon", file: File) {
    const form = new FormData();
    form.set("purpose", purpose);
    form.set("file", file);
    const res = await apiUpload<{ file: { id: string } }>(
      `/api/v1/stores/${store.id}/files`,
      form,
    );
    const key =
      purpose === "logo"
        ? "logo_file_id"
        : purpose === "favicon"
          ? "favicon_file_id"
          : "store_icon_file_id";
    const updated = await apiFetch<{ storefront: StorefrontConfig }>(
      `/api/v1/stores/${store.id}/storefront`,
      { method: "PATCH", body: { [key]: res.file.id } },
    );
    setCfg(updated.storefront);
    setMessage(`${purpose.replace("_", " ")} uploaded to draft.`);
    router.refresh();
  }

  return (
    <form onSubmit={saveIdentity} className="mx-auto max-w-reading space-y-6">
      <div>
        <h1 className="text-[30px] font-medium text-foreground">Store settings</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Identity, logo, favicon, and store icon. Branding changes stay in draft until you publish
          the storefront.
        </p>
      </div>

      <div className="space-y-2">
        <Label htmlFor="store-name">Store name</Label>
        <Input id="store-name" value={name} onChange={(e) => setName(e.target.value)} required />
      </div>
      <div className="space-y-2">
        <Label>Storefront URL</Label>
        <p className="text-sm text-foreground-secondary">{store.storefront_url}</p>
      </div>
      <div className="space-y-2">
        <Label htmlFor="description">Description</Label>
        <textarea
          id="description"
          className="min-h-24 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
          value={cfg.description || ""}
          onChange={(e) => setCfg((c) => ({ ...c, description: e.target.value }))}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="contact_email">Contact email</Label>
        <Input
          id="contact_email"
          type="email"
          value={cfg.contact_email || ""}
          onChange={(e) => setCfg((c) => ({ ...c, contact_email: e.target.value }))}
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="whatsapp">WhatsApp</Label>
        <Input
          id="whatsapp"
          value={cfg.whatsapp || ""}
          onChange={(e) => setCfg((c) => ({ ...c, whatsapp: e.target.value }))}
        />
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        {(
          [
            ["logo", "Logo", cfg.logo_url],
            ["favicon", "Favicon", cfg.favicon_url],
            ["store_icon", "Store icon", cfg.store_icon_url],
          ] as const
        ).map(([purpose, label, preview]) => (
          <div key={purpose} className="space-y-2 rounded-lg border border-border p-4">
            <Label htmlFor={`settings-${purpose}`}>{label}</Label>
            {preview ? (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={preview} alt="" className="h-16 w-16 rounded-md border border-border object-contain" />
            ) : (
              <div className="flex h-16 w-16 items-center justify-center rounded-md border border-dashed border-border text-xs text-foreground-muted">
                None
              </div>
            )}
            <Input
              id={`settings-${purpose}`}
              type="file"
              accept="image/png,image/jpeg,image/webp,image/svg+xml,image/x-icon"
              onChange={async (e) => {
                const file = e.target.files?.[0];
                if (!file) return;
                try {
                  await upload(purpose, file);
                } catch (err) {
                  setError(err instanceof Error ? err.message : "Upload failed.");
                }
              }}
            />
          </div>
        ))}
      </div>

      {message && <p className="text-sm text-success">{message}</p>}
      {error && <p className="text-sm text-error">{error}</p>}
      <Button type="submit" loading={saving}>
        Save identity
      </Button>
    </form>
  );
}
