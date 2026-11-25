"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { Wordmark } from "@/components/brand/wordmark";
import { HQDocumentTitle } from "@/components/hq/document-title";
import { hqFetch } from "@/lib/hq-api";

const TABS = [
  { href: "/system-health", label: "Overview", priority: "P0" },
  { href: "/system-health/errors", label: "Error Logs", priority: "P0" },
  { href: "/system-health/jobs", label: "Background Jobs", priority: "P0" },
  { href: "/system-health/services", label: "Services", priority: "P0" },
  { href: "/system-health/usage", label: "Usage & Costs", priority: "P1" },
  { href: "/system-health/alerts", label: "Alerts", priority: "P1" },
] as const;

export function HQShell({
  children,
  email,
}: {
  children: React.ReactNode;
  email?: string;
}) {
  const pathname = usePathname();
  const router = useRouter();

  async function logout() {
    try {
      await hqFetch("/auth/logout", { method: "POST" });
    } catch {
      /* ignore */
    }
    router.replace("/auth");
    router.refresh();
  }

  return (
    <div className="min-h-screen bg-background-subtle text-foreground">
      <HQDocumentTitle />
      <header className="border-b border-border bg-white">
        <div className="mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-4 md:px-6">
          <div className="flex items-center gap-3">
            <Wordmark href="/system-health" />
            <span className="rounded border border-border px-2 py-0.5 text-xs font-medium text-foreground-secondary">
              HQ
            </span>
          </div>
          <div className="flex items-center gap-3 text-sm text-foreground-secondary">
            {email ? <span className="hidden sm:inline">{email}</span> : null}
            <button
              type="button"
              onClick={logout}
              className="text-foreground hover:underline"
            >
              Sign out
            </button>
          </div>
        </div>
        <nav className="mx-auto flex max-w-6xl gap-1 overflow-x-auto px-4 pb-0 md:px-6">
          {TABS.map((tab) => {
            const active =
              pathname === tab.href ||
              (tab.href !== "/system-health" && pathname.startsWith(tab.href));
            return (
              <Link
                key={tab.href}
                href={tab.href}
                className={`whitespace-nowrap border-b-2 px-3 py-2.5 text-sm font-medium transition-colors ${
                  active
                    ? "border-ink text-foreground"
                    : "border-transparent text-foreground-secondary hover:text-foreground"
                }`}
              >
                {tab.label}
              </Link>
            );
          })}
        </nav>
      </header>
      <main className="mx-auto max-w-6xl px-4 py-6 md:px-6 md:py-8">{children}</main>
    </div>
  );
}
