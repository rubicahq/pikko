"use client";

import { FormEvent, useEffect, useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function PortalLoginPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
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

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await apiFetch<{ email: string; dev_token?: string }>(
        `/api/v1/portal/${storeSlug}/auth/magic-link/request`,
        { body: { email } },
      );
      if (res.dev_token && typeof window !== "undefined") {
        sessionStorage.setItem(`pikko_portal_dev_token_${storeSlug}`, res.dev_token);
      }
      router.push(`/portal/${storeSlug}/check-inbox?email=${encodeURIComponent(res.email)}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong. Try again.");
    } finally {
      setLoading(false);
    }
  }

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
          Your orders, securely.
        </h1>
        <p className="mt-2 text-body text-foreground-secondary">
          Enter the email you used at checkout. We&apos;ll send a one-time sign-in link — no password.
        </p>
        <form onSubmit={onSubmit} className="mt-8 space-y-5">
          <div className="space-y-2">
            <Label htmlFor="email">Email address</Label>
            <Input
              id="email"
              type="email"
              autoComplete="email"
              required
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              error={!!error}
              placeholder="you@example.com"
            />
            {error ? <p className="text-sm text-error">{error}</p> : null}
          </div>
          <Button type="submit" className="w-full" size="large" loading={loading}>
            Email me a link.
          </Button>
        </form>
        <p className="mt-6 text-label text-foreground-muted">
          Looking for a single order? Use the tracking link from your confirmation email.
        </p>
      </main>
    </div>
  );
}
