# PIKKO — Master Product Requirements Document

**Version:** 3.0  
**Status:** Source of truth for serial multi-agent development (Agents 01–15)  
**Supersedes:** Prior phased PRD v4.0 for *new* work. Live `pikko.site` build (phases 0–7) is preserved and mapped incrementally.  
**Audit baseline:** `main` @ `353498a` (2026-11-24), migrations through `000006`.

---

# 0. MASTER DEVELOPMENT INSTRUCTIONS

## 0.1 Purpose

Design, develop, integrate, test, and prepare Pikko for production as a fully functional commerce & operations SaaS — not a landing page, prototype, or disconnected feature set.

## 0.2 Development approach

Serial multi-agent development. Each agent completes assigned responsibilities before handoff. Do not run concurrent agents that modify the same schema, shared components, or business logic.

**Three stages only:**

1. Platform & Commerce Foundation (Agents 01–05)
2. Jastiper Operations (Agents 06–10)
3. Commerce Experience & Production Readiness (Agents 11–15)

## 0.3 Mandatory initial audit (Agent 01)

Before implementing features:

- Inspect repository frontend/backend architecture
- Inspect schemas and migrations
- Inspect auth, authorization, sessions
- Inspect APIs, UI, design tokens
- Inspect deployment and environment variables
- Identify preserve / incomplete / conflict / security risks

**Rules:** Do not rebuild working functionality. Do not delete production data, reset DBs, overwrite secrets, or run destructive migrations without explicit approval.

## 0.4 Agent execution protocol

Every agent must: read this PRD → read prior handoff → inspect implementation → define scope → document current vs expected → identify affected components → implement → tests → FE/BE integration → desktop/mobile → limitations → handoff.

Do not mark complete if placeholder-only, non-functional buttons, incomplete APIs, or disconnected screens.

## 0.5 Required documentation

| File | Owner |
| --- | --- |
| `docs/PIKKO_MASTER_PRD.md` | Agent 01 (this file) |
| `docs/ARCHITECTURE.md` | Agent 01 |
| `docs/DESIGN_SYSTEM.md` | Agent 02 (stub updated by 01) |
| `docs/DATABASE_SCHEMA.md` | Agent 01 |
| `docs/API_CONTRACTS.md` | Agent 01 |
| `docs/REQUIREMENTS_MATRIX.md` | All (01 seeds) |
| `docs/DEVELOPMENT_PROGRESS.md` | All (01 seeds Stage plan) |
| `docs/TEST_REPORT.md` | Agents 14+ (created when tests run) |
| `docs/DEPLOYMENT_RUNBOOK.md` | Agent 01 (evolves DEPLOYMENT/LAUNCH) |
| `docs/agents/{nn}-*-handoff.md` | Each agent |

Every requirement: unique ID, status, related files, test coverage, known limitations.

## 0.6 Pause conditions

Pause only for: destructive operations, unavailable credentials, critical architectural decisions, security-sensitive external actions requiring approval. Do **not** auto-cutover to `pikko.id`.

---

# 1. PRODUCT OVERVIEW

## 1.1 Identity

| Field | Value |
| --- | --- |
| Name | Pikko |
| Tagline | Sell your way. |
| Category | Commerce & Operations Platform for Independent Sellers |
| Beachhead | Jastipers (personal shoppers) in Indonesia |

## 1.2 Vision

Help independent sellers manage hundreds of customer orders without losing control over products, money, purchasing, and customer relationships.

Pikko connects: product discovery → order collection → purchasing → availability → financial reconciliation → customer communication → packing → shipment → team collaboration.

**Not** merely an online store builder.

## 1.3 Pillars

### Sell Better
Interactive PDF catalogs; preorder shopping; live shopping; customer checkout; admin-assisted ordering.

### Operate Smarter
Consolidated purchasing; vendor coordination; availability management; order-item allocation; financial reconciliation; packing/shipment; collaborative team ops.

