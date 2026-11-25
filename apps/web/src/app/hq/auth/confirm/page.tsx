"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { Wordmark } from "@/components/brand/wordmark";
import { ApiError, hqFetch } from "@/lib/hq-api";

function ConfirmBody() {
  const sp = useSearchParams();
  const router = useRouter();
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const token = sp.get("token") || "";
    if (!token) {
      setError("Missing token.");
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        await hqFetch("/auth/magic-link/verify", { body: { token } });
        if (!cancelled) router.replace("/system-health");
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof ApiError ? err.message : "Unable to sign in.");
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [sp, router]);

  return (
    <div className="w-full max-w-auth">
      <h1 className="text-[28px] font-medium tracking-[-0.02em] text-foreground md:text-4xl">
        {error ? "Sign-in failed" : "Signing you in…"}
      </h1>
      {error ? <p className="mt-3 text-base text-error">{error}</p> : null}
    </div>
  );
}

export default function HQConfirmPage() {
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
          <ConfirmBody />
        </Suspense>
      </main>
    </div>
  );
}
