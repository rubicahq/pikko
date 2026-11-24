# Requirements Matrix — PRD v3.0

**Statuses:** `done` | `partial` | `missing` | `owner` | `deferred`  
**Baseline:** `main` @ Agent 02 merge `54181f5` (+ Agent 03 RBAC), migrations `000001`–`000007`  
**Prior v4 matrix:** superseded for new work; phase IDs mapped below where relevant.

---

## Legend

| Status | Meaning |
| --- | --- |
| done | FE+BE+DB working in production path |
| partial | Real implementation incomplete vs v3 acceptance |
| missing | Not built |
| owner | Requires human/business action |
| deferred | Explicitly out of current stage |

---

## Infrastructure & platform

| ID | Requirement | Status | Files / evidence | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| INFRA-DOM-01 | pikko.site surfaces live | done | nginx examples, DEPLOYMENT | smoke | — |
| INFRA-DOM-02 | Domain-agnostic URL generation | done | `urls/`, `lib/site.ts` | Go + Playwright phase7 | — |
| INFRA-DOM-03 | pikko.id cutover | owner | `LAUNCH.md` / RUNBOOK | — | Not automatic |
| INFRA-EMAIL-01 | Brevo + email_jobs | done | `email/` | drain tests | Keys on VPS |
| P0-STACK | Next+Go+Postgres+tokens+compose | done | monorepo | go test, tsc, e2e | — |

---

## Design system

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| DS-01 | Brand tokens #FF3D00 + neutrals | done | `packages/design-tokens` | — | Type ramp + layout tokens expanded Agent 02 |
| DS-02 | Component library completeness | done | `components/ui` (table, DataTable, dialog, sheet, tabs, select, empty/error/skeleton, status, toast, page-header, filter-bar + prior primitives) | `/dev/components`, design-system e2e | Toast provider opt-in per tree; not global yet |
| DS-03 | Brex-like marketing discipline | done | marketing + `DESIGN_SYSTEM.md` rules | phase6 e2e | Preserve hero budget; no redesign of live marketing in this pass |
| DS-04 | 24 storefront themes | done | `themes/themes.go` | themes_test | — |
| DS-05 | Product/marketing type ramp wired | done | `tokens.css`, `tailwind.preset.js` | build | — |
| DS-06 | Dense data table patterns | done | `ui/table.tsx`, `ui/data-table.tsx`; Orders/Customers wired | design-system e2e | Server lists use primitives; interactive DataTable for client workflows |
| DS-07 | Empty / loading / error states | done | `empty-state`, `skeleton`, `error-state` | `/dev/components` | — |
| DS-08 | Status badges for ops labels | done | `status-badge.tsx` | `/dev/components` | Extend map as new statuses ship |
| DS-09 | App shell + mobile sheet nav | done | `app-shell`, `mobile-nav`, `sidebar` | multi-store / responsive | Collapsed desktop icon-rail not implemented (token reserved) |

---

## Auth / onboarding / multi-store

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| AUTH-01–08 | Magic link auth suite | done | `auth/`, auth handlers | integration + Playwright | — |
| ONB-01–04 | First-store onboarding | done | `stores/`, onboarding UI | auth-onboarding e2e | — |
| STORE-01–05 | Multi-store tenancy | done | stores + middleware | multi-store e2e | — |
| STORE-06 | Team roles beyond owner | done | `000007`, `rbac/`, `team/`, Team UI | team integration + team-rbac e2e | Owner remains superset; archive owner-only |

---

## Navigation

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| NAV-01 | Store-scoped nav | done | `app-shell`, `sidebar`, store switcher | multi-store e2e | Ops modules appear when implemented flags flip |
| NAV-02 | Workspace/Operations/Configuration | done | `nav-config.ts`, `sidebar-nav.tsx` | design-system e2e | Unimplemented items hidden (`implemented:false`) |
| NAV-03 | Role-aware nav | done | `nav-config.ts` + `lib/rbac.ts` filters by permissions | team-rbac e2e | Server still authoritative |

---

## Campaigns / catalog / PDF

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| CAMP-01–04 | Preorder batches | done | `commerce/preorders.go`, UI | phase4 e2e | Preserved; synced into campaigns |
| CAMP-05 | Unified campaign abstraction | done | `campaigns/`, `/campaigns` UI, migration `000008` | campaigns integration | Live behavior still Agent 12 |
| LIVE-01–03 | Live shopping | partial | campaign type=`live` + flags | campaigns integration | Quick-add / feed ops → Agent 12 |
| CAT-01–03,07 | Catalog core + snapshots | done | `catalog/` | phase2 e2e | Collections unchanged |
| CAT-04 | Brands entity | done | `brands` + `products.brand_id` | campaigns integration | Text `brand` column kept |
| CAT-05–06 | Vendors link / custom attrs | done (foundation) | `vendors`, `product_vendors`, `attribute_*` | campaigns integration | Purchasing UX → Agent 06 |
| PDF-01–05 | AI PDF + CSV + flipbook | done | imports, flipbook, ai | catalog-import, flipbook e2e | AI quality depends on key/model |

---

