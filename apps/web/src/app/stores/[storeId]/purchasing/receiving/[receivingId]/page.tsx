import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { ReceivingDetail } from "@/components/purchasing/receiving-detail";

export default async function ReceivingDetailPage({
  params,
}: {
  params: Promise<{ storeId: string; receivingId: string }>;
}) {
  const { storeId, receivingId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  return <ReceivingDetail storeId={storeId} receivingId={receivingId} />;
}
