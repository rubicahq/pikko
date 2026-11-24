import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, getStore } from "@/lib/server-api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { urls } from "@/lib/urls";

export default async function StoreOverviewPage({
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
    <div className="mx-auto max-w-dashboard">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-[30px] font-medium leading-9 text-foreground">{store.name}</h1>
          <p className="mt-2 text-sm text-foreground-secondary">
            Storefront:{" "}
            <a
              className="text-brand-dark underline-offset-2 hover:underline"
              href={store.storefront_url || urls.storefront(store.slug)}
              target="_blank"
              rel="noreferrer"
            >
              {store.storefront_url || urls.storefront(store.slug)}
            </a>
          </p>
        </div>
        <Badge tone={store.publication_status === "published" ? "success" : "neutral"}>
          {store.publication_status}
        </Badge>
      </div>

      <section className="mt-8 rounded-lg border border-border p-5">
        <h2 className="text-xl font-medium text-foreground">Setup checklist</h2>
        <p className="mt-2 text-sm text-foreground-secondary">
          Derived from your store state. Catalog, payments, and publication arrive in Phase 2+.
        </p>
        <ul className="mt-5 space-y-3 text-sm">
          <li className="flex items-center gap-2">
            <span className="text-success">✓</span> Create store
          </li>
          <li className="flex items-center gap-2 text-foreground-secondary">
            <span>○</span> Add your first product
          </li>
          <li className="flex items-center gap-2 text-foreground-secondary">
            <span>○</span> Customize storefront
          </li>
          <li className="flex items-center gap-2 text-foreground-secondary">
            <span>○</span> Configure payment methods
          </li>
          <li className="flex items-center gap-2 text-foreground-secondary">
            <span>○</span> Configure fulfillment
          </li>
          <li className="flex items-center gap-2 text-foreground-secondary">
            <span>○</span> Publish and share
          </li>
        </ul>
      </section>

      <section className="mt-6 grid gap-4 md:grid-cols-2">
        <div className="rounded-lg border border-border p-5">
          <h3 className="text-base font-medium">Operational tasks</h3>
          <p className="mt-2 text-sm text-foreground-secondary">
            No orders yet. Tasks appear here from real order and payment activity.
          </p>
        </div>
        <div className="rounded-lg border border-border p-5">
          <h3 className="text-base font-medium">Quick actions</h3>
          <div className="mt-4 flex flex-wrap gap-2">
            <Button asChild variant="outline" size="sm">
              <Link href={urls.stores()}>Manage stores</Link>
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link href={urls.newStore()}>Create another store</Link>
            </Button>
            <Button asChild variant="outline" size="sm">
              <Link href={urls.accountSettings()}>Account settings</Link>
            </Button>
          </div>
        </div>
      </section>
    </div>
  );
}
