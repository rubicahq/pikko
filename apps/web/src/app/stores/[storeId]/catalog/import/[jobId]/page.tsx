import { redirect } from "next/navigation";
import { urls } from "@/lib/urls";

/** Legacy import job → Catalogs review (PDF flow); CSV jobs also work there. */
export default async function LegacyCatalogImportJobRedirect({
  params,
}: {
  params: Promise<{ storeId: string; jobId: string }>;
}) {
  const { storeId, jobId } = await params;
  redirect(urls.catalogsImportJob(storeId, jobId));
}
