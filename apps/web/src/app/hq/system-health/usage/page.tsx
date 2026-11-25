"use client";

import { useEffect, useState } from "react";
import { KPI, RangePicker } from "@/components/hq/kpi";
import { fmtNum, hqFetch } from "@/lib/hq-api";

type Usage = {
  ai_calls: number | null;
  ai_success_calls: number | null;
  ai_tokens: number | null;
  ai_cost_estimate: number | null;
  ai_cost_note: string;
  storage_bytes: number | null;
  storage_files: number | null;
  email_sent: number | null;
  email_failed: number | null;
  email_cost_estimate: number | null;
  email_cost_note: string;
  unavailable: string[];
};

function fmtBytes(n: number | null): string {
  if (n === null || n === undefined) return "—";
  if (n < 1024) return `${n} B`;
  if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KB`;
  if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
  return `${(n / 1024 ** 3).toFixed(2)} GB`;
}

export default function HQUsagePage() {
  const [range, setRange] = useState("24h");
  const [data, setData] = useState<Usage | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await hqFetch<Usage>(`/system-health/usage?range=${range}`);
        if (!cancelled) {
          setData(res);
          setError(null);
        }
      } catch {
        if (!cancelled) setError("Unable to load usage.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [range]);

  const unavail = new Set(data?.unavailable || []);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-medium tracking-tight">Usage & Costs</h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            Measured consumption. Estimates labeled; unavailable is never shown as zero.
          </p>
        </div>
        <RangePicker value={range} onChange={setRange} />
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}

      {data ? (
        <div className="grid gap-x-10 md:grid-cols-3">
          <KPI label="AI calls" value={fmtNum(data.ai_calls)} />
          <KPI label="AI success" value={fmtNum(data.ai_success_calls)} />
          <KPI
            label="AI tokens"
            value={fmtNum(data.ai_tokens)}
            unavailable={unavail.has("ai_tokens")}
          />
          <KPI
            label="AI cost estimate"
            value="—"
            unavailable
            hint={data.ai_cost_note}
          />
          <KPI label="Storage" value={fmtBytes(data.storage_bytes)} hint={data.storage_files != null ? `${fmtNum(data.storage_files)} files` : undefined} />
          <KPI label="Email sent" value={fmtNum(data.email_sent)} />
          <KPI label="Email failed" value={fmtNum(data.email_failed)} />
          <KPI
            label="Email cost estimate"
            value="—"
            unavailable
            hint={data.email_cost_note}
          />
        </div>
      ) : null}
    </div>
  );
}