### Build Trust
Personal customer portal; transparent updates; payment history; refunds; store credit; shipment tracking.

## 1.4 Differentiation

| Driver | Capability |
| --- | --- |
| Acquisition | Interactive PDF catalog |
| Retention | Purchasing, financial reconciliation, packing |
| Trust | Customer portal |

## 1.5 Principles

Make complex ops feel simple; prioritize real merchant workflows; reduce spreadsheet work; minimize re-entry; transparent customer communication; preserve financial/operational history; collaborative operations; mobile + desktop; avoid unnecessary enterprise complexity.

---

# 2. TARGET USERS

## 2.1 Primary merchants (jastipers)

Collect orders via WhatsApp/Instagram/communities; use brand PDF catalogs; manage hundreds of order items; run preorder + live shopping; collect deposits and final payments; buy from multiple vendors; face unavailable products; consolidate before shipping; operate with team members; currently rely on spreadsheets.

## 2.2 Merchant personas / roles

| Role ID | Role | Responsibilities |
| --- | --- | --- |
| ROLE-OWNER | Store Owner | Full access; billing/settings; archive; invite staff |
| ROLE-ADMIN | Store Admin | Ops + config except ownership transfer |
| ROLE-CS | Customer Service | Orders, customers, payments verification, messaging |
| ROLE-CATALOG | Catalog Staff | Products, PDF import, flipbook, collections |
| ROLE-PURCH | Purchasing Staff | Purchasing lists, allocations, vendors, receiving |
| ROLE-PACK | Packing Staff | Packing queues, shipments, labels |
| ROLE-FIN | Finance Staff | Reconciliation, refunds, store credit, reports |

**Current implementation:** `store_memberships.role` CHECK allows `owner` only. Roles above are **target**; additive migration required (Agent 03/10).

## 2.3 Customer persona

Buy from jastipers; need discovery, ordering, payment, availability visibility, refunds, shipment tracking via portal/tracking.

---

# 3. DOMAIN & INFRASTRUCTURE

## 3.1 Current domains (use until cutover approved)

| Surface | Host |
| --- | --- |
| Marketing | `pikko.site` |
| Merchant app | `app.pikko.site` |
| Storefront / portal | `shop.pikko.site` |
| API | `api.pikko.site` |

## 3.2 Future production

`pikko.id` / `app.` / `shop.` / `api.` — **only with explicit approval**. Do not migrate automatically.

## 3.3 Domain abstraction

All URLs via env: `APP_BASE_URL` / `MARKETING_URL`, `DASHBOARD_URL` / app base, `STOREFRONT_BASE_URL` / `STOREFRONT_URL`, `API_BASE_URL`. No hardcoded domains in emails, magic links, CORS, SEO, redirects.

**Status:** Implemented (Phase 7 prep). Cutover not executed.

## 3.4 Email

Brevo for delivery. App owns token generation. Async delivery, retries, logs, duplicate prevention via `email_jobs`. Never expose keys to frontend.

**IDs:** `INFRA-DOM-01` … `INFRA-EMAIL-04`

---

# 4. DESIGN SYSTEM

Primary visual reference: https://www.brex.com/ (character, not copy/assets).

| Token | Value |
| --- | --- |
| Brand Orange | `#FF3D00` |
| Hover / Pressed / Accessible | `#EB3800` / `#D63200` / `#C83200` |
| Soft / Border brand | `#FFF1EB` / `#FFD2C2` |
| Background / Subtle / Surface | `#FFFFFF` / `#F9F9FB` / `#F3F3F7` |
| Text primary / secondary / muted | `#15191E` / `#60646C` / `#8B8D98` |
| Dark / Border | `#000710` / `#E1E3E8` |
| Font | Inter (product); Plus Jakarta Sans allowed for marketing display |
| Spacing | 4–128px scale |
| Radii | 4–16px; avoid rounded-full pills |
| Icons | Lucide stroke 1.75 |

