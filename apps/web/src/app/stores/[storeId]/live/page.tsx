import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { LiveSessionsList } from "@/components/live/live-sessions-list";

export default async function LiveSessionsPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  return <LiveSessionsList storeId={storeId} />;
}
