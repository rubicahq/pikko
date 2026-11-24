import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { FinanceWorkspace } from "@/components/finance/finance-workspace";

export default async function FinancePage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  return <FinanceWorkspace storeId={storeId} />;
}
