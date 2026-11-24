"use client";

import Link from "next/link";
import { useParams, useSearchParams } from "next/navigation";
import { Suspense } from "react";

function InboxInner() {
  const params = useParams();
  const search = useSearchParams();
  const storeSlug = String(params.storeSlug || "");
  const email = search.get("email") || "your inbox";
  const base = `/portal/${storeSlug}`;

  return (
    <div className="flex min-h-screen flex-col bg-workspace">
      <main className="mx-auto flex w-full max-w-auth flex-1 flex-col justify-center px-4 py-12">
        <h1 className="text-page font-bold tracking-tight text-foreground">Check your inbox.</h1>
        <p className="mt-3 text-body text-foreground-secondary">
          If <span className="font-medium text-foreground">{email}</span> has orders at this store, a
          sign-in link is on the way. The link expires in a few minutes and can only be used once.
        </p>
        <p className="mt-6 text-sm text-foreground-muted">
          Didn&apos;t get it?{" "}
          <Link href={base} className="font-medium text-foreground underline-offset-4 hover:underline">
            Try again
          </Link>
          .
        </p>
      </main>
    </div>
  );
}

export default function PortalCheckInboxPage() {
  return (
    <Suspense fallback={<p className="p-8 text-sm text-foreground-secondary">Loading…</p>}>
      <InboxInner />
    </Suspense>
  );
}
