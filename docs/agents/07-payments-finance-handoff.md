# Agent 07 → Agent 08 Handoff (Finance → Portal)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-07-payments-finance-4959`  
**Prior:** Agent 06 main tip `c05b170` (PR #18/#19)  
**Feature merge SHA:** _(filled after merge)_  
**Merged main SHA:** _(filled after merge)_  

---

## 1. What Agent 07 delivered

| Area | Outcome |
| --- | --- |
| Migration `000011_payments_finance` | Settlement cols on orders; deposit policies; payment_kind/idempotency; allocations; financial_events/adjustments; refunds; store credit ledger |
| Money | `money.Reconcile` + deposit bps math; E2E-F unit coverage |
| Payments | Record deposit/final/partial/consolidated with allocations; verify-settlement recomputes payable |
| Adjustments | Unavailable/manual/credit deltas; MarkUnavailable emits ledger via stable idempotency |
| Refunds | Create entitlement capped to entitled; process with optimistic version |
| Store credit | Global `FEATURE_STORE_CREDIT` + `stores.store_credit_enabled`; grant/apply/ledger |
| RBAC | `finance.read` / `finance.write` on `/finance/*`; payments verify keeps `orders.payments` |
| UI | `/finance` reconciliation workspace + nav enabled |
| Preserved | Bank transfer evidence upload/verify/reject; purchasing allocations; order revisions |

**No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test` money + commerce (E2E-F, refund concurrency, consolidated, store-credit flag) + rbac
- `tsc --noEmit`, eslint finance files, `next build` (route `/stores/[storeId]/finance`)
- No full Playwright suite required for this handover
- Merge without empty CI wait

### Exact money (E2E-F)

`300_000` original / `250_000` paid / `110_000` unavailable → **`190_000` payable / `60_000` refund entitled**.

---

## 3. How later agents use finance

1. Read settlement via `GET /finance` or `GET /finance/orders/{id}` — do not recompute payable ad hoc.  
2. Never mutate verified `payments.amount_idr`; record compensating refunds/credits.  
3. Store credit requires **both** env `FEATURE_STORE_CREDIT=true` and per-store enablement.  
4. Unavailable mark with `apply_adjustment` writes `financial_adjustments` with key `unavail:{order}:{item}:{qty}`.  
5. Portal (08) should surface payment/refund/credit history from these tables; packing (09) may gate on `settlement_status` / payable.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Deposit schedule UX | Policies + payment kinds shipped; calendar/reminder UI lite |
| Checkout credit apply | Admin `/finance/credit/apply` done; guest checkout wire optional |
| Refund evidence upload UI | `evidence_file_id` supported; dedicated upload UX thin |
| Playwright E2E-F | API-covered; UI smoke deferred to stage consolidation |

---

## 5. Next — Agent 08: Customer Portal

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` §17 PORT-* / E2E-M  
- Prior handoff: this file  
- Reuse tracking + finance read models; do not invent parallel payment totals  
- Preserve merchant finance APIs and RBAC
