import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";

const LATER = new Set(["orders", "preorders", "customers", "business-rules"]);

const PHASE2_REDIRECT: Record<string, (storeId: string) => string> = {
  storefront: urls.storeStorefront,
  catalog: urls.storeCatalog,
  settings: urls.storeSettings,
};

export default async function StoreSectionPlaceholder({
  params,
}: {
  params: Promise<{ storeId: string; section: string }>;
}) {
  const { storeId, section } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  if (section === "overview") redirect(urls.storeOverview(storeId));
  if (PHASE2_REDIRECT[section]) redirect(PHASE2_REDIRECT[section](storeId));
  if (!LATER.has(section)) redirect(urls.storeOverview(storeId));

  return (
    <div className="mx-auto max-w-reading">
      <h1 className="text-[30px] font-medium capitalize text-foreground">
        {section.replace("-", " ")}
      </h1>
      <p className="mt-3 text-sm text-foreground-secondary">
        This module is scheduled for a later phase and is intentionally not built yet. Catalog and
        storefront are available from the sidebar.
      </p>
    </div>
  );
}
