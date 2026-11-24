# Agent 06 → Agent 07 Handoff (Purchasing → Finance)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-06-purchasing-allocation-77d0`  
**Prior:** Agent 05 main tip `ee7e549` + soft-gradient `6778db1`  
**Feature merge SHA:** `257f3ab5a27c2bd03a69a2c2df345cefcbb1a789` (PR #18)  
**Merged main SHA:** `257f3ab5a27c2bd03a69a2c2df345cefcbb1a789`  
**PR:** https://github.com/rubicahq/pikko/pull/18  

---

## 1. What Agent 06 delivered

| Area | Outcome |
| --- | --- |
| Migration `000010_purchasing_allocation` | `vendor_id` / `received_qty` / `version` on purchasing records; receiving tables; alloc qty ≥ 0 |
| Eligibility | `GET /purchasing/eligibility` for closed/purchasing/awaiting batches |
| Aggregation | Store-wide by exact variant; filters campaign/batch/vendor/brand; group_by vendor/brand/campaign |
| Purchase records | Multi-vendor splits + optimistic `expected_version` |
| Allocation | `FOR UPDATE` locks; rejects oversell with 409; updates item availability |
| Unavailable | Impact analysis + mark-unavailable with payable adjust hooks; revisions preserve history |
| Receiving | Create/list/get + line patch; partial/complete; nudges batch/orders toward arrival |
| UI | `/purchasing`, `/purchasing/receiving/[id]`, `/vendors`; nav enabled with RBAC |
| Preserved | Checkout, admin/CSV orders, batch purchasing tab, soft-gradient tokens |

**No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test` commerce (incl. allocation concurrency), rbac, campaigns, team, catalog
- `tsc --noEmit`, eslint (purchasing/vendors), `next build`
- No full Playwright suite required for this handover
- Merge without empty CI wait

---

## 3. How later agents use purchasing

1. Aggregate via `/stores/{id}/purchasing` — do not re-sum order items ad hoc.  
2. Allocate only through batch allocations API (concurrency-safe).  
3. Unavailable payable hooks write `order_revisions` / `order_item_events`; Agent 07 should emit ledger/refund events from those, not rewrite snapshots.  
4. Receiving `received_qty` rolls into purchasing records — packing (09) reads allocations + availability.  
5. Gate with `purchasing.read` / `purchasing.write`.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| UNAV full finance | Adjust hooks only; refunds/credit/reconciliation → Agent 07 |
| RCV-03 packing | Status nudge only; pack queue → Agent 09 |
| Auto-allocate FIFO | Impact proposes; no bulk auto-allocate endpoint |
| Purchase orders entity | MVP-lite vendor_id on records (no separate PO docs) |

---

## 5. Next — Agent 07: Payments / Finance

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` §14–15 / FIN / REF / E2E-F  
- Prior handoff: this file  
- Consume unavailable adjustment deltas as ledger inputs  
- Preserve purchasing allocations + order revision history
