import Link from "next/link";
import { redirect } from "next/navigation";
import { getSession, getStores } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Wordmark } from "@/components/brand/wordmark";
import { ArchiveStoreButton } from "@/components/stores/archive-button";

function modelLabel(m: string) {
  switch (m) {
    case "ready_stock":
      return "Ready stock";
    case "preorder":
      return "Preorder";
    case "both":
      return "Both";
    default:
      return m;
  }
}

export default async function ManageStoresPage() {
  const session = await getSession();
  if (!session) redirect(urls.auth());

  const stores = await getStores(true);
  const active = stores.filter((s) => !s.archived_at);
  const archived = stores.filter((s) => s.archived_at);

  return (
    <div className="min-h-screen bg-background">
      <header className="flex items-center justify-between border-b border-border px-4 py-4 md:px-8">
        <Wordmark href={active[0] ? urls.storeOverview(active[0].id) : urls.onboarding()} />
        <div className="flex items-center gap-3">
          <Button asChild variant="ghost">
            <Link href={urls.accountSettings()}>Account</Link>
          </Button>
          <Button asChild>
            <Link href={urls.newStore()}>Create new store</Link>
          </Button>
        </div>
      </header>

      <main className="mx-auto max-w-dashboard px-4 py-10 md:px-8">
        <h1 className="text-[30px] font-medium leading-9 text-foreground">Manage stores</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Open a dashboard, preview a storefront URL, or archive a store you no longer use.
        </p>

        {active.length === 0 ? (
          <div className="mt-10 rounded-lg border border-border p-8">
            <p className="text-base font-medium text-foreground">You don&apos;t have an active store yet.</p>
            <p className="mt-2 text-sm text-foreground-secondary">
              Create a store to publish a storefront and start receiving orders.
            </p>
            <Button asChild className="mt-6">
              <Link href={urls.onboarding()}>Create your store.</Link>
            </Button>
          </div>
        ) : (
          <ul className="mt-8 grid gap-4 md:grid-cols-2">
            {active.map((store) => (
              <li key={store.id} className="rounded-lg border border-border p-5">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <h2 className="text-base font-medium text-foreground">{store.name}</h2>
                    <p className="mt-1 text-sm text-foreground-secondary">
                      {store.storefront_url || urls.storefront(store.slug)}
                    </p>
                  </div>
                  <Badge tone={store.publication_status === "published" ? "success" : "neutral"}>
                    {store.publication_status}
                  </Badge>
                </div>
                <dl className="mt-4 grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <dt className="text-foreground-muted">Selling model</dt>
                    <dd className="text-foreground">{modelLabel(store.selling_model)}</dd>
                  </div>
                  <div>
                    <dt className="text-foreground-muted">Created</dt>
                    <dd className="text-foreground">
                      {new Date(store.created_at).toLocaleDateString("id-ID")}
                    </dd>
                  </div>
                </dl>
                <div className="mt-5 flex flex-wrap gap-2">
                  <Button asChild size="sm">
                    <Link href={urls.storeOverview(store.id)}>Open dashboard</Link>
                  </Button>
                  <Button asChild size="sm" variant="outline">
                    <a href={store.storefront_url || urls.storefront(store.slug)} target="_blank" rel="noreferrer">
                      Preview storefront
                    </a>
                  </Button>
                  <ArchiveStoreButton storeId={store.id} storeName={store.name} />
                </div>
              </li>
            ))}
          </ul>
        )}

        {archived.length > 0 ? (
          <section className="mt-12">
            <h2 className="text-xl font-medium text-foreground">Archived</h2>
            <ul className="mt-4 space-y-2">
              {archived.map((store) => (
                <li
                  key={store.id}
                  className="flex items-center justify-between rounded-md border border-border px-4 py-3 text-sm"
                >
                  <span className="text-foreground-secondary">{store.name}</span>
                  <Badge>archived</Badge>
                </li>
              ))}
            </ul>
          </section>
        ) : null}
      </main>
    </div>
  );
}