Avoid AI-generic patterns (purple gradients, cream+terracotta serif, Inter-everything without hierarchy, icon-in-circle feature grids).

Storefront: 24 curated themes; merchant branding independent of backoffice.

**IDs:** `DS-01`–`DS-20` — Agent 02 owns expansion in `DESIGN_SYSTEM.md`.

---

# 5. AUTH, ONBOARDING & MULTI-STORE

## 5.1 Auth (passwordless magic link)

| ID | Requirement | Notes |
| --- | --- | --- |
| AUTH-01 | Email magic-link only (no passwords/social) | Live |
| AUTH-02 | Request → Brevo/log → peek (GET no consume) → confirm POST consume | Live |
| AUTH-03 | Token ≥256-bit entropy; store SHA-256 hash only | Live |
| AUTH-04 | TTL 15 min; resend cooldown 60 s; rate limit email+IP | Live |
| AUTH-05 | HttpOnly session cookie; Secure in prod; SameSite configurable | Live |
| AUTH-06 | Session list / revoke / logout-all; TTL ~30 days | Live |
| AUTH-07 | Open-redirect protection; generic responses (no enumeration) | Live |
| AUTH-08 | Account created only after successful verify | Live |

## 5.2 Onboarding

| ID | Requirement | Notes |
| --- | --- | --- |
| ONB-01 | Two-step: identity (name/slug/category) + selling model | Live |
| ONB-02 | Slug validation + reserved list + availability API | Live |
| ONB-03 | Draft resume; store+owner membership one transaction | Live |
| ONB-04 | Defaults: theme signature, IDR, Asia/Jakarta, draft, checkout off | Live |

## 5.3 Multi-store

| ID | Requirement | Notes |
| --- | --- | --- |
| STORE-01 | One account → many stores via memberships | Live |
| STORE-02 | Explicit `/stores/{storeId}/…` routes; multi-tab safe | Live |
| STORE-03 | Store switcher + manage stores + archive | Live |
| STORE-04 | Every protected op: auth + membership + resource ownership | Live |
| STORE-05 | No cross-store data leakage | Live; keep regression tests |
| STORE-06 | Team roles beyond owner | **Missing** — Stage 1/2 |

---

# 6. NAVIGATION (MERCHANT APP)

Target IA:

- **Workspace:** Overview/dashboard, notifications, analytics
- **Operations:** Orders, purchasing, packing, shipments, customers, payments/finance
- **Configuration:** Catalog, campaigns/preorders, storefront, business rules, vendors, team, settings

| ID | Requirement | Status |
| --- | --- | --- |
| NAV-01 | Store-scoped sidebar with explicit store context | Partial (current nav: Overview, Storefront, Catalog, Orders, Preorders, Customers, Business Rules, Settings, Notifications, Analytics) |
| NAV-02 | Workspace / Operations / Configuration grouping | Missing |
| NAV-03 | Role-aware nav visibility | Missing (owner-only) |

---

# 7. CAMPAIGNS (PREORDER + LIVE)

## 7.1 Preorder batches (campaign type)

| ID | Requirement | Status |
| --- | --- | --- |
| CAMP-01 | Create/edit batch: window (UTC), ETA, cover, publication + operational status | Done |
| CAMP-02 | Assign products/variants; price override; quota | Done |
| CAMP-03 | Open/close by absolute timestamps; job sync | Done |
| CAMP-04 | Public listing on storefront | Done |
| CAMP-05 | Unified “campaign” abstraction spanning preorder + live | Done (Agent 04; live ops deepen in 12) |

## 7.2 Live shopping

| ID | Requirement | Status |
| --- | --- | --- |
| LIVE-01 | Live shopping session (not video stream) — timed window, quick-add SKUs | Partial (campaign shell + flags; Agent 12) |
| LIVE-02 | Merchant quick-add to customer carts/orders during live | Missing |
| LIVE-03 | Live order feed / concurrency-safe inventory | Missing |

---

# 8. PRODUCT CATALOG

