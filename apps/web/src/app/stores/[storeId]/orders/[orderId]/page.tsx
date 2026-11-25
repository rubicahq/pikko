import { redirect, notFound } from "next/navigation";
import Link from "next/link";
import { getSession, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { OrderActions } from "@/components/orders/order-actions";
import { OrderRelationsEditor } from "@/components/orders/order-relations-editor";
import { OrderItemsAndRevisions } from "@/components/orders/order-revisions-panel";
import { Badge } from "@/components/ui/badge";

export default async function OrderDetailPage({
  params,
}: {
  params: Promise<{ storeId: string; orderId: string }>;
}) {
  const { storeId, orderId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const data = await serverFetch<{
    order: {
      id: string;
      order_number: string;
      order_status: string;
      payment_status: string;
      fulfillment_status: string;
      fulfillment_method: string;
      customer_name: string;
      customer_email: string;
      customer_phone?: string | null;
      customer_notes?: string | null;
      internal_notes?: string | null;
      total_idr: number;
      subtotal_idr: number;
      shipping_fee_idr: number;
      handling_fee_idr?: number;
      shipping_line1?: string | null;
      shipping_city?: string | null;
      source?: string;
      campaign_id?: string | null;
      preorder_batch_id?: string | null;
      batch_name_snapshot?: string | null;
      items?: {
        id: string;
        product_name: string;
        quantity: number;
        original_quantity: number;
        unit_price_idr: number;
        line_total_idr: number;
        availability_status: string;
        allocated_qty: number;
        packed_qty: number;
        sku?: string | null;
      }[];
      created_at: string;
    };
    payment?: {
      id: string;
      status: string;
      method_type?: string;
      bank_name_snapshot?: string | null;
      account_number_snapshot?: string | null;
      amount_idr: number;
      note?: string | null;
      evidence?: { id: string; submitted_at: string; content_type?: string }[];
    } | null;
  }>(`/api/v1/stores/${storeId}/orders/${orderId}`);

  if (!data?.order) notFound();
  const { order: o, payment: p } = data;

  return (
    <div className="mx-auto max-w-dashboard">
      <Link href={`/stores/${storeId}/orders`} className="text-sm text-foreground-secondary hover:text-foreground">
        ← Orders
      </Link>
      <div className="mt-2 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-[30px] font-medium text-foreground">{o.order_number}</h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            {new Date(o.created_at).toLocaleString("id-ID", { timeZone: "Asia/Jakarta" })} (Asia/Jakarta)
            {o.source ? ` · ${o.source.replaceAll("_", " ")}` : ""}
            {o.batch_name_snapshot ? ` · ${o.batch_name_snapshot}` : ""}
          </p>
        </div>
        <div className="flex flex-wrap gap-1">
          <Badge tone="neutral">{o.order_status.replaceAll("_", " ")}</Badge>
          <Badge tone={o.payment_status === "paid" ? "success" : "warning"}>
            {o.payment_status.replaceAll("_", " ")}
          </Badge>
          <Badge tone="neutral">{o.fulfillment_status.replaceAll("_", " ")}</Badge>
        </div>
      </div>

      <div className="mt-5 grid gap-4 lg:grid-cols-2">
        <section className="rounded-[10px] border border-border p-4">
          <h2 className="text-sm font-medium">Customer</h2>
          <dl className="mt-2 space-y-1.5 text-caption">
            <div>
              <dt className="text-foreground-secondary">Name</dt>
              <dd>{o.customer_name}</dd>
            </div>
            <div>
              <dt className="text-foreground-secondary">Email</dt>
              <dd>{o.customer_email}</dd>
            </div>
            {o.customer_phone ? (
              <div>
                <dt className="text-foreground-secondary">Phone</dt>
                <dd>{o.customer_phone}</dd>
              </div>
            ) : null}
            <div className="pt-2">
              <Link
                href={urls.storeMessages(storeId, {
                  audience: "customer",
                  code: "WA-C02",
                  order_id: o.id,
                })}
                className="text-sm font-medium text-foreground hover:underline"
              >
                WhatsApp templates →
              </Link>
            </div>
            {o.customer_notes ? (
              <div>
                <dt className="text-foreground-secondary">Notes</dt>
                <dd className="whitespace-pre-wrap">{o.customer_notes}</dd>
              </div>
            ) : null}
            {o.internal_notes ? (
              <div>
                <dt className="text-foreground-secondary">Internal</dt>
                <dd className="whitespace-pre-wrap">{o.internal_notes}</dd>
              </div>
            ) : null}
          </dl>
        </section>

        <section className="rounded-[10px] border border-border p-4">
          <h2 className="text-sm font-medium">Fulfillment</h2>
          <p className="mt-2 text-caption capitalize">{o.fulfillment_method.replaceAll("_", " ")}</p>
          {o.shipping_line1 ? (
            <p className="mt-1 text-caption text-foreground-secondary">
              {o.shipping_line1}
              {o.shipping_city ? `, ${o.shipping_city}` : ""}
            </p>
          ) : null}
          <div className="mt-3 space-y-0.5 border-t border-border pt-2.5 text-caption">
            <div className="flex justify-between">
              <span>Subtotal</span>
              <span className="tabular-nums">{formatIDR(o.subtotal_idr)}</span>
            </div>
            {(o.handling_fee_idr ?? 0) > 0 ? (
              <div className="flex justify-between">
                <span>Handling fee</span>
                <span className="tabular-nums">{formatIDR(o.handling_fee_idr || 0)}</span>
              </div>
            ) : null}
            {(o.shipping_fee_idr ?? 0) > 0 ? (
              <div className="flex justify-between">
                <span>Shipping</span>
                <span className="tabular-nums">{formatIDR(o.shipping_fee_idr)}</span>
              </div>
            ) : null}
            <div className="flex justify-between pt-0.5 text-sm font-medium">
              <span>Total</span>
              <span className="tabular-nums">{formatIDR(o.total_idr)}</span>
            </div>
          </div>
        </section>
      </div>

      <div className="mt-4">
        <OrderRelationsEditor
          storeId={storeId}
          orderId={o.id}
          campaignId={o.campaign_id}
          preorderBatchId={o.preorder_batch_id}
          batchName={o.batch_name_snapshot}
        />
      </div>

      <div className="mt-4">
        <OrderItemsAndRevisions
          storeId={storeId}
          orderId={o.id}
          items={o.items || []}
          canWrite
        />
      </div>

      {p ? (
        <section className="mt-4 rounded-[10px] border border-border p-4">
          <h2 className="text-sm font-medium">Payment</h2>
          <p className="mt-1.5 text-caption capitalize">
            {p.method_type === "paid_by_admin"
              ? "Paid by admin"
              : (p.method_type || "bank transfer").replaceAll("_", " ")}
            {" · "}
            {p.status.replaceAll("_", " ")}
          </p>
          {p.method_type === "paid_by_admin" ? (
            <p className="mt-1 text-caption text-foreground-secondary">
              {p.note || "Marked paid by admin"} · {formatIDR(p.amount_idr)}
            </p>
          ) : (
            <>
              <p className="mt-1 text-caption text-foreground-secondary">
                {p.bank_name_snapshot} · {p.account_number_snapshot} · {formatIDR(p.amount_idr)}
              </p>
              {(p.evidence || []).length ? (
                <ul className="mt-3 space-y-1.5 text-caption">
                  {p.evidence!.map((ev) => (
                    <li key={ev.id}>
                      <a
                        className="text-brand-accessible-dark underline"
                        href={`/api/v1/stores/${storeId}/orders/evidence/${ev.id}/file`}
                        target="_blank"
                        rel="noreferrer"
                      >
                        Evidence{" "}
                        {new Date(ev.submitted_at).toLocaleString("id-ID", {
                          timeZone: "Asia/Jakarta",
                        })}
                      </a>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="mt-2 text-caption text-foreground-secondary">No payment proof yet.</p>
              )}
            </>
          )}
        </section>
      ) : null}

      <OrderActions
        storeId={storeId}
        orderId={o.id}
        paymentId={p?.id}
        orderStatus={o.order_status}
        paymentStatus={o.payment_status}
        fulfillmentStatus={o.fulfillment_status}
        fulfillmentMethod={o.fulfillment_method}
        hasEvidence={(p?.evidence || []).length > 0}
      />
    </div>
  );
}
