"use client";

import { Suspense, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";

function CheckInboxInner() {
  const params = useSearchParams();
  const router = useRouter();
  const email = params.get("email") || "";
  const [seconds, setSeconds] = useState(60);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [devToken, setDevToken] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window !== "undefined") {
      setDevToken(sessionStorage.getItem("pikko_dev_magic_token"));
    }
  }, []);

  useEffect(() => {
    if (seconds <= 0) return;
    const t = setTimeout(() => setSeconds((s) => s - 1), 1000);
    return () => clearTimeout(t);
  }, [seconds]);

  const canResend = seconds <= 0;

  async function resend() {
    if (!email || !canResend) return;
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch<{ email: string; cooldown_seconds: number; dev_token?: string }>(
        "/api/v1/auth/magic-link/request",
        { body: { email } },
      );
      if (res.dev_token) {
        sessionStorage.setItem("pikko_dev_magic_token", res.dev_token);
        setDevToken(res.dev_token);
      }
      setSeconds(res.cooldown_seconds || 60);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to resend.");
    } finally {
      setLoading(false);
    }
  }

  const confirmHref = useMemo(
    () => (devToken ? urls.authConfirm(devToken) : null),
    [devToken],
  );

  return (
    <div>
      <h1 className="text-[28px] font-medium leading-[1.12] tracking-[-0.02em] text-foreground md:text-4xl">
        Check your inbox.
      </h1>
      <p className="mt-3 text-base text-foreground-secondary">
        We&apos;ve sent a secure sign-in link to your email address.
      </p>
      {email ? (
        <p className="mt-4 rounded-md border border-border bg-background-subtle px-3 py-2 text-sm text-foreground">
          {email}
        </p>
      ) : null}

      <div className="mt-8 space-y-3">
        <Button
          variant="outline"
          className="w-full"
          disabled={!canResend}
          loading={loading}
          onClick={resend}
        >
          {canResend ? "Resend link" : `Resend in ${seconds}s`}
        </Button>
        <Button asChild variant="ghost" className="w-full">
          <Link href="/auth">Use a different email</Link>
        </Button>
      </div>
      {error ? <p className="mt-3 text-sm text-error">{error}</p> : null}

      {confirmHref ? (
        <div className="mt-8 rounded-md border border-brand-border bg-brand-soft p-4">
          <p className="text-sm text-foreground-secondary">
            Dev mail mode: open the confirmation page without leaving the browser.
          </p>
          <Button
            className="mt-3 w-full"
            variant="secondary"
            onClick={() => router.push(confirmHref)}
          >
            Open sign-in link
          </Button>
        </div>
      ) : null}
    </div>
  );
}

export default function CheckInboxPage() {
  return (
    <Suspense fallback={<div className="text-sm text-foreground-secondary">Loading…</div>}>
      <CheckInboxInner />
    </Suspense>
  );
}
