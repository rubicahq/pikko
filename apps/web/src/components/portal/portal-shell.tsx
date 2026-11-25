"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

const NAV = [
  { href: "dashboard", label: "Orders" },
  { href: "addresses", label: "Addresses" },
  { href: "refunds", label: "Refunds" },
  { href: "credit", label: "Credit" },
] as const;

export function PortalShell({
  storeSlug,
  storeName,
  customerEmail,
  primaryColor,
  children,
}: {
  storeSlug: string;
  storeName: string;
  customerEmail?: string | null;
  primaryColor?: string;
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const router = useRouter();
  const base = `/portal/${storeSlug}`;

  async function logout() {
    try {
      await apiFetch(`/api/v1/portal/${storeSlug}/auth/logout`, { method: "POST" });
    } catch {
      /* ignore */
    }
    router.replace(base);
  }

  return (
    <div className="min-h-screen bg-workspace">
      <header className="border-b border-border bg-background">
        <div className="mx-auto flex max-w-[720px] items-center justify-between gap-4 px-4 py-4 md:px-6">
          <div className="min-w-0">
            <p className="truncate text-sm font-semibold text-foreground">{storeName}</p>
            <p className="text-label text-foreground-muted">Customer portal</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {customerEmail ? (
              <span className="hidden text-label text-foreground-secondary sm:inline">{customerEmail}</span>
            ) : null}
            <Button variant="secondary" size="sm" onClick={logout}>
              Sign out
            </Button>
          </div>
        </div>
        <nav className="mx-auto flex max-w-[720px] gap-1 overflow-x-auto px-4 pb-3 md:px-6">
          {NAV.map((item) => {
            const href = `${base}/${item.href}`;
            const active = pathname === href || pathname.startsWith(href + "/");
            return (
              <Link
                key={item.href}
                href={href}
                className={cn(
                  "rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  active
                    ? "bg-surface-active text-foreground"
                    : "text-foreground-secondary hover:bg-surface-hover hover:text-foreground",
                )}
                style={
                  active && primaryColor
                    ? { boxShadow: `inset 0 -2px 0 ${primaryColor}` }
                    : undefined
                }
              >
                {item.label}
              </Link>
            );
          })}
        </nav>
      </header>
      <main className="mx-auto max-w-[720px] px-4 py-8 md:px-6">{children}</main>
    </div>
  );
}
