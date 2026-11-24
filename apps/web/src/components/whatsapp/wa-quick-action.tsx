"use client";

import { useState } from "react";
import { MessageCircle } from "lucide-react";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";

type Props = {
  storeId: string;
  code: string;
  label?: string;
  orderId?: string;
  vendorId?: string;
  customerId?: string;
  phone?: string;
  overrides?: Record<string, string>;
  size?: "sm" | "default";
  variant?: "primary" | "secondary" | "ghost";
};

/** One-click open WhatsApp with a registry template (logs send). */
export function WAQuickAction({
  storeId,
  code,
  label = "WhatsApp",
  orderId,
  vendorId,
  customerId,
  phone,
  overrides,
  size = "sm",
  variant = "secondary",
}: Props) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function open() {
    setBusy(true);
    setError(null);
    try {
      const data = await apiFetch<{ deep_link: string }>(
        `/api/v1/stores/${storeId}/whatsapp/templates/${code}/open`,
        {
          method: "POST",
          body: {
            action: "open",
            order_id: orderId,
            vendor_id: vendorId,
            customer_id: customerId,
            phone,
            overrides,
          },
        },
      );
      window.open(data.deep_link, "_blank", "noopener,noreferrer");
    } catch (e) {
      setError(e instanceof ApiError ? e.message : "WhatsApp failed.");
    } finally {
      setBusy(false);
    }
  }

  return (
    <span className="inline-flex flex-col items-start gap-1">
      <Button type="button" size={size} variant={variant} disabled={busy} onClick={() => void open()}>
        <MessageCircle className="size-4" />
        {label}
      </Button>
      {error ? <span className="text-caption text-error">{error}</span> : null}
    </span>
  );
}