| ID | Requirement | Status |
| --- | --- | --- |
| CAT-01 | Products CRUD; draft/published/archived; images; SKU; prices integer IDR | Done |
| CAT-02 | Variants via options; price override; stock | Done |
| CAT-03 | Categories + collections + featured | Done (collections create; featured via storefront) |
| CAT-04 | Brands as first-class entities | Done (Agent 04; text field retained) |
| CAT-05 | Vendors linked to products | Done foundation (Agent 04; purchasing Agent 06) |
| CAT-06 | Custom attributes | Done foundation (Agent 04) |
| CAT-07 | Historical order line snapshots immutable | Done |

---

# 9. INTERACTIVE PDF CATALOG + AI + FLIPBOOK

| ID | Requirement | Status |
| --- | --- | --- |
| PDF-01 | Upload PDF → job queue → page render → AI extract → review items → confirm draft products | Done |
| PDF-02 | CSV import + template | Done |
| PDF-03 | Flipbook pages + hotspots → `link_ref` product contract | Done |
| PDF-04 | Public catalog flipbook on storefront | Done |
| PDF-05 | Mock AI when no key; OpenAI-compat gateway | Done |

---

# 10. LIVE SHOPPING (DETAIL)

See LIVE-01–03. Agent 12. Not video infrastructure. Focus: structured rapid order capture during merchant live sessions.

---

# 11. ORDERS

| ID | Requirement | Status |
| --- | --- | --- |
| ORD-01 | Guest checkout preview/submit; idempotency; price-change 409 | Done |
| ORD-02 | Manual bank transfer; evidence upload; verify/reject | Done |
| ORD-03 | Tracking token ≥256-bit; hash only stored | Done |
| ORD-04 | Multi-dimensional statuses: order / payment / fulfillment | Done (order-level) |
| ORD-05 | Order-item level status + operations (split, adjust qty, unavailable) | Partial (qty adjust + item events; allocations on preorder) |
| ORD-06 | Admin-assisted order entry | Done |
| ORD-07 | Spreadsheet / bulk order entry | Done |
| ORD-08 | CSV order export/import | Done (order CSV import; catalog CSV preserved) |
| ORD-09 | Historical revisions / immutable change log beyond status history | Done (`order_revisions` + status history) |
| ORD-10 | Deposit + final payment schedules | Done (policies + payment kinds; schedule UX lite) |
| ORD-11 | Cart groups ready-stock vs per-batch; one group per submit | Done |

---

# 12. PURCHASING AGGREGATION + ALLOCATION

| ID | Requirement | Status |
| --- | --- | --- |
| PUR-01 | Consolidated purchasing list per batch (required vs unpaid demand) | Done |
| PUR-02 | Record purchased / unavailable quantities | Done |
| PUR-03 | Allocate purchased qty to order items; notify on unavailable | Done |
| PUR-04 | Multi-vendor purchasing splits | Done |
| PUR-05 | Cross-batch / store-wide purchasing views | Done |
| PUR-06 | Purchasing → receiving handoff | Done (foundation) |

---

# 13. UNAVAILABLE ITEMS & ADJUSTMENTS

| ID | Requirement | Status |
| --- | --- | --- |
| UNAV-01 | Mark item unavailable; adjust payable | Done (hooks + revisions + finance ledger) |
| UNAV-02 | Customer-visible availability on portal | Missing (tracking shows order, not item-level portal) |
| UNAV-03 | Automatic financial adjustment proposals | Done (impact + financial_adjustments on apply) |

---

# 14. PAYMENTS & FINANCIAL RECONCILIATION

