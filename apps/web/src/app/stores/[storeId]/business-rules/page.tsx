import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { RulesClient } from "@/components/business-rules/rules-client";

type RuleRow = {
  id: string;
  rule_type: string;
  name: string;
  scope_type: string;
  scope_selling_model?: string | null;
  scope_batch_id?: string | null;
  is_enabled: boolean;
  current_version: number;
  config: Record<string, unknown>;
};

type BatchRow = { id: string; name: string };

export default async function BusinessRulesPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const [rulesData, batchesData] = await Promise.all([
    serverFetch<{ rules: RuleRow[] }>(`/api/v1/stores/${storeId}/business-rules`),
    serverFetch<{ batches: BatchRow[] }>(`/api/v1/stores/${storeId}/preorders`),
  ]);

  return (
    <div className="mx-auto max-w-dashboard">
      <h1 className="text-[30px] font-medium text-foreground">Business rules</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Handling fees, minimums, and order approval. More specific scopes override store defaults.
      </p>
      <RulesClient
        storeId={storeId}
        rules={(rulesData?.rules || []) as Parameters<typeof RulesClient>[0]["rules"]}
        batches={(batchesData?.batches || []).map((b) => ({ id: b.id, name: b.name }))}
      />
    </div>
  );
}
