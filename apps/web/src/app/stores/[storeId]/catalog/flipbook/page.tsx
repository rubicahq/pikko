import { redirect } from "next/navigation";
import { urls } from "@/lib/urls";

export default async function LegacyCatalogFlipbookRedirect({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  redirect(urls.catalogsFlipbook(storeId));
}
