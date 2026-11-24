import { redirect } from "next/navigation";
import { getSession, getStore } from "@/lib/server-api";
import { TeamPanel } from "@/components/team/team-panel";
import { urls } from "@/lib/urls";

export default async function TeamPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());

  return <TeamPanel storeId={storeId} />;
}
