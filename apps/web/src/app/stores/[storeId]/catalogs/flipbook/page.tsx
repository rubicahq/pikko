import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, getStore } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { FlipbookEditor } from "@/components/flipbook/flipbook-editor";

export default async function CatalogsFlipbookEditorPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());

  return (
    <div className="mx-auto max-w-5xl space-y-6">
      <div>
        <p className="text-sm text-foreground-secondary">
          <Link href={urls.storeCatalogs(storeId)} className="hover:text-foreground">
            Catalogs
          </Link>
          <span className="mx-1.5">/</span>
          Flipbook
        </p>
        <h1 className="mt-2 text-[30px] font-medium tracking-tight text-foreground">
          Catalog flipbook
        </h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Reorder pages, place product hotspots, and publish. Public viewers only see published
          pages and published products.
        </p>
      </div>
      <FlipbookEditor storeId={storeId} storeSlug={store.slug} />
    </div>
  );
}
