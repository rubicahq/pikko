import { redirect } from "next/navigation";
import { urls } from "@/lib/urls";

export default async function LegacyCatalogProductRedirect({
  params,
}: {
  params: Promise<{ storeId: string; productId: string }>;
}) {
  const { storeId, productId } = await params;
  if (productId === "import" || productId === "flipbook" || productId === "new") {
    redirect(urls.storeProducts(storeId));
  }
  redirect(urls.storeProduct(storeId, productId));
}
