"use client";

import { useEffect, useState } from "react";
import { EmptyOrUnavailable, KPI, RangePicker } from "@/components/hq/kpi";
import { JobRow, fmtMs, fmtNum, hqFetch, statusTone } from "@/lib/hq-api";

type Stats = {
  queued: number;
  running: number;
  success: number;
  failed: number;
  stuck: number;
  retrying: number;
  dead: number;
  avg_duration_ms: number | null;
};

export default function HQJobsPage() {
  const [range, setRange] = useState("24h");
  const [stats, setStats] = useState<Stats | null>(null);
  const [items, setItems] = useState<JobRow[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);

  async function load() {
    const [st, list] = await Promise.all([
      hqFetch<Stats>(`/system-health/jobs/stats?range=${range}`),
      hqFetch<{ items: JobRow[] }>(`/system-health/jobs?range=${range}&limit=50`),
    ]);
    setStats(st);
    setItems(list.items);
  }

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        await load();
        if (!cancelled) setError(null);
      } catch {
        if (!cancelled) setError("Unable to load jobs.");
      }
    })();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [range]);

  async function retry(id: string) {
    if (!confirm("Retry this job? Only safe for idempotent job_queue work.")) return;
    setBusy(id);
    try {
      await hqFetch(`/system-health/jobs/${id}/retry`, { method: "POST" });
      await load();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Retry failed");
    } finally {
      setBusy(null);
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-medium tracking-tight">Background Jobs</h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            job_queue, email_jobs, catalog imports — measured counts only.
          </p>
        </div>
        <RangePicker value={range} onChange={setRange} />
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}

      {stats ? (
        <div className="grid gap-x-10 md:grid-cols-4">
          <KPI label="Queued" value={fmtNum(stats.queued)} />
          <KPI label="Running" value={fmtNum(stats.running)} />
          <KPI label="Success (range)" value={fmtNum(stats.success)} />
          <KPI label="Failed (range)" value={fmtNum(stats.failed)} />
          <KPI label="Stuck" value={fmtNum(stats.stuck)} />
          <KPI label="Retrying" value={fmtNum(stats.retrying)} />
          <KPI label="Dead (range)" value={fmtNum(stats.dead)} />
          <KPI
            label="Avg duration"
            value={fmtMs(stats.avg_duration_ms)}
            unavailable={stats.avg_duration_ms === null}
          />
        </div>
      ) : null}

      {items.length === 0 ? (
        <EmptyOrUnavailable title="No jobs in range" />
      ) : (
        <div className="overflow-x-auto rounded border border-border bg-white">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-border text-xs uppercase text-foreground-muted">
              <tr>
                <th className="px-3 py-2">Queue</th>
                <th className="px-3 py-2">Type</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Attempts</th>
                <th className="px-3 py-2">Error</th>
                <th className="px-3 py-2">Updated</th>
                <th className="px-3 py-2" />
              </tr>
            </thead>
            <tbody>
              {items.map((j) => (
                <tr key={`${j.queue}-${j.id}`} className="border-b border-border-subtle">
                  <td className="px-3 py-2">{j.queue}</td>
                  <td className="px-3 py-2 font-medium">{j.job_type}</td>
                  <td className={`px-3 py-2 ${statusTone(j.status)}`}>
                    {j.status}
                    {j.stuck ? " (stuck)" : ""}
                  </td>
                  <td className="px-3 py-2">
                    {j.attempts}
                    {j.max_attempts != null ? `/${j.max_attempts}` : ""}
                  </td>
                  <td className="max-w-xs truncate px-3 py-2 text-foreground-secondary">
                    {j.last_error || "—"}
                  </td>
                  <td className="px-3 py-2 text-foreground-muted">
                    {new Date(j.updated_at).toLocaleString("en-GB", { timeZone: "Asia/Jakarta" })}
                  </td>
                  <td className="px-3 py-2">
                    {j.queue === "job_queue" && (j.status === "failed" || j.status === "dead") ? (
                      <button
                        type="button"
                        disabled={busy === j.id}
                        className="text-xs font-medium underline disabled:opacity-50"
                        onClick={() => retry(j.id)}
                      >
                        Retry
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
