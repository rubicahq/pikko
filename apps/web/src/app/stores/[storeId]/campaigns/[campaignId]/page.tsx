import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { CampaignDetailClient } from "@/components/campaigns/campaign-detail-client";

export default async function CampaignDetailPage({
  params,
}: {
  params: Promise<{ storeId: string; campaignId: string }>;
}) {
  const { storeId, campaignId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const [campRes, productsRes] = await Promise.all([
    serverFetch<{ campaign: Parameters<typeof CampaignDetailClient>[0]["campaign"] }>(
      `/api/v1/stores/${storeId}/campaigns/${campaignId}`,
    ),
    serverFetch<{ products: Parameters<typeof CampaignDetailClient>[0]["catalog"] }>(
      `/api/v1/stores/${storeId}/products?limit=100`,
    ),
  ]);

  if (!campRes?.campaign) redirect(urls.storeCampaigns(storeId));

  return (
    <div className="mx-auto max-w-dashboard">
      <CampaignDetailClient
        storeId={storeId}
        campaign={campRes.campaign}
        catalog={productsRes?.products || []}
      />
    </div>
  );
}
