import { Suspense } from "react";
import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { ShipmentsWorkspace } from "@/components/packing/shipments-workspace";

export default async function ShipmentsPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  return (
    <Suspense fallback={<p className="text-sm text-foreground-secondary">Loading shipments…</p>}>
      <ShipmentsWorkspace storeId={storeId} />
    </Suspense>
  );
}