| ID | Requirement | Status |
| --- | --- | --- |
| FIN-01 | Integer IDR only; no floats | Done |
| FIN-02 | Payment methods (bank accounts) per store | Done |
| FIN-03 | Evidence verify/reject; payment status dimension | Done |
| FIN-04 | Order payable = goods + fees − unavailable − credits | Done |
| FIN-05 | Reconciliation workspace (expected vs received vs refundable) | Done |
| FIN-06 | Scenario: Rp300k / 250k / 110k → 190k payable, 60k refund | Done (E2E-F API test) |
| FIN-07 | Idempotent payment mutations; concurrency-safe | Done |
| FIN-08 | Immutable financial event history | Done |

---

# 15. REFUNDS & STORE CREDIT

| ID | Requirement | Status |
| --- | --- | --- |
| REF-01 | Record refunds (full/partial); payment_status refunded states exist | Done |
| REF-02 | Store credit ledger per customer per store | Done (feature-flagged) |
| REF-03 | Apply store credit at checkout / admin order | Done (admin finance apply; checkout wire optional later) |
| REF-04 | Refund evidence and audit | Done (evidence_file_id + financial_events) |

---

# 16. CUSTOMER MANAGEMENT

| ID | Requirement | Status |
| --- | --- | --- |
| CRM-01 | Store-scoped customers from orders; list/search | Done |
| CRM-02 | Customer detail + notes | Done |
| CRM-03 | Addresses book | Partial (`customer_addresses` table; limited UX) |
| CRM-04 | Purchase history / lifetime value views | Partial |
| CRM-05 | Merge duplicate customers | Missing |

---

# 17. CUSTOMER PORTAL

| ID | Requirement | Status |
| --- | --- | --- |
| PORT-01 | Tracking page by token (noindex) | Done |
| PORT-02 | Authenticated customer portal (magic link) | Done |
| PORT-03 | Order history, payments, refunds, credit, shipments | Done (orders/payments/refunds/credit; shipments → Agent 09) |
| PORT-04 | Address change requests | Done |

---

# 18. ADDRESSES

| ID | Requirement | Status |
| --- | --- | --- |
| ADDR-01 | Capture shipping address at checkout | Done |
| ADDR-02 | Multiple saved addresses; default | Done |
| ADDR-03 | Post-order address change with merchant approval | Done |

---

# 19. VENDORS

| ID | Requirement | Status |
| --- | --- | --- |
| VEN-01 | Vendor entities (name, contacts, notes) | Done |
| VEN-02 | Link products/SKUs to preferred vendors | Done |
| VEN-03 | Purchasing lists grouped by vendor | Done |

---

# 20. RECEIVING

| ID | Requirement | Status |
| --- | --- | --- |
| RCV-01 | Record goods received against purchasing | Done (foundation) |
| RCV-02 | Partial receiving; discrepancy notes | Done |
| RCV-03 | Trigger availability → packing eligibility | Partial (status nudge; packing Agent 09) |

---

# 21. PACKING

| ID | Requirement | Status |
| --- | --- | --- |
| PACK-01 | Packing queue by order / customer consolidation | Missing |
| PACK-02 | Pack checklist per order items available | Missing |
| PACK-03 | Mark packed → ready to ship/pickup | Missing (manual fulfillment status only) |

---

# 22. SHIPMENTS

| ID | Requirement | Status |
| --- | --- | --- |
| SHIP-01 | Create shipment; carrier; tracking number | Missing |
| SHIP-02 | Customer notification with tracking | Partial (email on some status; no carrier tracking entity) |
| SHIP-03 | Multi-parcel / partial ship | Missing |

---

# 23. WHATSAPP TEMPLATES

| ID | Requirement | Status |
| --- | --- | --- |
| WA-V01…V08 | Vendor-facing templates | Missing (out of MVP email path; document only until Stage 2 Agent 10) |
| WA-C01…C18 | Customer-facing templates | Missing |
| WA-01 | Template registry + copy variables; deep-link to WhatsApp | Missing |

Email transactional templates exist for magic link, order, payment, unavailable — preserve.

---

# 24. TEAM RBAC

