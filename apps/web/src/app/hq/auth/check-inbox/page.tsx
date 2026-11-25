"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { Wordmark } from "@/components/brand/wordmark";

function Body() {
  const sp = useSearchParams();
  const email = sp.get("email") || "your inbox";
  return (
    <div className="w-full max-w-auth">
      <h1 className="text-[28px] font-medium tracking-[-0.02em] text-foreground md:text-4xl">
        Check your inbox
      </h1>
      <p className="mt-3 text-base text-foreground-secondary">
        If <span className="text-foreground">{email}</span> can receive mail, a HQ sign-in link is on the way.
      </p>
    </div>
  );
}

export default function HQCheckInbox() {
  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="px-4 py-5 md:px-8">
        <div className="flex items-center gap-3">
          <Wordmark href="/auth" />
          <span className="rounded border border-border px-2 py-0.5 text-xs font-medium">HQ</span>
        </div>
      </header>
      <main className="flex flex-1 items-start justify-center px-4 pb-16 pt-8 md:items-center">
        <Suspense>
          <Body />
        </Suspense>
      </main>
    </div>
  );
}
