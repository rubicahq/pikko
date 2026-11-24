# Development Progress — PRD v3.0 Serial Agents

**Program:** Agents 01–15 across Stages 1–3  
**Foundation:** Prior v4 phases 0–7 live on `pikko.site` (`main` @ `353498a`)  
**Policy:** Incremental delivery on existing modular monolith. No rewrite. No automatic `pikko.id` cutover.

---

## 1. What is reused (do not rebuild)

| Area | Reuse |
| --- | --- |
| Monorepo + Docker/VPS | As-is |
| Auth magic link + sessions | Harden only |
| Multi-store membership (owner) | Extend roles additively |
| Design tokens + basic UI | Agent 02 expands |
| Storefront + 24 themes | Preserve |
| Catalog + variants + collections | Extend brands/attrs |
| AI PDF/CSV + flipbook | Agent 11 hardened |
| Checkout + bank transfer + tracking | Preserve |
| Preorder batches + business rules | Campaign facade later |
| Purchasing list + allocations | Agent 06 extends |
| Notifications + analytics + customers | Extend |
| Marketing site + SEO | Preserve |
| job_queue + outbox + email_jobs | Extend event types |
| Domain-agnostic config + LAUNCH prep | Keep; cutover owner-gated |

---

## 2. Stage plan mapped to Agents 02–15

### Stage 1 — Platform & Commerce Foundation

| Agent | Scope | Depends on | Exit gate |
| --- | --- | --- | --- |
| **01 Lead Architect** | Audit + docs (this program) | — | Docs merged; architecture supports tenancy/item ops/purchasing/revisions/finance |
| **02 Design System** | Expand tokens/components; Brex reference; nav/visual foundations | 01 handoff | DESIGN_SYSTEM complete; components cover PRD tables; no AI-generic look |
| **03 Auth & Multi-Store** | Session hardening; additive RBAC schema; invite scaffolding if in scope | 02 | Auth regression green; role CHECK widened safely; STORE-06 path clear |
| **04 Campaign & Catalog** | Campaign model (preorder+live prep); brands/attrs gaps | 03 | Catalog gaps addressed; preorder preserved |
| **05 Order Management** | Item ops, admin entry foundation, revision log | 04 | ORD-05/06/09 partial→done per acceptance; no break checkout |

### Stage 2 — Jastiper Operations

| Agent | Scope | Reuses | Exit gate |
| --- | --- | --- | --- |
| **06 Purchasing** | Vendors, multi-vendor, receiving handoff | PUR-01–03 | VEN/RCV + extended purchasing |
| **07 Payments/Finance** | Reconciliation, refunds, store credit, E2E-F | FIN-01–03 | Ledger + REF-* + exact money tests |
| **08 Customer Portal** | Auth portal beyond tracking | PORT-01, CRM | **Done** PORT-02–04; E2E-M/J API |
| **09 Packing/Shipment** | Pack queues, shipments | Fulfillment statuses | **Done** PACK/SHIP; E2E-I |
| **10 Communication** | WhatsApp templates + email completeness; RBAC invites polish | notifications/email | **Done** WA registry + Messages; email_jobs preserved |

### Stage 3 — Experience & Production Readiness

| Agent | Scope | Reuses | Exit gate |
| --- | --- | --- | --- |
| **11 PDF/AI Catalog** | Reliability, limits, review UX | PDF-01–05 | **Done** — hardened import + E2E-B API; no schema destruction |
| **12 Live Shopping & Storefront** | Live sessions + storefront polish | storefront | **Done** — LIVE-01–03 + E2E-K API |
| **13 Analytics** | North-star fulfilled items metric | analytics | AN-01 satisfied |
| **14 Security/QA** | Threat pass, TEST_REPORT, E2E A–N | all e2e | Matrix accurate; critical gaps ticketed |
| **15 Integration/Deploy** | Runbook validation; smoke; **no auto cutover** | deploy scripts | DEPLOYMENT_RUNBOOK current; owner checklist only for pikko.id |

---

## 3. Agent 01 status

