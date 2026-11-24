"use client";

import { Suspense, useEffect, useState } from "react";
import { useParams, useRouter, useSearchParams } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";

function ConfirmInner() {
  const params = useParams();
  const search = useSearchParams();
  const router = useRouter();
  const storeSlug = String(params.storeSlug || "");
  const token = search.get("token") || "";
  const [valid, setValid] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!token) {
        setValid(false);
        return;
      }
      try {
        await apiFetch(
          `/api/v1/portal/${storeSlug}/auth/magic-link/peek?token=${encodeURIComponent(token)}`,
        );
        if (!cancelled) setValid(true);
      } catch {
        if (!cancelled) setValid(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token, storeSlug]);

  async function confirm() {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch<{ redirect_path: string }>(
        `/api/v1/portal/${storeSlug}/auth/magic-link/verify`,
        { body: { token } },
      );
      if (typeof window !== "undefined") {
        sessionStorage.removeItem(`pikko_portal_dev_token_${storeSlug}`);
      }
      router.replace(res.redirect_path || `/portal/${storeSlug}/dashboard`);
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_token") {
        setValid(false);
        setLoading(false);
        return;
      }
      setError(err instanceof ApiError ? err.message : "Unable to sign in.");
      setLoading(false);
    }
  }

  if (valid === null) {
    return <p className="text-sm text-foreground-secondary">Checking your link…</p>;
  }

  if (!valid) {
    return (
      <div>
        <h1 className="text-page font-bold tracking-tight text-foreground">Link expired.</h1>
        <p className="mt-3 text-body text-foreground-secondary">
          Request a new sign-in link from the portal.
        </p>
        <Button className="mt-8" onClick={() => router.replace(`/portal/${storeSlug}`)}>
          Back to portal.
        </Button>
      </div>
    );
  }

  return (
    <div>
      <h1 className="text-page font-bold tracking-tight text-foreground">Continue securely.</h1>
      <p className="mt-3 text-body text-foreground-secondary">
        Confirm to open your orders. This prevents email scanners from using your link.
      </p>
      <Button className="mt-8 w-full" size="large" loading={loading} onClick={confirm}>
        Open my portal.
      </Button>
      {error ? <p className="mt-3 text-sm text-error">{error}</p> : null}
    </div>
  );
}

export default function PortalConfirmPage() {
  return (
    <div className="flex min-h-screen flex-col bg-workspace">
      <main className="mx-auto flex w-full max-w-auth flex-1 flex-col justify-center px-4 py-12">
        <Suspense fallback={<p className="text-sm text-foreground-secondary">Loading…</p>}>
          <ConfirmInner />
        </Suspense>
      </main>
    </div>
  );
}
