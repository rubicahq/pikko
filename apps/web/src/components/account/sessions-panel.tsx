"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import type { Session } from "@/lib/api";
import { apiFetch, ApiError } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { urls } from "@/lib/urls";

export function SessionsPanel() {
  const router = useRouter();
  const [sessions, setSessions] = useState<Session[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function load() {
    try {
      const res = await apiFetch<{ sessions: Session[] }>("/api/v1/auth/sessions");
      setSessions(res.sessions || []);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to load sessions.");
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function logout() {
    await apiFetch("/api/v1/auth/logout", { method: "POST", body: {} });
    router.replace(urls.auth());
  }

  async function logoutAll() {
    setLoading(true);
    try {
      await apiFetch("/api/v1/auth/logout-all", { method: "POST", body: {} });
      router.replace(urls.auth());
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "Unable to sign out.");
      setLoading(false);
    }
  }

  async function revoke(id: string) {
    await apiFetch(`/api/v1/auth/sessions/${id}`, { method: "DELETE" });
    await load();
  }

  return (
    <section className="mt-10 rounded-lg border border-border p-5">
      <h2 className="text-xl font-medium text-foreground">Sessions</h2>
      <p className="mt-2 text-sm text-foreground-secondary">
        Sign out this device or revoke other active sessions.
      </p>
      {error ? <p className="mt-3 text-sm text-error">{error}</p> : null}
      <ul className="mt-5 space-y-3">
        {sessions.map((s) => (
          <li
            key={s.id}
            className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-border px-3 py-3 text-sm"
          >
            <div>
              <p className="text-foreground">{s.user_agent || "Unknown device"}</p>
              <p className="text-foreground-muted">
                Last seen {new Date(s.last_seen_at).toLocaleString("id-ID")}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {s.current ? <Badge tone="brand">Current</Badge> : null}
              {!s.current ? (
                <Button size="sm" variant="outline" onClick={() => revoke(s.id)}>
                  Revoke
                </Button>
              ) : null}
            </div>
          </li>
        ))}
      </ul>
      <div className="mt-5 flex flex-wrap gap-2">
        <Button variant="outline" onClick={logout}>
          Sign out
        </Button>
        <Button variant="destructive" loading={loading} onClick={logoutAll}>
          Sign out all devices
        </Button>
      </div>
    </section>
  );
}
