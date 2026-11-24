# Agent 14 → Agent 15 Handoff (Security & QA → Integration/Deploy)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md` §§29, 32)  
**Branch:** `cursor/agent-14-security-qa-5c6b`  
**Prior:** Agent 13 Analytics feature merge `0bf2d8c`; main tip includes handoff PR #34 (`481c0d3`)  
**Feature merge SHA:** _(filled after merge)_  
**Merged main SHA:** _(filled after merge)_

---

## 1. What Agent 14 delivered

| Area | Outcome |
| --- | --- |
| SEC-04 evidence privacy | `PublicFile` purpose allowlist; private purposes → 404 |
| SEC-02 file authz | Merchant `GetFile` purpose RBAC; catalog upload rejects `payment_evidence` |
| Money integrity | RejectPayment RowsAffected; refund pending reservation; credit remaining-due cap |
| SEC-07 rate limits | Prefer `X-Real-IP`; prod coarse limiter fail-closed |
| SEC-08 headers | Web CSP + CORP; force `CookieSecure` in production/staging |
| Tests | `security_hardening_test.go` + ClientIP/public purpose unit tests |
| Docs | `docs/TEST_REPORT.md`; matrix + PRD E2E/SEC honesty; this handoff |
| Playwright | Consolidated `e2e/stage3-critical-smoke.spec.ts` (not a huge visual suite) |
| Migration | **None** |
| Deploy / pikko.id | **None** — no cutover |

**Preserved:** Analytics contracts, live shopping, Friendly Workspace UI, Stage 1–2 ops.  
**No destructive migrations. No pikko.id cutover. No deploy.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go test` money/auth/rbac/commerce/portal/analytics/httpapi/storage (`TEST_DATABASE_URL`)
- `go build ./cmd/api`
- `tsc --noEmit` on web
- Playwright: Stage 3 smoke file added; full suite optional (API is money/security SoT)
- Merge without empty CI wait

---

## 3. How Agent 15 should use this

1. Read `docs/TEST_REPORT.md` for E2E A–N + residual risks.  
2. Keep PublicFile purpose allowlist and evidence routes intact during deploy hardening.  
3. Nginx already sets `X-Real-IP` from `$remote_addr` — do not strip it.  
4. **Do not** auto-cutover to `pikko.id`.  
5. SEC-06 PDPA remains owner/counsel.

---

## 4. Residual risks

| Item | Note |
| --- | --- |
| SEC-06 | Partial docs only |
| Tracking token TTL | Hashed; optional product expiry |
| Playwright UI for B/C/D/K | Intentionally API-primary |
| Staging CSP | May need connect-src tweaks if new third-party scripts added |

---

## 5. Next — Agent 15: Integration/Deploy

Start with:

- Source of truth: `docs/DEPLOYMENT_RUNBOOK.md` / `LAUNCH.md`  
- Prior handoff: this file + `docs/TEST_REPORT.md`  
- No auto pikko.id cutover  
- Preserve Agent 14 security fixes
