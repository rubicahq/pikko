"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { EmptyOrUnavailable, KPI, RangePicker } from "@/components/hq/kpi";
import {
  HQOverview,
  fmtMs,
  fmtNum,
  fmtPct,
  hqFetch,
  statusTone,
} from "@/lib/hq-api";

export default function HQOverviewPage() {
  const [range, setRange] = useState("24h");
  const [data, setData] = useState<HQOverview | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    setError(null);
    (async () => {
      try {
        const res = await hqFetch<HQOverview>(`/system-health/overview?range=${range}`);
        if (!cancelled) setData(res);
      } catch {
        if (!cancelled) setError("Unable to load overview.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [range]);

  const unavail = new Set(data?.unavailable || []);

  return (
    <div className="space-y-8">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-medium tracking-tight text-foreground">System Health</h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            Platform ops for Tim Pikko — measured data only.
          </p>
        </div>
        <RangePicker value={range} onChange={setRange} />
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}
      {!data && !error ? <p className="text-sm text-foreground-muted">Loading…</p> : null}

      {data ? (
        <>
          <div className="grid gap-x-10 gap-y-2 md:grid-cols-4">
            <KPI
              label="Unresolved issues"
              value={fmtNum(data.unresolved_issues)}
              unavailable={data.unresolved_issues === null}
            />
            <KPI
              label="Critical issues"
              value={fmtNum(data.critical_issues)}
              unavailable={data.critical_issues === null}
            />
            <KPI
              label="Affected stores"
              value={fmtNum(data.affected_stores)}
              unavailable={data.affected_stores === null}
            />
            <KPI
              label="API success rate"
              value={fmtPct(data.api_success_rate)}
              unavailable={unavail.has("api_success_rate")}
              hint={
                data.api_request_count != null
                  ? `${fmtNum(data.api_request_count)} requests measured`
                  : undefined
              }
            />
            <KPI
              label="P95 API latency"
              value={fmtMs(data.p95_latency_ms)}
              unavailable={unavail.has("p95_latency_ms")}
            />
            <KPI label="Failed jobs" value={fmtNum(data.failed_jobs)} unavailable={data.failed_jobs === null} />
            <KPI label="Stuck jobs" value={fmtNum(data.stuck_jobs)} unavailable={data.stuck_jobs === null} />
            <KPI
              label="Deploy version"
              value={data.deploy_version || "—"}
              unavailable={unavail.has("deploy_version")}
            />
          </div>

          <section>
            <div className="mb-3 flex items-center justify-between">
              <h2 className="text-sm font-medium text-foreground">Service health</h2>
              <Link href="/system-health/services" className="text-sm text-foreground-secondary hover:underline">
                View services
              </Link>
            </div>
            <div className="overflow-x-auto rounded border border-border bg-white">
              <table className="min-w-full text-left text-sm">
                <thead className="border-b border-border text-xs uppercase text-foreground-muted">
                  <tr>
                    <th className="px-3 py-2 font-medium">Service</th>
                    <th className="px-3 py-2 font-medium">Status</th>
                    <th className="px-3 py-2 font-medium">Detail</th>
                  </tr>
                </thead>
                <tbody>
                  {data.service_health.map((s) => (
                    <tr key={s.key} className="border-b border-border-subtle last:border-0">
                      <td className="px-3 py-2">{s.name}</td>
                      <td className={`px-3 py-2 capitalize ${statusTone(s.status)}`}>{s.status}</td>
                      <td className="px-3 py-2 text-foreground-secondary">{s.detail || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>

          <div className="grid gap-6 md:grid-cols-2">
            <section>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-sm font-medium">Recent critical issues</h2>
                <Link href="/system-health/errors" className="text-sm text-foreground-secondary hover:underline">
                  Error logs
                </Link>
              </div>
              {data.recent_critical_issues.length === 0 ? (
                <EmptyOrUnavailable title="No critical issues in range" />
              ) : (
                <ul className="divide-y divide-border rounded border border-border bg-white">
                  {data.recent_critical_issues.map((i) => (
                    <li key={i.id} className="px-3 py-2.5 text-sm">
                      <div className="font-medium">{i.title}</div>
                      <div className="text-xs text-foreground-muted">
                        {i.occurrence_count}× · {i.module || i.source}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </section>
            <section>
              <div className="mb-3 flex items-center justify-between">
                <h2 className="text-sm font-medium">Recently failed jobs</h2>
                <Link href="/system-health/jobs" className="text-sm text-foreground-secondary hover:underline">
                  Jobs
                </Link>
              </div>
              {data.recent_failed_jobs.length === 0 ? (
                <EmptyOrUnavailable title="No failed jobs in recent window" />
              ) : (
                <ul className="divide-y divide-border rounded border border-border bg-white">
                  {data.recent_failed_jobs.map((j) => (
                    <li key={`${j.queue}-${j.id}`} className="px-3 py-2.5 text-sm">
                      <div className="font-medium">{j.job_type}</div>
                      <div className="text-xs text-foreground-muted">
                        {j.queue} · {j.status}
                      </div>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </div>

          <section>
            <h2 className="mb-3 text-sm font-medium">Most affected stores</h2>
            {data.most_affected_stores.length === 0 ? (
              <EmptyOrUnavailable title="No store-scoped errors in range" />
            ) : (
              <div className="overflow-x-auto rounded border border-border bg-white">
                <table className="min-w-full text-left text-sm">
                  <thead className="border-b border-border text-xs uppercase text-foreground-muted">
                    <tr>
                      <th className="px-3 py-2 font-medium">Store</th>
                      <th className="px-3 py-2 font-medium">Errors</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.most_affected_stores.map((s) => (
                      <tr key={s.store_id} className="border-b border-border-subtle">
                        <td className="px-3 py-2">
                          {s.store_name || s.store_slug || s.store_id}
                        </td>
                        <td className="px-3 py-2">{s.count}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <p className="text-xs text-foreground-muted">
            Last monitoring update: {new Date(data.generated_at).toLocaleString("en-GB", { timeZone: "Asia/Jakarta" })} Asia/Jakarta
            {data.unavailable.length
              ? ` · Unavailable KPIs: ${data.unavailable.join(", ")}`
              : ""}
          </p>
        </>
      ) : null}
    </div>
  );
}
