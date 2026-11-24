"use client";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardDescription, CardTitle } from "@/components/ui/card";
import { StatusBadge } from "@/components/ui/status-badge";
import { EmptyState } from "@/components/ui/empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { Skeleton, TableSkeleton } from "@/components/ui/skeleton";
import { PageHeader } from "@/components/ui/page-header";
import { DataTable } from "@/components/ui/data-table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Toaster, useToast } from "@/components/ui/toast";
import { liveNavGroups } from "@/components/dashboard/nav-config";

type DemoRow = { id: string; order: string; customer: string; total: number; status: string };

const DEMO_ROWS: DemoRow[] = Array.from({ length: 28 }).map((_, i) => ({
  id: String(i + 1),
  order: `PK-${1000 + i}`,
  customer: ["Ayu", "Budi", "Citra", "Dewi", "Eko"][i % 5],
  total: 150_000 + i * 12_500,
  status: ["paid", "awaiting_verification", "confirmed", "draft", "failed", "cancelled"][i % 6],
}));

function ToastDemoButton() {
  const { publish } = useToast();
  return (
    <Button
      type="button"
      variant="outline"
      onClick={() =>
        publish({
          title: "Saved",
          description: "Toast uses the shared provider.",
          tone: "success",
        })
      }
    >
      Show toast
    </Button>
  );
}

function PreviewBody() {
  const nav = liveNavGroups();

  return (
    <div className="mx-auto max-w-dashboard space-y-12 px-4 py-10 md:px-8">
      <PageHeader
        title="Component preview"
        description="Dev-only catalog of Pikko UI primitives for dense merchant workflows. Not served in production."
      />

      <section className="space-y-4">
        <h2 className="text-section">Buttons</h2>
        <div className="flex flex-wrap gap-3">
          <Button>Primary (Ink)</Button>
          <Button variant="brand">Brand</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="outline">Outline</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="destructive">Destructive</Button>
          <Button variant="link">Text link</Button>
          <Button loading>Loading</Button>
        </div>
      </section>

      <section className="max-w-md space-y-3">
        <h2 className="text-section">Inputs</h2>
        <div className="space-y-2">
          <Label htmlFor="demo">Email address</Label>
          <Input id="demo" placeholder="you@example.com" />
        </div>
        <div className="space-y-2">
          <Label>With error</Label>
          <Input error defaultValue="bad" />
          <p className="text-sm text-error">Enter a valid email address.</p>
        </div>
        <div className="space-y-2">
          <Label>Select</Label>
          <Select defaultValue="idr">
            <SelectTrigger aria-label="Currency">
              <SelectValue placeholder="Currency" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="idr">IDR</SelectItem>
              <SelectItem value="usd">USD</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-section">Badges & status</h2>
        <div className="flex flex-wrap gap-2">
          <Badge>Draft</Badge>
          <Badge tone="success">Paid</Badge>
          <Badge tone="warning">Awaiting payment</Badge>
          <Badge tone="error">Failed</Badge>
          <Badge tone="info">Open</Badge>
          <Badge tone="brand">Current</Badge>
        </div>
        <div className="flex flex-wrap gap-2">
          {[
            "paid",
            "awaiting_verification",
            "confirmed",
            "draft",
            "open",
            "closed",
            "cancelled",
            "failed",
          ].map((s) => (
            <StatusBadge key={s} status={s} />
          ))}
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-section">Tabs / dialog / toast</h2>
        <Tabs defaultValue="all">
          <TabsList>
            <TabsTrigger value="all">All</TabsTrigger>
            <TabsTrigger value="open">Open</TabsTrigger>
            <TabsTrigger value="closed">Closed</TabsTrigger>
          </TabsList>
          <TabsContent value="all">
            <p className="text-body text-foreground-secondary">Filter chrome for list pages.</p>
          </TabsContent>
          <TabsContent value="open">
            <p className="text-body text-foreground-secondary">Open filter selected.</p>
          </TabsContent>
          <TabsContent value="closed">
            <p className="text-body text-foreground-secondary">Closed filter selected.</p>
          </TabsContent>
        </Tabs>
        <div className="flex flex-wrap gap-3">
          <Dialog>
            <DialogTrigger asChild>
              <Button variant="outline">Open dialog</Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Confirm action</DialogTitle>
                <DialogDescription>
                  Dialogs use restrained borders and radius — no heavy glass overlays.
                </DialogDescription>
              </DialogHeader>
            </DialogContent>
          </Dialog>
          <ToastDemoButton />
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-section">Empty / error / loading</h2>
        <div className="grid gap-4 md:grid-cols-2">
          <EmptyState
            title="No results"
            description="Try adjusting filters or create your first record."
            action={<Button size="sm">Create</Button>}
          />
          <ErrorState description="Sample recoverable error for list fetches." onRetry={() => undefined} />
        </div>
        <div className="space-y-3">
          <Skeleton className="h-4 w-48" />
          <TableSkeleton rows={4} cols={4} />
        </div>
      </section>

      <section className="space-y-4">
        <h2 className="text-section">Data table</h2>
        <DataTable
          columns={[
            {
              id: "order",
              header: "Order",
              accessor: (r) => r.order,
              sortable: true,
            },
            {
              id: "customer",
              header: "Customer",
              accessor: (r) => r.customer,
              sortable: true,
            },
            {
              id: "status",
              header: "Status",
              accessor: (r) => r.status,
              cell: (r) => <StatusBadge status={r.status} />,
              sortable: true,
            },
            {
              id: "total",
              header: "Total",
              accessor: (r) => r.total,
              align: "right",
              sortable: true,
              cell: (r) => `Rp ${r.total.toLocaleString("id-ID")}`,
            },
          ]}
          data={DEMO_ROWS}
          rowKey={(r) => r.id}
          filterPlaceholder="Filter orders…"
          stickyHeader
        />
      </section>

      <section className="space-y-4">
        <h2 className="text-section">Nav IA (live groups)</h2>
        <Card>
          <CardTitle>PRD §6 grouping</CardTitle>
          <CardDescription>
            Unimplemented modules are omitted from the merchant shell.
          </CardDescription>
          <Separator className="my-4" />
          <div className="grid gap-4 sm:grid-cols-3">
            {nav.map((group) => (
              <div key={group.id}>
                <p className="text-label uppercase tracking-wide text-foreground-muted">{group.label}</p>
                <ul className="mt-2 space-y-1 text-sm text-foreground-secondary">
                  {group.items.map((item) => (
                    <li key={item.href}>{item.label}</li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </Card>
      </section>
    </div>
  );
}

export function ComponentPreviewClient() {
  return (
    <Toaster>
      <PreviewBody />
    </Toaster>
  );
}
