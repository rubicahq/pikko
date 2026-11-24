import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
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

type Campaign = {
  id: string;
  name: string;
  slug: string;
  campaign_type: string;
  opens_at: string;
  closes_at: string;
  operational_status: string;
  publication_status: string;
  product_count?: number;
  preorder_batch_id?: string | null;
};

export default async function CampaignsPage({
  params,
}: {
  params: Promise<{ storeId: string }>;
}) {
  const { storeId } = await params;
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const data = await serverFetch<{ campaigns: Campaign[] }>(
    `/api/v1/stores/${storeId}/campaigns`,
  );
  const campaigns = data?.campaigns || [];

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Campaigns"
        description="Unified selling windows for preorder batches and live shopping sessions."
        actions={
          <Button asChild>
            <Link href={urls.storeCampaignNew(storeId)}>Create campaign</Link>
          </Button>
        }
      />

      {campaigns.length === 0 ? (
        <EmptyState
          title="No campaigns yet"
          description="Create a preorder campaign (links to a batch) or a live session shell for Agent 12."
          action={
            <Button asChild size="sm">
              <Link href={urls.storeCampaignNew(storeId)}>Create campaign</Link>
            </Button>
          }
        />
      ) : (
        <Table stickyHeader>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead sticky>Campaign</TableHead>
              <TableHead sticky>Type</TableHead>
              <TableHead sticky>Window</TableHead>
              <TableHead sticky>Status</TableHead>
              <TableHead sticky className="text-right">
                Products
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {campaigns.map((c) => (
              <TableRow key={c.id}>
                <TableCell>
                  <Link
                    href={urls.storeCampaign(storeId, c.id)}
                    className="font-medium text-foreground hover:underline"
                  >
                    {c.name}
                  </Link>
                  <p className="text-xs text-foreground-muted">/{c.slug}</p>
                </TableCell>
                <TableCell className="capitalize text-foreground-secondary">
                  {c.campaign_type}
                </TableCell>
                <TableCell className="tabular-nums text-foreground-secondary">
                  {formatJakarta(c.opens_at)} → {formatJakarta(c.closes_at)}
                </TableCell>
                <TableCell>
                  <StatusBadge status={c.operational_status} />
                  {c.publication_status === "draft" ? (
                    <span className="ml-2 text-xs text-foreground-muted">draft</span>
                  ) : null}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {c.product_count ?? 0}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}

      <p className="text-sm text-foreground-secondary">
        Legacy batch tools remain under{" "}
        <Link href={urls.storePreorders(storeId)} className="underline">
          Preorders
        </Link>
        . New work should prefer Campaigns.
      </p>
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
