# Architecture — Pikko Modular Monolith (PRD v3.0)

**Owner:** Agent 01 (Lead Architect)  
**Baseline:** `main` @ `353498a`, migrations `000001`–`000006`  
**Rule:** Additive evolution only. No greenfield rewrite. No destructive production migrations.

---

## 1. Overview

```
pikko/
  apps/web                 Next.js — marketing, auth, dashboard, storefront, checkout, tracking
  services/api             Go modular monolith — REST /api/v1
  packages/design-tokens   CSS + Tailwind preset
  packages/ui              Shared package stub (primary UI lives in apps/web/src/components/ui)
  packages/shared-types
  database/migrations      Numbered SQL up/down
  infrastructure/          nginx examples, backup, ops
  scripts/                 deploy.sh, smoke.sh
  docs/                    PRD + architecture + contracts
```

| Runtime | Role |
| --- | --- |
| `web` | All browser surfaces (host-routed via nginx) |
| `api` | Business logic + in-process job worker |
| `postgres` | System of record (`pikko-postgres` on VPS) |
| `migrate` | One-shot migration job |

---

## 2. Modular monolith boundaries

Packages under `services/api/internal/` are **logical modules**, not separate deployables. Cross-module calls are in-process Go imports. HTTP is the only external boundary.

### 2.1 Current modules (preserve)

| Module | Responsibility |
| --- | --- |
| `auth` / `account` | Magic link, sessions, account profile |
| `stores` | Store CRUD, onboarding, membership (owner) |
| `storefront` | Draft/published branding, themes, featured |
| `catalog` | Products, variants, categories, collections |
| `files` / `storage` | Store-scoped uploads |
| `imports` / `ai` / `pdfutil` / `flipbook` | PDF/CSV import + flipbook |
| `commerce` | Checkout, orders, payments, preorders, purchasing, rules, customers |
| `notifications` / `analytics` | In-app notifications + overview metrics |
| `email` / `jobs` | Email jobs + job_queue + outbox worker |
| `money` / `urls` / `config` / `httpapi` | Cross-cutting |

### 2.2 Target modules (additive; do not invent microservices)

| Module | Introduced by | Notes |
| --- | --- | --- |
| `team` | Agent 03 (done) / 10 polish | RBAC roles, invites |
| `campaigns` | Agent 04 (done) | Unify preorder + live under `campaigns`; legacy `/preorders` kept + synced |
| `vendors` | Agent 04 foundation / 06 expand | Vendor master + product links in catalog; purchasing Agent 06 |
| `receiving` | Agent 06 | Goods receipt |
| `finance` | Agent 07 | Ledger, refunds, store credit, reconciliation |
| `portal` | Agent 08 | Customer-authenticated portal |
| `packing` / `shipments` | Agent 09 | Pack queues + carrier tracking |
| `whatsapp` | Agent 10 | Template registry + deep links |
| `live` | Agent 12 | Live shopping sessions |

**Integration rule:** New modules own tables + service methods; `httpapi` registers routes; shared types stay thin. Prefer extending `commerce` only when the concept is already inseparable (e.g. order item allocation).

---

## 3. Tenancy

- **Tenant = store.** Every business row includes `store_id`.
- Routes: `/api/v1/stores/{storeID}/…` with `AuthMiddleware` + `StoreAuthMiddleware`.
- Public surfaces use store **slug**, then resolve to `store_id` server-side; still filter by store.
- Storage keys: `stores/{storeId}/{purpose}/{uuid}{ext}`.
- **Never** trust client-supplied store id without membership check.
- Multi-tab: store id is in the URL path, not a single global cookie.

### 3.1 Membership evolution (Agent 03)

`000007` widens `store_memberships.role` to owner/admin/cs/catalog/purchasing/packing/finance. Existing rows remain `owner`. Permission matrix lives in `internal/rbac`; `StoreAuthMiddleware.RequirePermission` enforces server-side. Invites: `team_invites` + Brevo magic-link accept.

---

## 4. Money

