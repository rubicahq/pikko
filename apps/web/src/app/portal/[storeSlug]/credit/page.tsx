"use client";

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { apiFetch, formatIDR, ApiError } from "@/lib/api";
import { PortalShell } from "@/components/portal/portal-shell";
import { usePortalSession } from "@/components/portal/use-portal-session";

type Ledger = {
  id: string;
  entry_type: string;
  amount_idr: number;
  balance_after_idr: number;
  note?: string | null;
  created_at: string;
};

export default function PortalCreditPage() {
  const params = useParams();
  const storeSlug = String(params.storeSlug || "");
  const { session, loading: sessionLoading } = usePortalSession(storeSlug);
  const [balance, setBalance] = useState(0);
  const [ledger, setLedger] = useState<Ledger[]>([]);
  const [disabled, setDisabled] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch<{
        credit: { balance_idr: number };
        ledger: Ledger[];
      }>(`/api/v1/portal/${storeSlug}/credit`);
      setBalance(res.credit.balance_idr);
      setLedger(res.ledger || []);
      setDisabled(false);
      setError(null);
    } catch (err) {
      if (err instanceof ApiError && err.code === "store_credit_disabled") {
        setDisabled(true);
        setError(null);
        return;
      }
      setError(err instanceof Error ? err.message : "Unable to load credit.");
    }
  }, [storeSlug]);

  useEffect(() => {
    if (session) load();
  }, [session, load]);

  if (sessionLoading || !session) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-workspace">
        <p className="text-sm text-foreground-secondary">Loading…</p>
      </div>
    );
  }

  return (
    <PortalShell
      storeSlug={storeSlug}
      storeName={session.store.name}
      customerEmail={session.customer.email}
      primaryColor={session.store.primary_color}
    >
      <h1 className="text-page font-bold tracking-tight text-foreground">Store credit</h1>
      <p className="mt-1.5 text-body text-foreground-secondary">
        Credit history for this store when the merchant has enabled store credit.
      </p>

      {disabled ? (
        <p className="mt-8 rounded-xl border border-border bg-background p-5 text-sm text-foreground-secondary">
          Store credit is not enabled for this shop yet.
        </p>
      ) : (
        <>
          <p className="mt-8 text-metric font-bold tabular-nums text-foreground">
            {formatIDR(balance)}
          </p>
          <p className="mt-1 text-label text-foreground-muted">Available balance</p>

          <section className="mt-8">
            <h2 className="text-section font-bold text-foreground">History</h2>
            <ul className="mt-4 divide-y divide-border-subtle overflow-hidden rounded-xl border border-border bg-background">
              {ledger.length === 0 ? (
                <li className="px-4 py-8 text-sm text-foreground-secondary">No credit activity yet.</li>
              ) : (
                ledger.map((e) => (
                  <li key={e.id} className="flex items-center justify-between gap-3 px-4 py-3 text-sm">
                    <div>
                      <p className="capitalize font-medium text-foreground">
                        {e.entry_type.replaceAll("_", " ")}
                      </p>
                      {e.note ? (
                        <p className="text-label text-foreground-muted">{e.note}</p>
                      ) : null}
                    </div>
                    <div className="text-right">
                      <p className="tabular-nums font-semibold">
                        {e.entry_type === "debit" ? "−" : "+"}
                        {formatIDR(e.amount_idr)}
                      </p>
                      <p className="text-label tabular-nums text-foreground-muted">
                        bal {formatIDR(e.balance_after_idr)}
                      </p>
                    </div>
                  </li>
                ))
              )}
            </ul>
          </section>
        </>
      )}

      {error ? <p className="mt-4 text-sm text-error">{error}</p> : null}
    </PortalShell>
  );
}