| ID | Requirement | Status |
| --- | --- | --- |
| RBAC-01 | Roles: owner, admin, CS, catalog, purchasing, packing, finance | Missing (owner only) |
| RBAC-02 | Invite by email; accept via magic link | Missing |
| RBAC-03 | Permission checks server-side per action | Missing |
| RBAC-04 | Audit log of membership changes | Partial (`audit_logs` exists) |

---

# 25. MULTI-DIMENSIONAL STATUS ARCHITECTURE

## 25.1 Order-level (current)

| Dimension | Values |
| --- | --- |
| Order | `pending_confirmation`, `confirmed`, `completed`, `cancelled` |
| Payment | `unpaid`, `awaiting_verification`, `paid`, `failed`, `refunded`, `partially_refunded` |
| Fulfillment | `unfulfilled`, `purchasing`, `awaiting_arrival`, `ready_to_ship`, `ready_for_pickup`, `shipped`, `delivered`, `collected`, `cancelled` |

## 25.2 Target extensions

| ID | Requirement | Status |
| --- | --- | --- |
| STAT-01 | Independent dimensions; history per change | Done (order-level) |
| STAT-02 | Item-level availability / allocation / packing status | Partial |
| STAT-03 | Financial settlement status separate from payment_status | Missing |
| STAT-04 | Never collapse dimensions into a single “order state” enum | Done (principle) |

---

# 26. OPERATIONAL DASHBOARD

| ID | Requirement | Status |
| --- | --- | --- |
| DASH-01 | Overview metrics from real data; never fabricate | Done |
| DASH-02 | Operational task list (confirm, verify payment, fulfill, batch deadlines) | Partial |
| DASH-03 | Setup checklist from persisted state | Done |
| DASH-04 | Purchasing / packing / finance widgets | Missing |

---

# 27. BUSINESS RULES

| ID | Requirement | Status |
| --- | --- | --- |
| RULE-01 | Handling fee / minimum order / approval | Done |
| RULE-02 | Scope precedence: batch > selling model > store | Done |
| RULE-03 | Versioned immutable snapshots on orders | Done |
| RULE-04 | Preview at checkout | Done |
| RULE-05 | Not a visual workflow builder | Done (boundary) |

---

# 28. TECHNICAL ARCHITECTURE

## 28.1 Preferred stack (implemented)

| Layer | Choice |
| --- | --- |
| Frontend | Next.js + TypeScript + Tailwind + Radix/shadcn + Lucide |
| Backend | Go modular monolith REST |
| DB | PostgreSQL |
| Storage | S3-compatible / local filesystem adapter |
| Email | Brevo |
| Jobs | DB-backed `job_queue` + `outbox_events` + in-process worker |
| Hosting | Existing VPS Docker Compose (alongside Emplicon nginx) |

## 28.2 Modular domains (packages under `services/api/internal/`)

`auth`, `account`, `stores`, `storefront`, `catalog`, `files`, `imports`, `flipbook`, `ai`, `commerce` (orders/payments/preorders/purchasing/rules/customers), `notifications`, `analytics`, `email`, `jobs`, `money`, `urls`, `storage`, `httpapi`, `config`

**Target modules (additive):** `vendors`, `receiving`, `packing`, `shipments`, `finance` (ledger/refunds/credit), `portal`, `team`, `live`, `whatsapp`

## 28.3 Core entities (current + target)

Users, sessions, stores, memberships, products, variants, collections, storefront configs, files, import jobs, catalog pages/hotspots, customers, addresses, orders, order_items, payments, evidence, reservations, preorder batches, purchasing records, allocations, business rules/versions, notifications, analytics_events, email_jobs, job_queue, outbox_events, audit_logs.

**Target:** vendors, purchase_orders, receiving_records, packing_tasks, shipments, refunds, store_credits, order_revisions, live_sessions, whatsapp_template_sends, team_invites.

## 28.4 Money & time

- Money: integer IDR (`BIGINT`); helpers in `internal/money`
- Storage timestamps: `TIMESTAMPTZ` UTC
- UI default: Asia/Jakarta