| Rule | Implementation |
| --- | --- |
| Currency | IDR only in MVP |
| Representation | `BIGINT` integer rupiah (not cents-of-cents) |
| No floats | Enforced in Go (`internal/money`) and SQL CHECKs `>= 0` |
| Fees | Deterministic half-up percentage helpers (`money.Round`) |
| Display | `money.FormatIDR` / UI thousand separators; timezone Asia/Jakarta for dates |

Financial features (Agent 07) must add an **append-only ledger** (or event table) for refunds/credits; never rewrite historical `order_items.unit_price_idr` snapshots.

---

## 5. Outbox & jobs

| Mechanism | Use |
| --- | --- |
| `email_jobs` | Auth SendNow + commerce DrainPending with idempotency keys |
| `job_queue` | Long work: PDF import, preorder batch status sync (`FOR UPDATE SKIP LOCKED`) |
| `outbox_events` | Transactional side-effects after commits |

**Target events (document for later agents):** `order.placed`, `payment.verified`, `item.unavailable`, `allocation.updated`, `refund.issued`, `credit.applied`, `shipment.created`, `packing.completed`.

Worker runs in-process inside `api` (see `httpapi.NewRouter`). Do not introduce Redis/Kafka for MVP.

---

## 6. Status dimensions

Statuses are **independent dimensions**, never collapsed into one enum.

### 6.1 Order-level (live)

```
order_status        pending_confirmation | confirmed | completed | cancelled
payment_status      unpaid | awaiting_verification | paid | failed | refunded | partially_refunded
fulfillment_status  unfulfilled | purchasing | awaiting_arrival | ready_to_ship |
                    ready_for_pickup | shipped | delivered | collected | cancelled
```

History: `order_status_history(dimension, from_status, to_status, actor, reason)`.

### 6.2 Target item-level (Agents 05–09)

Documented for schema planning (additive columns/tables):

| Dimension | Example values |
| --- | --- |
| Item availability | available / partially_unavailable / unavailable |
| Item allocation | unallocated / partial / allocated |
| Item packing | unpacked / packed |
| Item settlement | open / adjusted / settled |

Order-level fulfillment may aggregate from items but must not erase item history.

### 6.3 Publication vs checkout vs operational

Keep separate:

- Store `publication_status` (draft/published)
- `checkout_enabled` (payments + fulfillment configured)
- Batch publication vs operational open/closed
- Financial settlement (future) vs `payment_status`

---

## 7. Frontend surfaces

| Host | App routes (conceptual) |
| --- | --- |
| `pikko.site` | Marketing `(marketing)/*` |
| `app.pikko.site` | Auth, onboarding, `/stores/{id}/*` dashboard |
| `shop.pikko.site` | `/{storeSlug}/*` storefront, cart, checkout; future portal |
| Shared | `/track/{token}`, `/api/v1/*` proxy |

Web proxies API via `apps/web/src/app/api/v1/[...path]` so session cookies can be host-friendly.

---

## 8. Security architecture

1. AuthN: session cookie validated on API
2. AuthZ: membership (+ future RBAC permissions)
3. Public endpoints: rate-limited; no private evidence without token/membership
4. Secrets: env only (Brevo, AI, DB, storage)
5. Headers: CSP, HSTS, nosniff, frame deny
6. CORS: allowlist from surface URLs (+ legacy domain when set)

---

## 9. Deployment topology

VPS Docker Compose project `pikko` beside Emplicon. Emplicon nginx terminates TLS and routes hosts. See `DEPLOYMENT_RUNBOOK.md`.

Domains today: `*.pikko.site`. `pikko.id` cutover is **manual owner action**, not automatic.

---

## 10. Acceptance gate (Agent 01)

This architecture documents support for:

- Multi-store tenancy — **implemented**
- Order-item operations — **partial; qty adjust + availability columns (Agent 05); split/unavailable finance later**
- Purchasing allocations — **implemented for preorder batches**
- Historical revisions — **status history + `order_revisions` / `order_item_events` (Agent 05)**
- Financial reconciliation — **payments live; ledger/refund/credit planned for Agent 07**

No application feature code in Agent 01 scope.
