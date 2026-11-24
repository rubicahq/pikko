import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { PackingWorkspace } from "@/components/packing/packing-workspace";

export default async function PackingPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  return <PackingWorkspace storeId={storeId} />;
}
