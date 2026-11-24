# Agent 08 → Agent 09 Handoff (Customer Portal → Packing)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-08-customer-portal-af34`  
**Prior:** Agent 07 main tip `97a9276` (PR #20/#21) + Friendly Commerce Workspace `bf346e2` (PR #22)
**Feature merge SHA:** `bd002b793e8ff814c0d25e05af5a834e78ec4a7b` (PR #23)
**Merged main SHA:** `bd002b793e8ff814c0d25e05af5a834e78ec4a7b`  

---

## 1. What Agent 08 delivered

| Area | Outcome |
| --- | --- |
| Migration `000012_customer_portal` | Portal tokens/sessions; refund preferences; address change requests; `customer_addresses.updated_at` |
| Auth | Store-scoped customer magic link (≥256-bit, SHA-256 stored); separate `pikko_portal_session` cookie |
| Dashboard | Orders list; order detail with payments, adjustments, finance summary, refunds |
| Refund prefs | Bank transfer details or store credit (when flagged) after verified session |
| Store credit | Portal ledger read when `FEATURE_STORE_CREDIT` + store enablement |
| Addresses | Saved addresses CRUD + default; post-order change requests; merchant review API |
| UI | `/portal/[storeSlug]/*` on Friendly Commerce Workspace tokens (Ink primary, Cloud White workspace) |
| Preserved | Guest checkout + `/track/[token]` |

**No pikko.id cutover. No destructive migrations.**

---

## 2. Acceptance gate evidence

- `go test ./internal/portal/` — E2E-M IDOR (cross-customer, cross-store, address IDOR)
- `go test ./internal/commerce/ ./internal/money/`
- `tsc --noEmit`, eslint, `next build` (portal routes registered)
- Merge without empty CI wait

### Security (E2E-M)

Customer A cannot read Customer B orders/addresses in the same store; portal session for store X cannot access store Y.

---

## 3. How later agents use portal

1. Do not reuse merchant `users`/`sessions` for customers — portal tables are separate.  
2. Surface packing/shipment status on portal order detail when Agent 09 ships `shipments`.  
3. Merchant address-change review: `POST /stores/{id}/address-change-requests/{id}/review` (`customers.write`).  
4. Refund prefs inform finance refund method selection; do not invent parallel bank tables.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Merchant address-change UI | API done; dashboard queue UI lite |
| Shipment carrier tracking | Agent 09 |
| Playwright portal journey | API-covered; UI smoke deferred to stage consolidation |

---

## 5. Next — Agent 09: Packing / Shipment

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` §21–22 PACK/SHIP / E2E-I  
- Prior handoff: this file  
- Reuse fulfillment statuses; notify portal/tracking when shipped  
- Preserve portal auth and guest tracking
