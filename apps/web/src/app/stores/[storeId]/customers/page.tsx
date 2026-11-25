import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";
import { CreateCustomerForm } from "@/components/customers/create-customer-form";
import { PageHeader } from "@/components/ui/page-header";
import { EmptyState } from "@/components/ui/empty-state";
import { FilterBar } from "@/components/ui/filter-bar";
import { Input } from "@/components/ui/input";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

type CustomerRow = {
  id: string;
  name: string;
  email?: string | null;
  phone?: string | null;
  total_orders: number;
  completed_order_value_idr: number;
  last_order_at?: string | null;
};

export default async function CustomersPage({
  params,
  searchParams,
}: {
  params: Promise<{ storeId: string }>;
  searchParams: Promise<{ q?: string }>;
}) {
  const { storeId } = await params;
  const { q } = await searchParams;
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const qs = q ? `?q=${encodeURIComponent(q)}` : "";
  const data = await serverFetch<{ customers: CustomerRow[]; total: number }>(
    `/api/v1/stores/${storeId}/customers${qs}`,
  );
  const customers = data?.customers || [];

  return (
    <div className="mx-auto max-w-dashboard space-y-8">
      <PageHeader
        title="Customers"
        description="Store-scoped customers from orders or added with name and WhatsApp only."
      />

      <CreateCustomerForm storeId={storeId} />

      <FilterBar>
        <form method="get" className="w-full max-w-md">
          <Input
            name="q"
            defaultValue={q || ""}
            placeholder="Search name, email, or phone"
            aria-label="Search customers"
          />
        </form>
      </FilterBar>

      {customers.length === 0 ? (
        <EmptyState
          title="No customers yet"
          description="Add a customer above, or they appear automatically when orders are placed."
        />
      ) : (
        <Table stickyHeader>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead sticky>Customer</TableHead>
              <TableHead sticky>Contact</TableHead>
              <TableHead sticky className="text-right">
                Orders
              </TableHead>
              <TableHead sticky className="text-right">
                Completed value
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {customers.map((c) => (
              <TableRow key={c.id}>
                <TableCell>
                  <Link
                    href={urls.storeCustomer(storeId, c.id)}
                    className="font-medium text-foreground hover:underline"
                  >
                    {c.name}
                  </Link>
                  {c.last_order_at ? (
                    <p className="text-caption text-foreground-muted">
                      Last order{" "}
                      {new Date(c.last_order_at).toLocaleString("id-ID", {
                        timeZone: "Asia/Jakarta",
                      })}
                    </p>
                  ) : null}
                </TableCell>
                <TableCell>
                  {c.phone ? <p>{c.phone}</p> : null}
                  {c.email ? (
                    <p className={c.phone ? "text-foreground-secondary" : undefined}>{c.email}</p>
                  ) : !c.phone ? (
                    <p className="text-foreground-muted">—</p>
                  ) : null}
                </TableCell>
                <TableCell className="text-right tabular-nums">{c.total_orders}</TableCell>
                <TableCell className="text-right tabular-nums">
                  {formatIDR(c.completed_order_value_idr)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
