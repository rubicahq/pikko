import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, getStore, serverFetch } from "@/lib/server-api";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { StatusBadge } from "@/components/ui/status-badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type CatalogDoc = {
  id: string;
  title: string;
  status: string;
  page_count: number;
  source_import_job_id?: string;
  created_at: string;
};

export default async function CatalogsPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const store = await getStore(storeId);
  if (!store) redirect(urls.stores());

  const data = await serverFetch<{ catalogs: CatalogDoc[] }>(
    `/api/v1/stores/${storeId}/catalogs`,
  );
  const catalogs = data?.catalogs ?? [];

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Catalogs"
        description={`PDF flipbook catalogs for ${store.name}.`}
        actions={
          catalogs.length > 0 ? (
            <>
              <Button asChild variant="outline">
                <Link href={urls.catalogsFlipbook(storeId)}>Flipbook</Link>
              </Button>
              <Button asChild>
                <Link href={urls.storeCatalogsNew(storeId)}>Add Catalog</Link>
              </Button>
            </>
          ) : null
        }
      />

      {catalogs.length === 0 ? (
        <EmptyState
          className="items-center py-16 text-center"
          title="No catalogs yet"
          description="Upload a PDF to create a display flipbook catalog."
          action={
            <Button asChild size="large">
              <Link href={urls.storeCatalogsNew(storeId)}>Add Catalog</Link>
            </Button>
          }
        />
      ) : (
        <Table stickyHeader>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead sticky>Catalog</TableHead>
              <TableHead sticky>Pages</TableHead>
              <TableHead sticky>Status</TableHead>
              <TableHead sticky>Created</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {catalogs.map((c) => (
              <TableRow key={c.id}>
                <TableCell>
                  <Link
                    href={urls.storeCatalogDoc(storeId, c.id)}
                    className="font-medium text-foreground hover:underline"
                  >
                    {c.title}
                  </Link>
                </TableCell>
                <TableCell className="tabular-nums text-foreground-secondary">
                  {c.page_count}
                </TableCell>
                <TableCell>
                  <StatusBadge status={c.status} />
                </TableCell>
                <TableCell className="text-foreground-secondary">
                  {new Date(c.created_at).toLocaleDateString("id-ID", {
                    day: "numeric",
                    month: "short",
                    year: "numeric",
                  })}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
