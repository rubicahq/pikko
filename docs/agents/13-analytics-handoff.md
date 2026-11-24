# Agent 13 → Agent 14 Handoff (Operational Analytics → Security/QA)

**Date:** 2026-11-25 (Asia/Jakarta)  
**PRD:** v3.0 (`docs/PIKKO_MASTER_PRD.md`)  
**Branch:** `cursor/agent-13-analytics-8f4b`  
**Prior:** Agent 12 feature merge `b2a9307` (Live Shopping); main tip includes handoff PR #32  
**Feature merge SHA:** `0bf2d8cbb62e2def5e2e7a6f805098d7ec04a437` (PR #33)  
**Merged main SHA:** `0bf2d8cbb62e2def5e2e7a6f805098d7ec04a437`

---

## 1. What Agent 13 delivered

| Area | Outcome |
| --- | --- |
| AN-01 North star | `fulfilled_order_items` (+ value, all-time) from paid orders in `shipped`/`delivered`/`collected`; timed via `order_status_history` |
| AN-02 | Activation / engagement / operational / retention blocks on overview API |
| AN-03 | `RecordEvent` + `POST …/analytics/events`; emit on payment verify, fulfillment ship/collect, live quick-add |
| AN-04 | Empty arrays / zero metrics when no rows — never fabricated |
| DASH widgets | Overview attention + Analytics ops rows (packing, refunds, live, payments) |
| UI | Analytics + Overview vertical hierarchy (Friendly Commerce Workspace) |
| Migration | **None** — aggregates existing tables after `000016_live_shopping` |
| Tests | `TestAnalyticsNorthStarAndEmptyStates` (needs `TEST_DATABASE_URL`) |
| Docs | Matrix / progress / PRD AN+DASH / contracts / this handoff |

**Preserved:** Live shopping, packing, finance, Stage 1–2 surfaces.  
**No pikko.id cutover. No destructive migrations. No deploy.**

---

## 2. Acceptance gate evidence

Per owner testing policy:

- `go build ./...` (api)
- `go test ./internal/analytics/` (skips without Postgres)
- `tsc --noEmit` on web
- No Playwright required for this handover
- Merge without empty CI wait

---

## 3. How later agents use analytics

1. North star = **item quantity** successfully fulfilled, not order count.  
2. Read `GET /api/v1/stores/{id}/analytics?tz=Asia/Jakarta&from=&to=` — do not recompute ad hoc on Overview.  
3. Sink events via `analytics.Emit` / `POST …/analytics/events`; never invent dashboard numbers.  
4. Unpaid order value is reported separately and must not be labeled revenue.  
5. Agent 14 can treat AN-* / DASH-01–04 as done for matrix accuracy; deepen E2E coverage if needed.

---

## 4. Known limitations

| Item | Note |
| --- | --- |
| Postgres in this agent VM | Integration test skipped when DB unavailable |
| Purchasing-specific widget | Covered indirectly via packing queue / ops; dedicated purchasing depth optional |
| Storefront pageview events | Not instrumented on public FE yet; sink ready |
| Playwright analytics UI | Deferred to stage consolidation smoke |

---

## 5. Next — Agent 14: Security/QA

Start with:

- Source of truth: `docs/PIKKO_MASTER_PRD.md` SEC-* / E2E A–N / TEST_REPORT  
- Prior handoff: this file  
- Preserve analytics contracts and empty-state honesty  
- No auto pikko.id cutover
