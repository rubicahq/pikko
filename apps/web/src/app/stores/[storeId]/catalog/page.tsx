import { redirect } from "next/navigation";
import { urls } from "@/lib/urls";

/** Legacy /catalog → Products master. */
export default async function LegacyCatalogRedirect({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  redirect(urls.storeProducts(storeId));
}
