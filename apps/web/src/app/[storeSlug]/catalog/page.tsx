import { FlipbookViewer } from "@/components/flipbook/flipbook-viewer";

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
      <main className="mx-auto max-w-reading px-4 py-16">
        <h1 className="text-2xl font-medium text-foreground">Catalog unavailable</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          This store catalog is not published or could not be found.
        </p>
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
