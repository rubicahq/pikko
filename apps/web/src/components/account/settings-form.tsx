"use client";

import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import type { Account } from "@/lib/api";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

export function AccountSettingsForm({ account }: { account: Account }) {
  const router = useRouter();
  const [displayName, setDisplayName] = useState(account.display_name || "");
  const [language, setLanguage] = useState(account.language || "id-ID");
  const [timezone, setTimezone] = useState(account.timezone || "Asia/Jakarta");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError(null);
    setMessage(null);
    try {
      await apiFetch("/api/v1/account", {
        method: "PATCH",
        body: {
          display_name: displayName,
          language,
          timezone,
        },
      });
      setMessage("Saved.");
      router.refresh();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to save.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <form onSubmit={onSubmit} className="mt-8 space-y-5 rounded-lg border border-border p-5">
      <div className="space-y-2">
        <Label>Verified email</Label>
        <Input value={account.email} disabled readOnly />
      </div>
      <div className="space-y-2">
        <Label htmlFor="display_name">Display name</Label>
        <Input
          id="display_name"
          value={displayName}
          onChange={(e) => setDisplayName(e.target.value)}
          placeholder="Your name"
        />
      </div>
      <div className="grid gap-5 md:grid-cols-2">
        <div className="space-y-2">
          <Label htmlFor="language">Language</Label>
          <select
            id="language"
            className="flex h-11 w-full rounded-md border border-border bg-background px-3 text-sm"
            value={language}
            onChange={(e) => setLanguage(e.target.value)}
          >
            <option value="id-ID">Bahasa Indonesia</option>
            <option value="en-US">English</option>
          </select>
        </div>
        <div className="space-y-2">
          <Label htmlFor="timezone">Timezone</Label>
          <Input
            id="timezone"
            value={timezone}
            onChange={(e) => setTimezone(e.target.value)}
          />
        </div>
      </div>
      {error ? <p className="text-sm text-error">{error}</p> : null}
      {message ? <p className="text-sm text-success">{message}</p> : null}
      <Button type="submit" loading={loading}>
        Save changes
      </Button>
    </form>
  );
}
