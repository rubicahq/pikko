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
| AI PDF/CSV + flipbook | Agent 11 hardens |
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
| **08 Customer Portal** | Auth portal beyond tracking | PORT-01, CRM | PORT-02–04; E2E-M |
| **09 Packing/Shipment** | Pack queues, shipments | Fulfillment statuses | PACK/SHIP; E2E-I |
| **10 Communication** | WhatsApp templates + email completeness; RBAC invites polish | notifications/email | WA registry; team invites usable |

### Stage 3 — Experience & Production Readiness

| Agent | Scope | Reuses | Exit gate |
| --- | --- | --- | --- |
| **11 PDF/AI Catalog** | Reliability, limits, review UX | PDF-01–05 | Hardened import; no schema destruction |
| **12 Live Shopping & Storefront** | Live sessions + storefront polish | storefront | LIVE-*; E2E-K |
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

## 4. Open decisions (non-blocking for Agent 04)

1. Customer portal auth model: separate customer identities vs linked users (Agent 08 decides with 01 note preferring separate).
2. Campaign table vs facade over `preorder_batches` + `live_sessions` (Agent 04).
3. Commercial pricing / counsel legal — owner (unchanged).
4. pikko.id cutover date — owner only.

---

## 5. Historical v4 progress (reference)

Phases 0–7 completed on pikko.site: foundation, auth, storefront/catalog/import, checkout, preorder/rules, merchant efficiency, marketing/ops, domain cutover **prep**. See prior `PRODUCT_REQUIREMENTS.md` and phase handoffs in project store `internal/phase-*-handoff.md`.
