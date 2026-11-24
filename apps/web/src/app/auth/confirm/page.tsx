"use client";

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";

function ConfirmInner() {
  const params = useSearchParams();
  const router = useRouter();
  const token = params.get("token") || "";
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
        // GET peek — must NOT consume the token (PRD §9)
        await apiFetch(`/api/v1/auth/magic-link/peek?token=${encodeURIComponent(token)}`);
        if (!cancelled) setValid(true);
      } catch {
        if (!cancelled) setValid(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [token]);

  async function confirm() {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch<{ redirect_path: string }>("/api/v1/auth/magic-link/verify", {
        body: { token },
      });
      if (typeof window !== "undefined") {
        sessionStorage.removeItem("pikko_dev_magic_token");
      }
      router.replace(res.redirect_path || "/onboarding");
    } catch (err) {
      if (err instanceof ApiError && err.code === "invalid_token") {
        router.replace("/auth/invalid");
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
    router.replace("/auth/invalid");
    return null;
  }

  return (
    <div>
      <h1 className="text-[28px] font-medium leading-[1.12] tracking-[-0.02em] text-foreground md:text-4xl">
        Continue to Pikko.
      </h1>
      <p className="mt-3 text-base text-foreground-secondary">
        Confirm to sign in securely. This prevents email scanners from using your link.
      </p>
      <Button className="mt-8 w-full" size="large" loading={loading} onClick={confirm}>
        Sign in securely.
      </Button>
      {error ? <p className="mt-3 text-sm text-error">{error}</p> : null}
    </div>
  );
}

export default function ConfirmPage() {
  return (
    <Suspense fallback={<div className="text-sm text-foreground-secondary">Loading…</div>}>
      <ConfirmInner />
    </Suspense>
  );
}
