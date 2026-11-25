"use client";

import { useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";

/**
 * Durable customer portal entry: /{storeSlug}/customer/{secret}-{phone}
 * Authenticates via access_key alone (no email magic-link).
 */
export default function CustomerPortalAccessPage() {
  const params = useParams();
  const router = useRouter();
  const storeSlug = String(params.storeSlug || "");
  const accessKey = decodeURIComponent(String(params.accessKey || ""));
  const [error, setError] = useState<string | null>(null);
  const [status, setStatus] = useState<"working" | "error">("working");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!storeSlug || !accessKey) {
        setStatus("error");
        setError("This portal link is incomplete.");
        return;
      }
      try {
        // Reuse existing session for this store when already signed in.
        try {
          await apiFetch(`/api/v1/portal/${storeSlug}/auth/session`);
          if (!cancelled) {
            router.replace(`/portal/${storeSlug}/dashboard`);
          }
          return;
        } catch {
          /* no session — continue with access key */
        }

        const res = await apiFetch<{ redirect_path?: string }>(
          `/api/v1/portal/${storeSlug}/auth/access`,
          { method: "POST", body: { access_key: accessKey } },
        );
        if (!cancelled) {
          router.replace(res.redirect_path || `/portal/${storeSlug}/dashboard`);
        }
      } catch (err) {
        if (!cancelled) {
          setStatus("error");
          setError(
            err instanceof ApiError
              ? err.message
              : "This portal link is invalid or no longer works.",
          );
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [storeSlug, accessKey, router]);

  return (
    <div className="flex min-h-screen flex-col bg-workspace">
      <main className="mx-auto flex w-full max-w-auth flex-1 flex-col justify-center px-4 py-12">
        {status === "working" ? (
          <>
            <p className="text-sm font-semibold text-foreground">{storeSlug}</p>
            <h1 className="mt-2 text-page font-bold tracking-tight text-foreground">
              Opening your portal…
            </h1>
            <p className="mt-2 text-body text-foreground-secondary">
              Checking your personal link. This only works for the customer it was created for.
            </p>
          </>
        ) : (
          <>
            <p className="text-sm font-semibold text-foreground">{storeSlug}</p>
            <h1 className="mt-2 text-page font-bold tracking-tight text-foreground">
              Link not valid
            </h1>
            <p className="mt-2 text-body text-foreground-secondary">
              {error || "Ask the store for a fresh customer portal link."}
            </p>
            <div className="mt-8">
              <Button
                type="button"
                variant="secondary"
                onClick={() => router.replace(`/${storeSlug}`)}
              >
                Back to store
              </Button>
            </div>
          </>
        )}
      </main>
    </div>
  );
}
