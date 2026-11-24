import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { CampaignForm } from "@/components/campaigns/campaign-form";

export default async function NewCampaignPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  return (
    <div className="mx-auto max-w-reading">
      <h1 className="text-[30px] font-medium text-foreground">Create campaign</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Preorder campaigns create a linked batch. Live campaigns prepare the session shell.
      </p>
      <CampaignForm storeId={storeId} />
    </div>
  );
}
