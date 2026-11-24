"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { apiFetch, ApiError } from "@/lib/api";
import { urls } from "@/lib/urls";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export default function AuthEnterPage() {
  const router = useRouter();
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setError(null);
    setLoading(true);
    try {
      const res = await apiFetch<{ email: string; dev_token?: string }>(
        "/api/v1/auth/magic-link/request",
        { body: { email } },
      );
      if (res.dev_token && typeof window !== "undefined") {
        sessionStorage.setItem("pikko_dev_magic_token", res.dev_token);
      }
      router.push(urls.authCheckInbox(res.email));
    } catch (err) {
      if (err instanceof ApiError) {
        setError(err.message);
      } else {
        setError("Something went wrong. Try again.");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div>
      <h1 className="text-[28px] font-medium leading-[1.12] tracking-[-0.02em] text-foreground md:text-4xl">
        Let&apos;s get started.
      </h1>
      <p className="mt-3 text-base text-foreground-secondary">
        Enter your email to create an account or sign in.
      </p>
      <form onSubmit={onSubmit} className="mt-8 space-y-5">
        <div className="space-y-2">
          <Label htmlFor="email">Email address</Label>
          <Input
            id="email"
            name="email"
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
          Continue with email.
        </Button>
      </form>
      <p className="mt-4 text-sm text-foreground-muted">No password needed.</p>
    </div>
  );
}
