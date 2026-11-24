import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { MessagesWorkspace } from "@/components/whatsapp/messages-workspace";

export default async function MessagesPage({
  params,
  searchParams,
}: {
  params: Promise<{ storeId: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const { storeId } = await params;
  const sp = await searchParams;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const one = (v: string | string[] | undefined) =>
    Array.isArray(v) ? v[0] : v;

  const audienceRaw = one(sp.audience);
  const audience =
    audienceRaw === "vendor" || audienceRaw === "customer" || audienceRaw === "all"
      ? audienceRaw
      : "all";

  return (
    <MessagesWorkspace
      storeId={storeId}
      initialAudience={audience}
      initialCode={one(sp.code)}
      initialOrderId={one(sp.order_id)}
      initialVendorId={one(sp.vendor_id)}
      initialCustomerId={one(sp.customer_id)}
    />
  );
}
