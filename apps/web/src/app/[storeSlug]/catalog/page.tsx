import Link from "next/link";
import { FlipbookViewer } from "@/components/flipbook/flipbook-viewer";
import { urls } from "@/lib/urls";

const API = process.env.API_BASE_URL || process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080";

export default async function PublicCatalogPage({
  params,
}: {
  params: Promise<{ storeSlug: string }>;
}) {
  const { storeSlug } = await params;
  const res = await fetch(`${API}/api/v1/public/stores/${storeSlug}/catalog`, {
    cache: "no-store",
  });
  if (!res.ok) {
    return (
      <main className="mx-auto flex min-h-[60vh] max-w-reading flex-col justify-center px-4 py-16">
        <p className="text-sm text-foreground-secondary">
          <Link href={urls.storefrontPath(storeSlug)} className="hover:text-foreground">
            Store
          </Link>
          <span className="mx-1.5">/</span>
          Catalog
        </p>
        <h1 className="mt-3 text-2xl font-semibold text-foreground">Catalog unavailable</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          This store catalog is not published or could not be found.
        </p>
        <Link
          href={urls.storefrontPath(storeSlug)}
          className="mt-6 text-sm font-medium text-brand-dark underline-offset-2 hover:underline"
        >
          Back to store
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
    />
  );
}
