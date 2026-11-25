"use client";

import { useEffect, useState } from "react";
import { EmptyOrUnavailable, RangePicker } from "@/components/hq/kpi";
import { Issue, hqFetch, statusTone } from "@/lib/hq-api";

export default function HQErrorsPage() {
  const [range, setRange] = useState("24h");
  const [q, setQ] = useState("");
  const [status, setStatus] = useState("");
  const [items, setItems] = useState<Issue[]>([]);
  const [total, setTotal] = useState(0);
  const [selected, setSelected] = useState<{
    issue: Issue;
    occurrences: { id: string; occurred_at: string; message: string; request_id?: string; endpoint?: string }[];
  } | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const params = new URLSearchParams({ range });
        if (q) params.set("q", q);
        if (status) params.set("status", status);
        const res = await hqFetch<{ items: Issue[]; total: number }>(
          `/system-health/issues?${params}`,
        );
        if (!cancelled) {
          setItems(res.items);
          setTotal(res.total);
          setError(null);
        }
      } catch {
        if (!cancelled) setError("Unable to load issues.");
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [range, q, status]);

  async function openIssue(id: string) {
    const res = await hqFetch<{
      issue: Issue;
      occurrences: { id: string; occurred_at: string; message: string; request_id?: string; endpoint?: string }[];
    }>(`/system-health/issues/${id}`);
    setSelected(res);
  }

  async function patchStatus(id: string, next: string) {
    await hqFetch(`/system-health/issues/${id}`, {
      method: "PATCH",
      body: { status: next },
    });
    setSelected(null);
    setStatus((s) => s); // noop to keep
    const params = new URLSearchParams({ range });
    if (q) params.set("q", q);
    if (status) params.set("status", status);
    const res = await hqFetch<{ items: Issue[]; total: number }>(`/system-health/issues?${params}`);
    setItems(res.items);
    setTotal(res.total);
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-medium tracking-tight">Error Logs</h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            Grouped issues — secrets sanitized. Validation 4xx are not unexpected errors.
          </p>
        </div>
        <RangePicker value={range} onChange={setRange} />
      </div>

      <div className="flex flex-wrap gap-2">
        <input
          className="min-w-[220px] flex-1 rounded border border-border bg-white px-3 py-2 text-sm"
          placeholder="Search request / issue / store ID"
          value={q}
          onChange={(e) => setQ(e.target.value)}
        />
        <select
          className="rounded border border-border bg-white px-3 py-2 text-sm"
          value={status}
          onChange={(e) => setStatus(e.target.value)}
        >
          <option value="">All statuses</option>
          <option value="unresolved">Unresolved</option>
          <option value="investigating">Investigating</option>
          <option value="resolved">Resolved</option>
          <option value="ignored">Ignored</option>
        </select>
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}

      {items.length === 0 && !error ? (
        <EmptyOrUnavailable title="No issues in range" detail={`${total} total matching filters.`} />
      ) : (
        <div className="overflow-x-auto rounded border border-border bg-white">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-border text-xs uppercase text-foreground-muted">
              <tr>
                <th className="px-3 py-2">Severity</th>
                <th className="px-3 py-2">Title</th>
                <th className="px-3 py-2">Module</th>
                <th className="px-3 py-2">Count</th>
                <th className="px-3 py-2">Stores</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Last seen</th>
              </tr>
            </thead>
            <tbody>
              {items.map((i) => (
                <tr
                  key={i.id}
                  className="cursor-pointer border-b border-border-subtle hover:bg-background-subtle"
                  onClick={() => openIssue(i.id)}
                >
                  <td className={`px-3 py-2 capitalize ${statusTone(i.severity)}`}>{i.severity}</td>
                  <td className="px-3 py-2 font-medium">{i.title}</td>
                  <td className="px-3 py-2 text-foreground-secondary">{i.module || i.source}</td>
                  <td className="px-3 py-2">{i.occurrence_count}</td>
                  <td className="px-3 py-2">{i.affected_store_count}</td>
                  <td className="px-3 py-2 capitalize">{i.status}</td>
                  <td className="px-3 py-2 text-foreground-muted">
                    {new Date(i.last_seen_at).toLocaleString("en-GB", { timeZone: "Asia/Jakarta" })}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {selected ? (
        <div className="fixed inset-0 z-50 flex justify-end bg-ink/30" onClick={() => setSelected(null)}>
          <aside
            className="h-full w-full max-w-lg overflow-y-auto bg-white p-5 shadow-lg"
            onClick={(e) => e.stopPropagation()}
          >
            <div className="flex items-start justify-between gap-3">
              <h2 className="text-lg font-medium">{selected.issue.title}</h2>
              <button type="button" className="text-sm text-foreground-secondary" onClick={() => setSelected(null)}>
                Close
              </button>
            </div>
            <p className="mt-2 text-sm text-foreground-secondary">
              {selected.issue.severity} · {selected.issue.status} · {selected.issue.occurrence_count} occurrences
            </p>
            <div className="mt-4 flex flex-wrap gap-2">
              {["investigating", "resolved", "ignored", "unresolved"].map((s) => (
                <button
                  key={s}
                  type="button"
                  className="rounded border border-border px-2.5 py-1 text-xs capitalize hover:bg-background-subtle"
                  onClick={() => patchStatus(selected.issue.id, s)}
                >
                  Mark {s}
                </button>
              ))}
            </div>
            <h3 className="mt-6 text-sm font-medium">Recent occurrences</h3>
            <ul className="mt-2 space-y-2">
              {selected.occurrences.map((o) => (
                <li key={o.id} className="rounded border border-border px-3 py-2 text-sm">
                  <div>{o.message}</div>
                  <div className="mt-1 text-xs text-foreground-muted">
                    {new Date(o.occurred_at).toLocaleString("en-GB", { timeZone: "Asia/Jakarta" })}
                    {o.request_id ? ` · req ${o.request_id}` : ""}
                    {o.endpoint ? ` · ${o.endpoint}` : ""}
                  </div>
                </li>
              ))}
            </ul>
          </aside>
        </div>
      ) : null}
    </div>
  );
}
