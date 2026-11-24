import { redirect } from "next/navigation";
import { getSession } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { BatchForm } from "@/components/preorders/batch-form";

export default async function NewPreorderPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  return (
    <div className="mx-auto max-w-reading">
      <h1 className="text-[30px] font-medium text-foreground">Create preorder batch</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Set the open window and ETA. Assign products after creating the batch.
      </p>
      <BatchForm storeId={storeId} />
    </div>
  );
}