| Deliverable | Status |
| --- | --- |
| `PIKKO_MASTER_PRD.md` | Done |
| `ARCHITECTURE.md` | Done |
| `DATABASE_SCHEMA.md` | Done |
| `API_CONTRACTS.md` | Done |
| `REQUIREMENTS_MATRIX.md` | Done |
| `DEVELOPMENT_PROGRESS.md` | Done (this file) |
| `DEPLOYMENT_RUNBOOK.md` | Done |
| `docs/agents/01-lead-architect-handoff.md` | Done |
| `DESIGN_SYSTEM.md` stub update | Done |
| App feature code | **Not in scope** |

---

## 3b. Agent 02 status (Design System)

| Deliverable | Status |
| --- | --- |
| `DESIGN_SYSTEM.md` full spec | Done |
| Tokens type ramp + layout | Done (`packages/design-tokens`) |
| Shared UI (table/DataTable/dialog/sheet/tabs/select/empty/error/skeleton/status/toast/page-header) | Done |
| App shell + NAV-02 grouped nav (hide unimplemented) | Done |
| Orders/Customers/Catalog wired to primitives | Done |
| `/dev/components` + design-system Playwright | Done |
| `docs/agents/02-design-system-handoff.md` | Done |
| DB migrations / pikko.id cutover | **Not in scope** |

---

## 3c. Agent 03 status (Auth & Multi-Store / RBAC)

| Deliverable | Status |
| --- | --- |
| Migration `000007_team_rbac` (widen roles + `team_invites`) | Done |
| `internal/rbac` permission matrix + server enforcement | Done |
| Team invite/accept/resend/cancel/revoke APIs + Brevo email | Done |
| Team UI + role-aware nav (NAV-03) + store switcher roles | Done |
| Setup checklist payment/fulfillment gap fix | Done |
| Tenant isolation + invite integration tests + Playwright | Done |
| `docs/agents/03-auth-multistore-handoff.md` | Done |
| Magic-link auth / owner multi-store preserved | Yes |
| pikko.id cutover | **Not in scope** |

---

## 3d. Agent 04 status (Campaign & Catalog)

| Deliverable | Status |
| --- | --- |
| Migration `000008_campaign_catalog` | Done |
| `internal/campaigns` + HTTP `/campaigns` | Done (preorder links batch; live shell + flags) |
| Brands / vendors / custom attributes foundations | Done |
| Campaign-product overrides (price/availability/phase) | Done |
| Nav Campaigns + RBAC `campaign.*` | Done |
| Preorder APIs/UI preserved + synced | Yes |
| `docs/agents/04-campaign-catalog-handoff.md` | Done |
| Live shopping ops (quick-add feed) | Deferred Agent 12 |
| pikko.id cutover | **Not in scope** |

---

## 3e. Agent 05 status (Order Management)

| Deliverable | Status |
| --- | --- |
| Migration `000009_order_management` | Done |
| Admin create + spreadsheet drafts + bulk validate | Done |
| CSV order import (template/preview/match/import) | Done |
| Item qty records + revisions / item-events | Done |
| Campaign association on checkout + admin orders | Done |
| RBAC `orders.*` on new routes | Done |
| UI: `/orders/new`, `/orders/import`, detail revisions | Done |
| `docs/agents/05-order-management-handoff.md` | Done |
| Guest checkout / payments / tracking preserved | Yes |
| pikko.id cutover | **Not in scope** |

---

## 3f. Agent 06 status (Purchasing & Allocation)

| Deliverable | Status |
| --- | --- |
| Migration `000010_purchasing_allocation` | Done |
| Store-wide aggregation (campaign/vendor/brand/variant) | Done |
| Purchasing eligibility + workspace UI | Done |
| Multi-vendor purchase records + version concurrency | Done |
| Allocation FOR UPDATE + concurrency test | Done |
| Unavailable impact + order adjustment hooks (revisions) | Done |
| Receiving foundation + UI | Done |
| Nav Purchasing + Vendors with RBAC | Done |
| `docs/agents/06-purchasing-allocation-handoff.md` | Done |
| Checkout / admin orders preserved | Yes |
| pikko.id cutover | **Not in scope** |

---

## 3g. Agent 07 status (Payments & Financial Reconciliation)

