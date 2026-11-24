# TEST REPORT — Agent 14 Security & QA (Stage 3)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 §§29, 32  
**Branch:** `cursor/agent-14-security-qa-5c6b`  
**Prior main tip:** includes Agent 13 Analytics merge `0bf2d8c` (+ handoff PR #34)

---

## 1. Scope & method

| Layer | Approach |
| --- | --- |
| Security (SEC-*) | Code audit + concrete fixes + regression tests (auth, tenant, evidence, money, rate limits, headers) |
| Money / refunds / allocations | Targeted Go/API tests (preferred over Playwright) |
| E2E A–N | Matrix honesty: API coverage where Playwright is expensive; Stage 3 consolidated Playwright smoke only |
| UI | No exhaustive visual regression (preferences) |

Local gates run on this agent VM with Postgres `pikko_test` (migrations applied).

---

## 2. Security hardening shipped (Agent 14)

| ID | Fix |
| --- | --- |
| SEC-04 | `PublicFile` purpose allowlist — `payment_evidence` / PDFs / CSV no longer downloadable unauthenticated |
| SEC-02/04 | Catalog `UploadFile` rejects private evidence purposes; merchant `GetFile` gates evidence/PDF by RBAC |
| Money | `RejectPayment` requires `RowsAffected==1` before forcing order unpaid |
| Money | `CreateRefund` locks order + counts pending/processing/completed toward entitlement |
| Money | `ProcessRefund` re-checks completed sum ≤ entitled/paid before complete |
| Money | `ApplyStoreCreditToOrder` capped to `RemainingDueIDR` |
| SEC-04 | Refund `evidence_file_id` must belong to store + evidence purpose |
| SEC-07 | `ClientIP` prefers `X-Real-IP`; rightmost XFF fallback (stop leftmost spoof) |
| SEC-07 | Coarse API IP limiter fails closed in production/staging on DB errors |
| SEC-08 | Production forces `CookieSecure`; web middleware adds CSP + CORP |

**Already solid (verified, not rewritten):** SHA-256 hashed auth/portal/tracking/session tokens; store membership `404` not `403` for cross-store; tracking evidence stream joins; auth email rate limits; API CSP/HSTS unit test.

---

## 3. E2E A–N status

| ID | Status | What Agent 14 verified |
| --- | --- | --- |
| E2E-A Multi-store | **done** | Existing Playwright `multi-store` + Stage 3 smoke; cross-store 404 API tests across packages |
| E2E-B PDF→checkout | **done (API)** | `TestE2EB_PDFImportToPublicFlipbook` (Agent 11); Playwright deferred (smoke does not re-run full flipbook UI) |
| E2E-C Admin entry | **done (API)** | `TestAdminOrderCreateRevisionAndCSV` — was matrix “partial”; API+UI exist, Playwright deferred |
| E2E-D Purchasing | **done (API)** | Purchasing workspace + allocation API tests |
| E2E-E Partial availability | **done (API)** | `TestPurchasingAllocationConcurrency` marks unavailable + partial receiving |
| E2E-F Reconciliation | **done (API)** | `TestFinanceReconciliationE2EF` + `money.Reconcile` unit |
| E2E-G Store credit | **done (API)** | Feature flag + **new** `TestStoreCreditCannotExceedRemainingDue` |
| E2E-H Consolidated payment | **done (API)** | `TestFinanceConsolidatedPayment` |
| E2E-I Packing | **done (API)** | `TestPackingAndShipmentFlow` |
| E2E-J Address change | **done (API)** | Portal + merchant review (Agent 08) |
| E2E-K Live shopping | **done (API)** | `TestLiveShoppingQuickAddFeedAndQuota` |
| E2E-L Concurrency | **done (API)** | Stock/quota/allocation/refund version tests (live + purchasing + finance) |
| E2E-M Portal security | **done (API)** | `portal_security_test` IDOR cross-customer/store |
| E2E-N Domain migration | **done (code)** | Env/URL abstraction; cutover remains owner — no auto pikko.id |

Playwright Stage 3 smoke file: `apps/web/e2e/stage3-critical-smoke.spec.ts` (headers/CSP, auth page, multi-store). Full suite not re-run in this VM (no blocking CI; API gates are the money/security source of truth).

---

## 4. Test commands executed

```bash
export TEST_DATABASE_URL=postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable
cd services/api
go test ./internal/storage/ ./internal/httpapi/ ./internal/money/ ./internal/auth/ \
  ./internal/rbac/ ./internal/commerce/ ./internal/portal/ ./internal/analytics/ -count=1 -p 1
go build ./cmd/api
cd ../../apps/web && ./node_modules/.bin/tsc --noEmit
```

New / focused security tests:

- `TestPublicFileRejectsPaymentEvidence`
- `TestRefundPendingReservesEntitlement`
- `TestStoreCreditCannotExceedRemainingDue`
- `TestRejectPaymentAfterVerifyDoesNotUnpay`
- `TestClientIPPrefersRealIPOverSpoofedXFF`
- `TestIsPublicPurpose`

---

## 5. Residual risks (honest)

| Risk | Severity | Notes |
| --- | --- | --- |
| SEC-06 PDPA retention | Low (policy) | Privacy page partial; owner/counsel |
| Tracking token TTL | Low | Tokens hashed; optional expiry still product decision |
| Playwright UI depth for B/C/D/K | Low | API covers critical paths; Stage 3 smoke is thin by design |
| Rate-limit fail-open in non-prod | Info | Intentional for local DX |
| Nginx still prepends client XFF | Mitigated | API trusts `X-Real-IP` first |

---

## 6. Next — Agent 15

Integration/deploy runbook support only; **do not** auto-cutover to `pikko.id`. Preserve security fixes and analytics contracts.
