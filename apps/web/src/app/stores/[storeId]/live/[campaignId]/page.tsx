import { redirect } from "next/navigation";
import { getSession, getStore, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { LiveConsole } from "@/components/live/live-console";

type LiveSession = Parameters<typeof LiveConsole>[0]["initialSession"];

export default async function LiveConsolePage({
  params,
}: {
  params: Promise<{ storeId: string; campaignId: string }>;
}) {
  const { storeId, campaignId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const [store, liveRes] = await Promise.all([
    getStore(storeId),
    serverFetch<{ session: LiveSession }>(`/api/v1/stores/${storeId}/live/${campaignId}`),
  ]);
  if (!liveRes?.session) redirect(urls.storeLive(storeId));

  return (
    <div className="mx-auto max-w-[1200px]">
      <LiveConsole
        storeId={storeId}
        storeSlug={store?.slug}
        initialSession={liveRes.session}
      />
    </div>
  );
}