| Deliverable | Status |
| --- | --- |
| Migration `000011_payments_finance` | Done |
| `money.Reconcile` + deposit math (integer IDR) | Done |
| Deposit policies + multi/consolidated payments + allocations | Done |
| Financial adjustments + events ledger | Done |
| Refunds workflow + version concurrency | Done |
| Store credit ledger (`FEATURE_STORE_CREDIT` + per-store flag) | Done |
| Finance RBAC + `/finance` workspace UI | Done |
| E2E-F / concurrency / consolidated API tests | Done |
| Bank transfer evidence verify preserved | Yes |
| `docs/agents/07-payments-finance-handoff.md` | Done |
| pikko.id cutover | **Not in scope** |

---

## 3h. Agent 08 status (Customer Portal)

| Deliverable | Status |
| --- | --- |
| Migration `000012_customer_portal` | Done |
| Portal magic-link auth + session cookie | Done |
| Dashboard / orders / refund prefs / addresses | Done |
| E2E-M IDOR API tests | Done |
| Guest tracking preserved | Yes |
| `docs/agents/08-customer-portal-handoff.md` | Done |
| pikko.id cutover | **Not in scope** |

---

## 3i. Agent 09 status (Packing & Shipment)

| Deliverable | Status |
| --- | --- |
| Migration `000013_packing_shipment` | Done |
| Packing queue by order / customer | Done |
| Pack checklist + mark packed → ready_* | Done |
| Shipments carrier/tracking + multi-parcel | Done |
| Customer email via email_jobs + outbox | Done |
| Portal + guest tracking show shipments | Done |
| Nav Packing/Shipments + packing.* RBAC | Done |
| E2E-I API tests | Done |
| `docs/agents/09-packing-shipment-handoff.md` | Done |
| pikko.id cutover | **Not in scope** |

---

## 3j. Agent 10 status (Communication / WhatsApp)

| Deliverable | Status |
| --- | --- |
| Migration `000014_whatsapp_templates` | Done |
| Code registry WA-V01…V08 + WA-C01…C18 | Done |
| Preview / open / copy + `wa.me` deep-links | Done |
| Optional `whatsapp_template_sends` log | Done |
| Messages UI + nav (`notifications.read`) | Done |
| Callable from vendors / purchasing / packing / shipments / orders | Done |
| Brevo `email_jobs` preserved | Yes |
| `docs/agents/10-communication-handoff.md` | Done |
| Stage 2 complete | Yes |
| pikko.id cutover | **Not in scope** |

---

## 3k. Agent 11 status (PDF/AI Catalog Hardening)

| Deliverable | Status |
| --- | --- |
| Migration `000015_pdf_import_hardening` | Done (additive) |
| Confidence + review_status + bulk review APIs | Done |
| Review UX (filter, reject, bulk confirm) | Done |
| CSV clearer errors / edge cases | Done |
| Flipbook hotspot clamp + drag + public polish | Done |
| Mock AI without key; OpenAI-compat path | Done |
| E2E-B API chain test | Done |
| `docs/agents/11-pdf-ai-catalog-handoff.md` | Done |
| pikko.id cutover | **Not in scope** |

---

## 3l. Agent 12 status (Live Shopping & Storefront)

| Deliverable | Status |
| --- | --- |
| Migration `000016_live_shopping` | Done (additive) |
| Live console (quick-add + feed) | Done |
| Public live listing + session page | Done (theme-respecting) |
| Quota concurrency safety | Done (`FOR UPDATE` + sold sum) |
| Nav / RBAC (`campaign.read` + `orders.write` for quick-add) | Done |
| E2E-K API test | Done |
| `docs/agents/12-live-shopping-handoff.md` | Done |
| pikko.id cutover | **Not in scope** |

---

## 4. Open decisions (non-blocking for Agent 13+)

1. Customer portal auth model: separate customer identities (shipped Agent 08).
2. Campaign table chosen (`campaigns` + linked `preorder_batches`); live ops shipped Agent 12.
3. Commercial pricing / counsel legal — owner (unchanged).
4. pikko.id cutover date — owner only.
5. Checkout-time store credit apply UX — optional follow-up; admin apply shipped.

---

## 5. Historical v4 progress (reference)

Phases 0–7 completed on pikko.site: foundation, auth, storefront/catalog/import, checkout, preorder/rules, merchant efficiency, marketing/ops, domain cutover **prep**. See prior `PRODUCT_REQUIREMENTS.md` and phase handoffs in project store `internal/phase-*-handoff.md`.
