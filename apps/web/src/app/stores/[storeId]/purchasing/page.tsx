import { Suspense } from "react";
import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { PurchasingWorkspace } from "@/components/purchasing/purchasing-workspace";

export default async function PurchasingPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  return (
    <Suspense fallback={null}>
      <PurchasingWorkspace storeId={storeId} />
    </Suspense>
  );
}