## 28.5 Outbox events

Transactional outbox for catalog import completion and commerce notifications. Extend for finance, packing, portal events without dual-write races.

## 28.6 Tenancy

Store is the tenant boundary. All business rows carry `store_id`. Storage keys: `stores/{storeId}/{purpose}/{uuid}`.

---

# 29. SECURITY & PRIVACY

| ID | Requirement | Status |
| --- | --- | --- |
| SEC-01 | Tenant isolation on every query | Done (pattern); continuous tests |
| SEC-02 | Server-side authz; never trust client store id | Done |
| SEC-03 | Hashed tokens (auth + tracking) | Done |
| SEC-04 | Private payment evidence | Done |
| SEC-05 | No secrets in frontend / git | Done (env-only) |
| SEC-06 | PDPA-aware retention policy docs | Partial |
| SEC-07 | Rate limits auth + public checkout/tracking + coarse API | Done |
| SEC-08 | Security headers (CSP, HSTS, etc.) | Done |

**Known risks to track:** membership role expansion must not widen owner-only checks incorrectly; public file endpoints must stay published-only; evidence URLs stay authz-gated; AI keys never logged.

---

# 30. ANALYTICS

| ID | Requirement | Status |
| --- | --- | --- |
| AN-01 | North star (v3): successfully fulfilled order items per active merchant store | Partial (order-level overview exists) |
| AN-02 | Activation / engagement / operational / retention metrics | Partial |
| AN-03 | `analytics_events` sink + overview API | Done (lightweight) |
| AN-04 | Never fabricate production metrics | Done |

---

# 31. SERIAL MULTI-AGENT EXECUTION PLAN

### STAGE 1 — Platform & Commerce Foundation

| Agent | Focus | Reuses |
| --- | --- | --- |
| 01 Lead Architect | Audit, docs, architecture, matrix | — |
| 02 Design System | Tokens, components, Brex alignment | `packages/design-tokens`, `components/ui` |
| 03 Auth & Multi-Store | Harden auth; prepare RBAC schema additive | Phase 1 auth/stores |
| 04 Campaign & Catalog | Campaign model; catalog gaps (brands/attrs) | Catalog + preorders |
| 05 Order Management | Item ops, admin entry, revisions foundation | Commerce orders |

### STAGE 2 — Jastiper Operations

| Agent | Focus | Reuses |
| --- | --- | --- |
| 06 Purchasing | Vendors, multi-vendor, receiving handoff | Purchasing/allocations |
| 07 Payments/Finance | Reconciliation, refunds, store credit | Payments + money |
| 08 Customer Portal | Auth portal beyond tracking | Tracking + customers |
| 09 Packing/Shipment | Packing queues, shipments | Fulfillment statuses |
| 10 Communication | WhatsApp templates + email completeness | Notifications + email |

### STAGE 3 — Experience & Production

| Agent | Focus | Reuses |
| --- | --- | --- |
| 11 PDF/AI Catalog | Harden/extend import+flipbook | Imports + flipbook |
| 12 Live Shopping & Storefront | Live sessions + storefront polish | Storefront + themes |
| 13 Analytics | North-star dashboards | Analytics service |
| 14 Security/QA | Hardening, TEST_REPORT, E2E A–N | Existing e2e |
| 15 Integration/Deploy | Runbook execution support; no auto pikko.id | Deploy scripts |

Do not skip acceptance gates.

---

# 32. E2E SCENARIOS A–N

