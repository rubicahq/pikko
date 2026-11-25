import { redirect } from "next/navigation";
import { urls } from "@/lib/urls";

export default async function LegacyCatalogNewRedirect({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  redirect(urls.storeProductNew(storeId));
}
