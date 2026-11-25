"use client";

import { useEffect, useState } from "react";
import { EmptyOrUnavailable } from "@/components/hq/kpi";
import { hqFetch, statusTone } from "@/lib/hq-api";
import { Button } from "@/components/ui/button";

type Rule = {
  id: string;
  name: string;
  enabled: boolean;
  rule_type: string;
  threshold: number;
  window_minutes: number;
  cooldown_minutes: number;
  notify_emails: string[];
  last_fired_at?: string | null;
};

type Event = {
  id: string;
  rule_type: string;
  severity: string;
  title: string;
  message: string;
  status: string;
  fired_at: string;
};

export default function HQAlertsPage() {
  const [rules, setRules] = useState<Rule[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function load() {
    const [r, e] = await Promise.all([
      hqFetch<{ items: Rule[] }>("/system-health/alerts/rules"),
      hqFetch<{ items: Event[] }>("/system-health/alerts/events?limit=40"),
    ]);
    setRules(r.items);
    setEvents(e.items);
  }

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await load();
        if (!cancelled) setError(null);
      } catch {
        if (!cancelled) setError("Unable to load alerts.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  async function toggle(rule: Rule) {
    await hqFetch(`/system-health/alerts/rules/${rule.id}`, {
      method: "PATCH",
      body: { enabled: !rule.enabled },
    });
    await load();
  }

  async function evaluate() {
    setLoading(true);
    try {
      await hqFetch("/system-health/alerts/evaluate", { method: "POST" });
      await load();
      setError(null);
    } catch {
      setError("Evaluate failed.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-medium tracking-tight">Alerts</h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            Founder alerts via existing email infra — deduped with cooldown.
          </p>
        </div>
        <Button type="button" loading={loading} onClick={evaluate}>
          Evaluate now
        </Button>
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}

      <section>
        <h2 className="mb-3 text-sm font-medium">Rules</h2>
        {rules.length === 0 ? (
          <EmptyOrUnavailable title="No alert rules" />
        ) : (
          <div className="overflow-x-auto rounded border border-border bg-white">
            <table className="min-w-full text-left text-sm">
              <thead className="border-b border-border text-xs uppercase text-foreground-muted">
                <tr>
                  <th className="px-3 py-2">Name</th>
                  <th className="px-3 py-2">Type</th>
                  <th className="px-3 py-2">Threshold</th>
                  <th className="px-3 py-2">Window</th>
                  <th className="px-3 py-2">Cooldown</th>
                  <th className="px-3 py-2">Enabled</th>
                </tr>
              </thead>
              <tbody>
                {rules.map((r) => (
                  <tr key={r.id} className="border-b border-border-subtle">
                    <td className="px-3 py-2 font-medium">{r.name}</td>
                    <td className="px-3 py-2">{r.rule_type}</td>
                    <td className="px-3 py-2">{r.threshold}</td>
                    <td className="px-3 py-2">{r.window_minutes}m</td>
                    <td className="px-3 py-2">{r.cooldown_minutes}m</td>
                    <td className="px-3 py-2">
                      <button
                        type="button"
                        className="underline"
                        onClick={() => toggle(r)}
                      >
                        {r.enabled ? "On" : "Off"}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </section>

      <section>
        <h2 className="mb-3 text-sm font-medium">Recent events</h2>
        {events.length === 0 ? (
          <EmptyOrUnavailable title="No alert events yet" />
        ) : (
          <ul className="divide-y divide-border rounded border border-border bg-white">
            {events.map((e) => (
              <li key={e.id} className="px-3 py-2.5 text-sm">
                <div className="flex flex-wrap items-center gap-2">
                  <span className={`capitalize ${statusTone(e.severity)}`}>{e.severity}</span>
                  <span className="font-medium">{e.title}</span>
                  <span className="text-foreground-muted">{e.status}</span>
                </div>
                <p className="mt-0.5 text-foreground-secondary">{e.message}</p>
                <p className="text-xs text-foreground-muted">
                  {new Date(e.fired_at).toLocaleString("en-GB", { timeZone: "Asia/Jakarta" })} Asia/Jakarta
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