| ID | Scenario | Target stage | Current |
| --- | --- | --- | --- |
| E2E-A | Multi-store isolation | 1 | Covered (Playwright multi-store) |
| E2E-B | PDF catalog → products → flipbook hotspot checkout | 3 | Partial (import+flipbook+checkout exist; full chain test) |
| E2E-C | Admin order entry | 1 | Missing |
| E2E-D | Purchasing aggregation | 2 | Done (API + workspace; Playwright deferred) |
| E2E-E | Partial availability allocation | 2 | Partial |
| E2E-F | Financial reconciliation (300k/250k/110k → 190k payable, 60k refund) | 2 | Done (API) |
| E2E-G | Store credit apply | 2 | Done (API, flag) |
| E2E-H | Consolidated payment | 2 | Done (API) |
| E2E-I | Packing flow | 2 | Missing |
| E2E-J | Address change | 2 | Done (API) |
| E2E-K | Live shopping session | 3 | Missing |
| E2E-L | Concurrency (stock/quota) | 1–3 | Partial tests |
| E2E-M | Portal security (no cross-customer) | 2 | Done (API IDOR) |
| E2E-N | Domain migration readiness (env-only) | 3 | Code ready; cutover owner |

---

# 33. MVP BOUNDARIES

## 33.1 Included (v3 MVP across stages)

Auth; multi-store; catalog; PDF/AI/flipbook; storefront themes; ready-stock + preorder checkout; bank transfer; tracking; preorder purchasing/allocation; business rules; notifications; basic analytics; customers; marketing site; ops modules to the depth required by E2E A–N; team RBAC; portal; packing/shipments; refunds/store credit; live shopping (non-video); WhatsApp deep-link templates.

## 33.2 Excluded (do not build without approval)

- Payment gateway integrations (QRIS/VA beyond manual transfer)
- Marketplace / multi-merchant storefront
- Native mobile apps
- Visual workflow builder
- Real-time video streaming infrastructure
- Custom domain per merchant (post-MVP)
- Advanced marketing automation / CRM campaigns
- Wholesale / B2B pricing engines
- Automatic pikko.id DNS cutover

## 33.3 Preserve from prior v4 build

Do not rewrite working auth, storefront, catalog, import, checkout, preorder, notifications, analytics, marketing, or deploy tooling. Extend additively.

---

# 34. DEFINITION OF DONE

A feature is done when:

1. Frontend + backend + DB changes shipped together
2. Authorization enforced server-side
3. Validation + error states complete
4. Responsive desktop + mobile
5. Audit/history where financial or status-changing
6. Automated tests (unit/integration and/or e2e as appropriate)
7. Docs + requirements matrix updated
8. Financial features: exact integer money, idempotency, concurrency safety, immutable history

Placeholder UI or disconnected screens ≠ done.

---

# 35. FINAL EXECUTION INSTRUCTION

Begin with **Agent 01: Lead Architect**. Audit existing repository before writing application feature code. Preserve completed work. Plan incremental delivery against this v3.0 matrix. Do not stop after a plan. Do not auto-migrate to `pikko.id`.

**Handoff:** `docs/agents/01-lead-architect-handoff.md` → Agent 02 Design System.

---

## Appendix A — Live inventory snapshot (preserve)

| Area | Live on pikko.site |
| --- | --- |
| Auth magic link + sessions | Yes |
| Multi-store (owner) | Yes |
| Storefront draft/publish + 24 themes | Yes |
| Catalog + variants + collections | Yes |
| AI PDF import + CSV + flipbook | Yes |
| Checkout + bank transfer + tracking | Yes |
| Preorder batches + rules | Yes |
| Purchasing list + allocations | Yes (API + some UI) |
| Notifications + analytics | Yes |
| Customers | Yes |
| Marketing + SEO | Yes |
| Domain-agnostic cutover prep | Yes (not executed) |
| Deploy scripts / backups | Yes |

## Appendix B — Document map

| Prior v4 doc | v3 disposition |
| --- | --- |
| `PRODUCT_REQUIREMENTS.md` | Historical (v4); superseded by this file for new work |
| `API_REFERENCE.md` | Superseded by `API_CONTRACTS.md` (keep as alias note) |
| `DEPLOYMENT.md` / `LAUNCH.md` | Evolved into `DEPLOYMENT_RUNBOOK.md` |
| `REQUIREMENTS_MATRIX.md` | Replaced with v3 ID matrix |
