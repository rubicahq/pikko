"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function OrderActions({
  storeId,
  orderId,
  paymentId,
  orderStatus,
  paymentStatus,
  fulfillmentStatus,
  fulfillmentMethod,
  hasEvidence,
}: {
  storeId: string;
  orderId: string;
  paymentId?: string;
  orderStatus: string;
  paymentStatus: string;
  fulfillmentStatus: string;
  fulfillmentMethod: string;
  hasEvidence: boolean;
}) {
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);

  async function run(fn: () => Promise<void>) {
    setBusy(true);
    setError(null);
    try {
      await fn();
      router.refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Action failed.");
    } finally {
      setBusy(false);
    }
  }

  const nextFulfillment = (() => {
    if (fulfillmentMethod === "pickup") {
      if (fulfillmentStatus === "unfulfilled" || fulfillmentStatus === "awaiting_arrival") {
        return "ready_for_pickup";
      }
      if (fulfillmentStatus === "ready_for_pickup") return "collected";
      return null;
    }
    const chain: Record<string, string> = {
      unfulfilled: "ready_to_ship",
      purchasing: "awaiting_arrival",
      awaiting_arrival: "ready_to_ship",
      ready_to_ship: "shipped",
      shipped: "delivered",
    };
    return chain[fulfillmentStatus] || null;
  })();

  return (
    <section className="mt-6 space-y-4 rounded-[10px] border border-border p-5">
      <h2 className="text-base font-medium">Actions</h2>
      {error ? <p className="text-sm text-error">{error}</p> : null}
      <div className="flex flex-wrap gap-2">
        {orderStatus === "pending_confirmation" ? (
          <>
            <Button
              type="button"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await apiFetch(`/api/v1/stores/${storeId}/orders/${orderId}`, {
                    method: "PATCH",
                    body: { order_status: "confirmed" },
                  });
                })
              }
            >
              Confirm order
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await apiFetch(`/api/v1/stores/${storeId}/orders/${orderId}`, {
                    method: "PATCH",
                    body: { order_status: "cancelled", cancel_reason: "Rejected by merchant" },
                  });
                })
              }
            >
              Reject order
            </Button>
          </>
        ) : null}

        {paymentId && hasEvidence && paymentStatus === "awaiting_verification" ? (
          <>
            <Button
              type="button"
              disabled={busy}
              onClick={() =>
                run(async () => {
                  await apiFetch(
                    `/api/v1/stores/${storeId}/orders/${orderId}/payments/${paymentId}/verify`,
                    { method: "POST", body: {} },
                  );
                })
              }
            >
              Verify payment
            </Button>
            <div className="flex flex-wrap items-end gap-2">
              <div className="space-y-1">
                <Label htmlFor="reject-reason">Reject reason</Label>
                <Input
                  id="reject-reason"
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="Unclear transfer proof"
                />
              </div>
              <Button
                type="button"
                variant="outline"
                disabled={busy || !reason.trim()}
                onClick={() =>
                  run(async () => {
                    await apiFetch(
                      `/api/v1/stores/${storeId}/orders/${orderId}/payments/${paymentId}/reject`,
                      { method: "POST", body: { reason } },
                    );
                  })
                }
              >
                Reject evidence
              </Button>
            </div>
          </>
        ) : null}

        {nextFulfillment && paymentStatus === "paid" ? (
          <Button
            type="button"
            variant="secondary"
            disabled={busy}
            onClick={() =>
              run(async () => {
                await apiFetch(`/api/v1/stores/${storeId}/orders/${orderId}`, {
                  method: "PATCH",
                  body: { fulfillment_status: nextFulfillment },
                });
              })
            }
          >
            Mark {nextFulfillment.replaceAll("_", " ")}
          </Button>
        ) : null}

        {orderStatus === "confirmed" &&
        paymentStatus === "paid" &&
        (fulfillmentStatus === "delivered" || fulfillmentStatus === "collected") ? (
          <Button
            type="button"
            disabled={busy}
            onClick={() =>
              run(async () => {
                await apiFetch(`/api/v1/stores/${storeId}/orders/${orderId}`, {
                  method: "PATCH",
                  body: { order_status: "completed" },
                });
              })
            }
          >
            Complete order
          </Button>
        ) : null}

        {orderStatus !== "cancelled" && orderStatus !== "completed" && paymentStatus !== "paid" ? (
          <Button
            type="button"
            variant="destructive"
            disabled={busy}
            onClick={() =>
              run(async () => {
                await apiFetch(`/api/v1/stores/${storeId}/orders/${orderId}`, {
                  method: "PATCH",
                  body: { order_status: "cancelled", cancel_reason: "Cancelled by merchant" },
                });
              })
            }
          >
            Cancel order
          </Button>
        ) : null}
      </div>
    </section>
  );
}
