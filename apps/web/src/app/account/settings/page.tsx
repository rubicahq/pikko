import { redirect } from "next/navigation";
import Link from "next/link";
import { getAccount, getSession, getStores } from "@/lib/server-api";
import { urls } from "@/lib/urls";
import { Wordmark } from "@/components/brand/wordmark";
import { AccountSettingsForm } from "@/components/account/settings-form";
import { SessionsPanel } from "@/components/account/sessions-panel";

export default async function AccountSettingsPage() {
  const session = await getSession();
  if (!session) redirect(urls.auth());
  const [account, stores] = await Promise.all([getAccount(), getStores()]);
  if (!account) redirect(urls.auth());

  return (
    <div className="min-h-screen bg-background">
      <header className="flex items-center justify-between border-b border-border px-4 py-4 md:px-8">
        <Wordmark href={stores[0] ? urls.storeOverview(stores[0].id) : urls.stores()} />
        <Link href={urls.stores()} className="text-sm text-foreground-secondary hover:text-foreground">
          Manage stores
        </Link>
      </header>
      <main className="mx-auto max-w-reading px-4 py-10 md:px-8">
        <h1 className="text-[30px] font-medium leading-9 text-foreground">Account settings</h1>
        <p className="mt-2 text-sm text-foreground-secondary">
          Passwordless account. No password settings.
        </p>
        <AccountSettingsForm account={account} />
        <SessionsPanel />
      </main>
    </div>
  );
}
