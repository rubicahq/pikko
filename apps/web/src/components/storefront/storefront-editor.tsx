"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import {
  apiFetch,
  apiUpload,
  type Store,
  type StorefrontConfig,
  type Theme,
} from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { safeCssColor } from "@/lib/safe-css-color";
import { urls } from "@/lib/urls";

export function StorefrontEditor({
  store,
  draft,
  themes,
}: {
  store: Store;
  draft: StorefrontConfig;
  themes: Theme[];
}) {
  const router = useRouter();
  const [cfg, setCfg] = useState(draft);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const theme = useMemo(
    () => themes.find((t) => t.id === cfg.theme_id) || themes[0],
    [themes, cfg.theme_id],
  );

  async function patch(body: Record<string, unknown>) {
    setSaving(true);
    setError(null);
    try {
      const res = await apiFetch<{ storefront: StorefrontConfig }>(
        `/api/v1/stores/${store.id}/storefront`,
        { method: "PATCH", body },
      );
      setCfg(res.storefront);
      setMessage("Draft saved.");
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Unable to save.");
    } finally {
      setSaving(false);
    }
  }

  async function uploadAsset(purpose: "logo" | "favicon" | "store_icon", file: File) {
    const form = new FormData();
    form.set("purpose", purpose);
    form.set("file", file);
    const res = await apiUpload<{ file: { id: string; url?: string } }>(
      `/api/v1/stores/${store.id}/files`,
      form,
    );
    const key =
      purpose === "logo"
        ? "logo_file_id"
        : purpose === "favicon"
          ? "favicon_file_id"
          : "store_icon_file_id";
    await patch({ [key]: res.file.id });
  }

  async function publish() {
    if (saving) return;
    setSaving(true);
    setError(null);
    setMessage(null);
    try {
      await apiFetch(`/api/v1/stores/${store.id}/storefront/publish`, { method: "POST" });
      setMessage(`Storefront published. Live at ${urls.storefront(store.slug)}`);
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Publish failed.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="grid gap-8 lg:grid-cols-[minmax(0,420px)_minmax(0,1fr)]">
      <div className="space-y-6">
        <div>
          <h1 className="text-[30px] font-medium text-foreground">Storefront</h1>
          <p className="mt-2 text-sm text-foreground-secondary">
            Edit the draft. Publish when you are ready — live customers only see published config.
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="theme">Theme</Label>
          <select
            id="theme"
            className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={cfg.theme_id}
            onChange={(e) => patch({ theme_id: e.target.value })}
          >
            {themes.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>
        </div>

        <div className="grid grid-cols-3 gap-3">
          {(
            [
              ["background_color", "Background"],
              ["text_color", "Text"],
              ["primary_color", "Primary"],
            ] as const
          ).map(([key, label]) => (
            <div key={key} className="space-y-2">
              <Label htmlFor={key}>{label}</Label>
              <Input
                id={key}
                type="color"
                value={cfg[key]}
                onChange={(e) => setCfg((c) => ({ ...c, [key]: e.target.value.toUpperCase() }))}
                onBlur={(e) => patch({ [key]: e.target.value.toUpperCase() })}
              />
            </div>
          ))}
        </div>

        <div className="space-y-2">
          <Label htmlFor="description">Store description</Label>
          <textarea
            id="description"
            className="min-h-24 w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
            value={cfg.description || ""}
            onChange={(e) => setCfg((c) => ({ ...c, description: e.target.value }))}
            onBlur={(e) => patch({ description: e.target.value })}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="contact_email">Contact email</Label>
          <Input
            id="contact_email"
            type="email"
            value={cfg.contact_email || ""}
            onChange={(e) => setCfg((c) => ({ ...c, contact_email: e.target.value }))}
            onBlur={(e) => patch({ contact_email: e.target.value })}
          />
        </div>

        <div className="space-y-2">
          <Label htmlFor="whatsapp">WhatsApp</Label>
          <Input
            id="whatsapp"
            value={cfg.whatsapp || ""}
            onChange={(e) => setCfg((c) => ({ ...c, whatsapp: e.target.value }))}
            onBlur={(e) => patch({ whatsapp: e.target.value })}
          />
        </div>

        {(
          [
            ["logo", "Logo"],
            ["favicon", "Favicon"],
            ["store_icon", "Store icon"],
          ] as const
        ).map(([purpose, label]) => (
          <div key={purpose} className="space-y-2">
            <Label htmlFor={purpose}>{label}</Label>
            <Input
              id={purpose}
              type="file"
              accept={
                purpose === "favicon"
                  ? "image/png,image/x-icon,image/svg+xml,image/webp"
                  : "image/png,image/jpeg,image/webp,image/svg+xml"
              }
              onChange={async (e) => {
                const file = e.target.files?.[0];
                if (!file) return;
                try {
                  await uploadAsset(purpose, file);
                } catch (err) {
                  setError(err instanceof Error ? err.message : "Upload failed.");
                }
              }}
            />
          </div>
        ))}

        <div className="flex flex-wrap gap-3">
          <Button type="button" loading={saving} onClick={() => publish()}>
            Publish storefront
          </Button>
          {store.publication_status === "published" ? (
            <Button type="button" variant="outline" asChild>
              <a href={urls.storefront(store.slug)} target="_blank" rel="noreferrer">
                Open live storefront
              </a>
            </Button>
          ) : (
            <p className="self-center text-sm text-foreground-secondary">
              Live URL opens after you publish — until then{" "}
              <span className="font-medium text-foreground">{urls.storefront(store.slug)}</span>{" "}
              returns 404.
            </p>
          )}
        </div>
        {message && <p className="text-sm text-success">{message}</p>}
        {error && <p className="text-sm text-error">{error}</p>}
      </div>

      <div
        className="rounded-lg border border-border p-6"
        style={{
          background: safeCssColor(cfg.background_color, "#FFFFFF"),
          color: safeCssColor(cfg.text_color, "#15191E"),
          borderColor: safeCssColor(
            cfg.border_color || theme?.border_color,
            "#E1E3E8",
          ),
        }}
      >
        <p className="text-xs uppercase tracking-wide opacity-70">Draft preview</p>
        <div className="mt-4 flex items-center gap-3">
          {cfg.logo_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={cfg.logo_url} alt="" className="h-10 w-auto" />
          ) : (
            <span className="text-2xl font-medium">{store.name}</span>
          )}
        </div>
        <p className="mt-4 max-w-md text-sm opacity-80">
          {cfg.description || "Your store description appears here."}
        </p>
        <button
          type="button"
          className="mt-6 rounded-md px-4 py-2 text-sm font-medium"
          style={{
            background: cfg.primary_color,
            color: cfg.button_foreground || "#15191E",
          }}
        >
          Shop now
        </button>
      </div>
    </div>
  );
}
