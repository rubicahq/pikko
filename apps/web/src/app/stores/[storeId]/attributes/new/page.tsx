import { redirect } from "next/navigation";
import { getSession, getStore } from "@/lib/server-api";
import { AttributeForm } from "@/components/catalog/attribute-form";
import { urls } from "@/lib/urls";

export default async function NewAttributePage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());
  return <AttributeForm storeId={storeId} />;
}
