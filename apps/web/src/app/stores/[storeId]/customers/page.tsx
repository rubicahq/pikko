import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, serverFetch } from "@/lib/server-api";
import { formatIDR } from "@/lib/api";
import { urls } from "@/lib/urls";

type CustomerRow = {
  id: string;
  name: string;
  email: string;
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
    <div className="mx-auto max-w-dashboard">
      <h1 className="text-[30px] font-medium text-foreground">Customers</h1>
      <p className="mt-2 text-sm text-foreground-secondary">
        Store-scoped customer records created from orders.
      </p>

      <form className="mt-6 max-w-md" method="get">
        <input
          name="q"
          defaultValue={q || ""}
          placeholder="Search name, email, or phone"
          className="h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
        />
      </form>

      {customers.length === 0 ? (
        <div className="mt-8 rounded-[10px] border border-border p-8">
          <p className="text-sm text-foreground-secondary">No customers yet.</p>
          <p className="mt-2 text-sm text-foreground-secondary">
            Customers appear automatically when orders are placed.
          </p>
        </div>
      ) : (
        <div className="mt-8 overflow-hidden rounded-[10px] border border-border">
          <table className="w-full text-left text-sm">
            <thead className="bg-surface text-xs font-medium uppercase tracking-wide text-foreground-secondary">
              <tr>
                <th className="px-4 py-3">Customer</th>
                <th className="px-4 py-3">Contact</th>
                <th className="px-4 py-3 text-right">Orders</th>
                <th className="px-4 py-3 text-right">Completed value</th>
              </tr>
            </thead>
            <tbody>
              {customers.map((c) => (
                <tr key={c.id} className="border-t border-border hover:bg-background-subtle">
                  <td className="px-4 py-3">
                    <Link
                      href={urls.storeCustomer(storeId, c.id)}
                      className="font-medium text-foreground hover:underline"
                    >
                      {c.name}
                    </Link>
                    {c.last_order_at ? (
                      <p className="text-xs text-foreground-muted">
                        Last order{" "}
                        {new Date(c.last_order_at).toLocaleString("id-ID", {
                          timeZone: "Asia/Jakarta",
                        })}
                      </p>
                    ) : null}
                  </td>
                  <td className="px-4 py-3">
                    <p>{c.email}</p>
                    {c.phone ? <p className="text-foreground-secondary">{c.phone}</p> : null}
                  </td>
                  <td className="px-4 py-3 text-right tabular-nums">{c.total_orders}</td>
                  <td className="px-4 py-3 text-right tabular-nums">
                    {formatIDR(c.completed_order_value_idr)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
