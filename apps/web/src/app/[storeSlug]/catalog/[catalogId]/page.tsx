import Link from "next/link";
import { FlipbookViewer } from "@/components/flipbook/flipbook-viewer";
import { urls } from "@/lib/urls";

const API = process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

const CATALOG_ID_UUID =
  /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

export default async function PublicCatalogDocPage({
  params,
}: {
  params: Promise<{ storeSlug: string; catalogId: string }>;
}) {
  const { storeSlug, catalogId } = await params;
  if (!CATALOG_ID_UUID.test(catalogId)) {
    return (
      <main className="mx-auto flex min-h-[60vh] max-w-reading flex-col justify-center px-4 py-16">
        <h1 className="text-2xl font-semibold text-foreground">Catalog unavailable</h1>
        <Link
          href={urls.storefrontCatalogs(storeSlug)}
          className="mt-6 text-sm font-medium text-brand-dark underline-offset-2 hover:underline"
        >
          Browse catalogs
        </Link>
      </main>
    );
  }

  const res = await fetch(
    `${API}/api/v1/public/stores/${storeSlug}/catalogs/${catalogId}`,
    { cache: "no-store" },
  );
  if (!res.ok) {
    return (
      <main className="mx-auto flex min-h-[60vh] max-w-reading flex-col justify-center px-4 py-16">
        <p className="text-sm text-foreground-secondary">
          <Link href={urls.storefrontPath(storeSlug)} className="hover:text-foreground">
            Store
          </Link>
          <span className="mx-1.5">/</span>
          <Link href={urls.storefrontCatalogs(storeSlug)} className="hover:text-foreground">
            Catalogs
          </Link>
          <span className="mx-1.5">/</span>
          Catalog
        </p>
        <h1 className="mt-3 text-2xl font-semibold text-foreground">Catalog unavailable</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          This catalog is not published or could not be found.
        </p>
        <Link
          href={urls.storefrontCatalogs(storeSlug)}
          className="mt-6 text-sm font-medium text-brand-dark underline-offset-2 hover:underline"
        >
          Browse catalogs
        </Link>
      </main>
    );
  }
  const data = await res.json();
  return (
    <FlipbookViewer
      storeSlug={storeSlug}
      storeName={data.store?.name || storeSlug}
      pages={data.pages || []}
      storeWhatsapp={data.store?.whatsapp || null}
    />
  );
}
