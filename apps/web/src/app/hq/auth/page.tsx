"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { Wordmark } from "@/components/brand/wordmark";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { ApiError, hqFetch } from "@/lib/hq-api";

export default function HQAuthPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await hqFetch<{ email: string; dev_token?: string }>("/auth/magic-link/request", {
        body: { email },
      });
      if (res.dev_token && typeof window !== "undefined") {
        sessionStorage.setItem("pikko_hq_dev_magic_token", res.dev_token);
      }
      router.push(`/auth/check-inbox?email=${encodeURIComponent(res.email)}`);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Something went wrong.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="flex min-h-screen flex-col bg-background">
      <header className="px-4 py-5 md:px-8">
        <div className="flex items-center gap-3">
          <Wordmark href="/auth" />
          <span className="rounded border border-border px-2 py-0.5 text-xs font-medium text-foreground-secondary">
            HQ
          </span>
        </div>
      </header>
      <main className="flex flex-1 items-start justify-center px-4 pb-16 pt-8 md:items-center md:pt-0">
        <div className="w-full max-w-auth">
          <h1 className="text-[28px] font-medium leading-[1.12] tracking-[-0.02em] text-foreground md:text-4xl">
            Pikko HQ
          </h1>
          <p className="mt-3 text-base text-foreground-secondary">
            Internal platform admin for Tim Pikko. Magic link only.
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
                placeholder="you@pikko.site"
              />
              {error ? <p className="text-sm text-error">{error}</p> : null}
            </div>
            <Button type="submit" className="w-full" size="large" loading={loading}>
              Continue with email
            </Button>
          </form>
        </div>
      </main>
    </div>
  );
}
