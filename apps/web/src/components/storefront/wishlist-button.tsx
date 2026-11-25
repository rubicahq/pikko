"use client";

import { useEffect, useState } from "react";
import { Heart } from "lucide-react";
import { ApiError, apiFetch } from "@/lib/api";
import { isValidIndonesianPhone } from "@/lib/phone";
import { getWishlistIdentity, setWishlistIdentity } from "@/lib/wishlist";
import { Button } from "@/components/ui/button";

type AddResult = {
  item: { id: string; product_id: string };
  portal_url?: string;
  customer_created?: boolean;
  already_saved?: boolean;
  customer_id?: string;
  phone?: string;
};

export function WishlistButton({
  storeSlug,
  liveSlug,
  productId,
  productName,
  primaryColor,
  buttonFg,
  enabled,
}: {
  storeSlug: string;
  liveSlug: string;
  productId: string;
  productName: string;
  primaryColor: string;
  buttonFg: string;
  enabled: boolean;
}) {
  const [phone, setPhone] = useState("");
  const [name, setName] = useState("");
  const [showForm, setShowForm] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [portalUrl, setPortalUrl] = useState<string | null>(null);
  const [createdCustomer, setCreatedCustomer] = useState(false);

  useEffect(() => {
    const id = getWishlistIdentity(storeSlug);
    if (id?.phone) setPhone(id.phone);
    if (id?.portalUrl) setPortalUrl(id.portalUrl);
  }, [storeSlug]);

  async function submit(withPhone: string, withName?: string) {
    setBusy(true);
    setError(null);
    try {
      const res = await apiFetch<AddResult>(
        `/api/v1/public/stores/${storeSlug}/live/${liveSlug}/wishlist`,
        {
          method: "POST",
          body: {
            phone: withPhone.trim(),
            name: withName?.trim() || undefined,
            product_id: productId,
          },
        },
      );
      setSaved(true);
      setCreatedCustomer(Boolean(res.customer_created));
      if (res.portal_url) {
        setPortalUrl(res.portal_url);
        setWishlistIdentity({
          storeSlug,
          phone: res.phone || withPhone.trim(),
          portalUrl: res.portal_url,
          customerId: res.customer_id,
        });
      } else {
        setWishlistIdentity({
          storeSlug,
          phone: res.phone || withPhone.trim(),
          customerId: res.customer_id,
        });
      }
      setShowForm(false);
      window.setTimeout(() => setSaved(false), 2200);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Gagal menyimpan wishlist.");
    } finally {
      setBusy(false);
    }
  }

  async function onClick() {
    if (!enabled) return;
    const existing = getWishlistIdentity(storeSlug);
    if (existing?.phone) {
      await submit(existing.phone);
      return;
    }
    setShowForm(true);
  }

  async function onFormSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!isValidIndonesianPhone(phone)) {
      setError("Masukkan nomor WA/HP Indonesia yang valid (contoh 0812…).");
      return;
    }
    await submit(phone, name);
  }

  if (!enabled) {
    return null;
  }

  return (
    <div className="space-y-3">
      <Button
        type="button"
        onClick={onClick}
        disabled={busy}
        className="gap-2"
        style={{ background: primaryColor, color: buttonFg }}
      >
        <Heart className={`h-4 w-4 ${saved ? "fill-current" : ""}`} strokeWidth={1.75} />
        {saved ? "Tersimpan" : "Wishlist"}
      </Button>

      {showForm ? (
        <form
          onSubmit={onFormSubmit}
          className="space-y-3 rounded-md border p-3"
          style={{ borderColor: "currentColor" }}
        >
          <p className="text-sm opacity-80">
            Simpan <span className="font-medium">{productName}</span> ke wishlist. Nomor WhatsApp
            diperlukan sekali untuk membuat akun pelanggan.
          </p>
          <label className="block text-sm">
            Nama (opsional)
            <input
              className="mt-1 h-10 w-full rounded-md border bg-transparent px-3"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="Nama kamu"
              autoComplete="name"
            />
          </label>
          <label className="block text-sm">
            WhatsApp / HP
            <input
              className="mt-1 h-10 w-full rounded-md border bg-transparent px-3"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
              placeholder="0812…"
              inputMode="tel"
              autoComplete="tel"
              required
            />
          </label>
          {error ? <p className="text-sm text-red-600">{error}</p> : null}
          <div className="flex flex-wrap gap-2">
            <Button type="submit" disabled={busy} size="sm">
              {busy ? "Menyimpan…" : "Simpan wishlist"}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={busy}
              onClick={() => {
                setShowForm(false);
                setError(null);
              }}
            >
              Batal
            </Button>
          </div>
        </form>
      ) : null}

      {error && !showForm ? <p className="text-sm text-red-600">{error}</p> : null}

      {portalUrl && (createdCustomer || saved) ? (
        <div className="rounded-md border p-3 text-sm" style={{ borderColor: "currentColor" }}>
          <p className="opacity-80">
            {createdCustomer
              ? "Akun pelanggan dibuat. Simpan link portal kamu:"
              : "Link portal kamu:"}
          </p>
          <a
            href={portalUrl}
            className="mt-1 block break-all font-medium underline-offset-2 hover:underline"
            style={{ color: primaryColor }}
          >
            {portalUrl}
          </a>
        </div>
      ) : null}
    </div>
  );
}
