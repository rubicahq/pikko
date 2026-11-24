import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";

const PHASE2 = new Set([
  "storefront",
  "catalog",
  "orders",
  "preorders",
  "customers",
  "business-rules",
  "settings",
]);

export default async function StoreSectionPlaceholder({
  params,
}: {
  params: Promise<{ storeId: string; section: string }>;
}) {
  const { storeId, section } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  if (section === "overview") redirect(urls.storeOverview(storeId));
  if (!PHASE2.has(section)) redirect(urls.storeOverview(storeId));

  return (
    <div className="mx-auto max-w-reading">
      <h1 className="text-[30px] font-medium capitalize text-foreground">
        {section.replace("-", " ")}
      </h1>
      <p className="mt-3 text-sm text-foreground-secondary">
        This module is scheduled for Phase 2 onward and is intentionally not built in this
        foundation release. Your store, membership, and authorization are already active.
      </p>
    </div>
  );
}