## Orders / purchasing / unavailable

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| ORD-01–04,11 | Checkout, bank transfer, tracking, dimensions | done | `commerce/*` | phase3/4 e2e | — |
| ORD-05 | Item-level operations | partial | allocations | phase4 | No general item ops API |
| ORD-06–07 | Admin / spreadsheet entry | missing | — | — | Agent 05 |
| ORD-08 | Order CSV | missing | catalog CSV only | — | Agent 05 |
| ORD-09 | Full revisions | partial | `order_status_history` | — | Agent 05 |
| ORD-10 | Deposit/final payments | missing | — | — | Agent 07 |
| PUR-01–03 | Purchasing + allocate | done | `purchasing.go` | phase4 integration | UI depth varies |
| PUR-04–06 | Multi-vendor / receiving | missing | — | — | Agent 06 |
| UNAV-01 | Unavailable adjust | partial | allocations + email | phase4 | No full finance adjust |
| UNAV-02–03 | Portal visibility / auto proposals | missing | — | — | 07/08 |

---

## Finance / refunds / customers / portal

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| FIN-01–03 | Integer money + bank verify | done | `money/`, payments | unit + e2e | — |
| FIN-04–08 | Reconciliation / ledger | missing/partial | — | — | Agent 07; E2E-F |
| REF-01 | Refund status enums | partial | payment_status values | — | No workflow |
| REF-02–04 | Store credit + audit | missing | — | — | Agent 07 |
| CRM-01–02 | Customers list/detail | done | customers handlers | phase5 | — |
| CRM-03–05 | Addresses UX / merge | partial/missing | `customer_addresses` | — | Agent 08 |
| PORT-01 | Tracking token page | done | `/track/[token]` | phase3 | — |
| PORT-02–04 | Auth portal | missing | — | — | Agent 08 |
| ADDR-01 | Checkout address | done | checkout | phase3 | — |
| ADDR-02–03 | Saved + change request | partial/missing | — | — | Agent 08 |

---

## Vendors / receiving / packing / shipments / WA / RBAC

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| VEN-01–03 | Vendors | partial | `vendors` + product links (Agent 04) | campaigns integration | Purchasing splits / UI → Agent 06 |
| RCV-01–03 | Receiving | missing | — | — | Agent 06 |
| PACK-01–03 | Packing | missing | fulfillment status manual | — | Agent 09 |
| SHIP-01–03 | Shipments | missing/partial | emails only | — | Agent 09 |
| WA-* | WhatsApp templates | missing | email templates exist | — | Agent 10 |
| RBAC-01–03 | Team RBAC | done | `rbac/`, `team/`, permission middleware, invites | team tests + e2e | Agent 10 may polish WA copy |
| RBAC-04 | Audit membership | done | invite/member audit_logs actions | team integration | — |

---

## Status / dashboard / rules / analytics / security

| ID | Requirement | Status | Files | Tests | Limitations |
| --- | --- | --- | --- | --- | --- |
| STAT-01,04 | Multi-dimension order statuses | done | orders schema | commerce tests | — |
| STAT-02–03 | Item + settlement dimensions | partial/missing | — | — | 05/07 |
| DASH-01,03 | Overview + checklist | done | overview UI | — | — |
| DASH-02,04 | Rich ops tasks / widgets | partial/missing | — | — | Later agents |
| RULE-01–05 | Business rules | done | `rules.go` | rules_test + e2e | — |
| AN-01–04 | Analytics | partial | `analytics/` | phase5 | North-star item metric incomplete |
| SEC-01–05,07–08 | Tenant + tokens + headers + limits | done | middleware, auth | security_test | Continuous |
| SEC-06 | PDPA retention policy | partial | privacy page | — | Owner/counsel |

---

## E2E scenarios

| ID | Status | Notes |
| --- | --- | --- |
| E2E-A Multi-store | done | Playwright |
| E2E-B PDF→checkout | partial | Pieces exist |
| E2E-C Admin entry | missing | Stage 1 |
| E2E-D Purchasing | partial | Stage 2 |
| E2E-E Partial availability | partial | Stage 2 |
| E2E-F Reconciliation math | missing | Stage 2 |
| E2E-G Store credit | missing | Stage 2 |
| E2E-H Consolidated payment | missing | Stage 2 |
| E2E-I Packing | missing | Stage 2 |
| E2E-J Address change | missing | Stage 2 |
| E2E-K Live shopping | missing | Stage 3 |
| E2E-L Concurrency | partial | Stage 1–3 |
| E2E-M Portal security | partial | Tracking only |
| E2E-N Domain migration readiness | done (code) | Cutover owner |

---

## Preserve inventory (do not rebuild)

Auth, multi-store, storefront, catalog, AI PDF/flipbook/CSV, checkout, bank transfer, preorder/batches/rules, purchasing/allocations APIs, notifications, analytics overview, customers, marketing, deploy/backup/domain-agnostic prep.

## Top gaps by stage

| Stage | Top missing |
| --- | --- |
| 1 | Campaign unification, admin order entry, item revisions (RBAC done by Agent 03) |
| 2 | Vendors/receiving, finance ledger/refunds/credit, portal, packing/shipments, WhatsApp |
| 3 | Live shopping, analytics north-star depth, full E2E A–N, security/QA pass, deploy hardening (still no auto pikko.id) |
