"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { apiFetch } from "@/lib/api";

/**
 * Portal landing without a personal access key.
 * Email magic-link login is retired — customers must open their
 * /{slug}/customer/{secret}-{phone} link.
 */
export default function PortalLoginPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const router = useRouter();
  const [checking, setChecking] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await apiFetch(`/api/v1/portal/${storeSlug}/auth/session`);
        if (!cancelled) router.replace(`/portal/${storeSlug}/dashboard`);
      } catch {
        if (!cancelled) setChecking(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [storeSlug, router]);

  if (checking) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-workspace px-4">
        <p className="text-sm text-foreground-secondary">Checking session…</p>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen flex-col bg-workspace">
      <main className="mx-auto flex w-full max-w-auth flex-1 flex-col justify-center px-4 py-12">
        <p className="text-sm font-semibold text-foreground">{storeSlug}</p>
        <h1 className="mt-2 text-page font-bold tracking-tight text-foreground">
          Personal link required
        </h1>
        <p className="mt-2 text-body text-foreground-secondary">
          Open the customer portal link from your store (it looks like{" "}
          <span className="font-mono text-sm">
            /{storeSlug}/customer/••••-62…
          </span>
          ). Email sign-in is not used for the portal.
        </p>
        <p className="mt-6 text-label text-foreground-muted">
          Looking for a single order? Use the tracking link from your confirmation message.
        </p>
      </main>
    </div>
  );
}
