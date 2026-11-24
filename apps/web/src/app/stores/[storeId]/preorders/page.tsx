import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";

type Batch = {
  id: string;
  name: string;
  slug: string;
  opens_at: string;
  closes_at: string;
  operational_status: string;
  publication_status: string;
  product_count?: number;
  order_count?: number;
};

export default async function PreordersPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const data = await serverFetch<{ batches: Batch[] }>(`/api/v1/stores/${storeId}/preorders`);
  const batches = data?.batches || [];

  return (
    <div className="mx-auto max-w-dashboard">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-[30px] font-medium text-foreground">Preorders</h1>
          <p className="mt-2 text-sm text-foreground-secondary">
            Open and close batches, assign products, and manage purchasing.
          </p>
        </div>
        <Button asChild>
          <Link href={`/stores/${storeId}/preorders/new`}>Create batch</Link>
        </Button>
      </div>

      {batches.length === 0 ? (
        <div className="mt-10 rounded-[10px] border border-border p-8">
          <h2 className="text-base font-medium text-foreground">No preorder batches yet.</h2>
          <p className="mt-2 text-sm text-foreground-secondary">
            Create a batch with open and close dates, then assign preorder-eligible products.
          </p>
          <Button asChild className="mt-5">
            <Link href={`/stores/${storeId}/preorders/new`}>Create batch</Link>
          </Button>
        </div>
      ) : (
        <div className="mt-8 overflow-hidden rounded-[10px] border border-border">
          <table className="w-full text-left text-sm">
            <thead className="border-b border-border bg-background-subtle text-xs font-medium uppercase tracking-wide text-foreground-muted">
              <tr>
                <th className="px-4 py-3">Batch</th>
                <th className="px-4 py-3">Window</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-4 py-3 text-right">Products</th>
                <th className="px-4 py-3 text-right">Orders</th>
              </tr>
            </thead>
            <tbody>
              {batches.map((b) => (
                <tr key={b.id} className="border-b border-border last:border-0 hover:bg-surface">
                  <td className="px-4 py-3">
                    <Link
                      href={`/stores/${storeId}/preorders/${b.id}`}
                      className="font-medium text-foreground hover:underline"
                    >
                      {b.name}
                    </Link>
                    <p className="text-xs text-foreground-muted">/{b.slug}</p>
                  </td>
                  <td className="px-4 py-3 text-foreground-secondary tabular-nums">
                    {formatJakarta(b.opens_at)} → {formatJakarta(b.closes_at)}
                  </td>
                  <td className="px-4 py-3">
                    <StatusPill status={b.operational_status} />
                    {b.publication_status === "draft" ? (
                      <span className="ml-2 text-xs text-foreground-muted">draft</span>
                    ) : null}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums">{b.product_count ?? 0}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{b.order_count ?? 0}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function formatJakarta(iso: string) {
  try {
    return new Intl.DateTimeFormat("id-ID", {
      timeZone: "Asia/Jakarta",
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(iso));
  } catch {
    return iso;
  }
}

function StatusPill({ status }: { status: string }) {
  return (
    <span className="inline-flex rounded-sm border border-border bg-surface px-2 py-0.5 text-xs capitalize text-foreground">
      {status.replaceAll("_", " ")}
    </span>
  );
}
