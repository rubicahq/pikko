"use client";

import { useEffect, useState } from "react";
import { EmptyOrUnavailable } from "@/components/hq/kpi";
import { ServiceRow, hqFetch, statusTone } from "@/lib/hq-api";
import { Button } from "@/components/ui/button";

export default function HQServicesPage() {
  const [items, setItems] = useState<ServiceRow[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  async function load(refresh = false) {
    setLoading(true);
    try {
      const res = await hqFetch<{ items: ServiceRow[] }>(
        `/system-health/services${refresh ? "?refresh=1" : ""}`,
      );
      setItems(res.items);
      setError(null);
    } catch {
      setError("Unable to load services.");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load(true);
  }, []);

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-medium tracking-tight">Services</h1>
          <p className="mt-1 text-sm text-foreground-secondary">
            Lightweight checks only. Unknown means not probed — never assumed healthy from config alone.
          </p>
        </div>
        <Button type="button" loading={loading} onClick={() => load(true)}>
          Refresh checks
        </Button>
      </div>

      {error ? <p className="text-sm text-error">{error}</p> : null}

      {items.length === 0 ? (
        <EmptyOrUnavailable title="No service checks yet" />
      ) : (
        <div className="overflow-x-auto rounded border border-border bg-white">
          <table className="min-w-full text-left text-sm">
            <thead className="border-b border-border text-xs uppercase text-foreground-muted">
              <tr>
                <th className="px-3 py-2">Service</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Latency</th>
                <th className="px-3 py-2">Checked</th>
                <th className="px-3 py-2">Detail</th>
              </tr>
            </thead>
            <tbody>
              {items.map((s) => (
                <tr key={s.key} className="border-b border-border-subtle">
                  <td className="px-3 py-2 font-medium">{s.name}</td>
                  <td className={`px-3 py-2 capitalize ${statusTone(s.status)}`}>{s.status}</td>
                  <td className="px-3 py-2">
                    {s.latency_ms == null ? "—" : `${s.latency_ms} ms`}
                  </td>
                  <td className="px-3 py-2 text-foreground-muted">
                    {s.checked_at
                      ? new Date(s.checked_at).toLocaleString("en-GB", { timeZone: "Asia/Jakarta" })
                      : "—"}
                  </td>
                  <td className="px-3 py-2 text-foreground-secondary">{s.detail || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
