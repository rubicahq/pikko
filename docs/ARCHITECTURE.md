# Architecture

## Overview

Modular monolith (PRD §31). Phase 4 adds preorder batches, business rules (handling fee / minimum / approval with precedence + versioning), and multi-group carts on the Phase 0–3 foundation.

```
pikko/
  apps/web              Next.js — marketing, auth, dashboard, storefront, checkout, tracking
  services/api          Go — auth, stores, storefront, catalog, commerce, files, imports, flipbook
  packages/design-tokens
  database/migrations
  docs/
```

## Runtime

| Service | Role |
| --- | --- |
| `web` | Dashboard + public `/{slug}` storefront + cart/checkout + `/track/{token}` |
| `api` | Auth, account, stores, storefront, catalog, commerce, files, workers |
| `postgres` | System of record |
| `migrate` | SQL migrations |

## Multi-tenancy

Store is the tenant. Every catalog/storefront/file row carries `store_id`. Routes are `/api/v1/stores/{storeId}/…` with auth + membership middleware. Storage keys: `stores/{storeId}/{purpose}/{uuid}{ext}`.

## Storefront draft vs published

`storefront_configs` has `channel` `draft` \| `published`. Editing draft never changes live. Publish copies draft (+ featured/banners) atomically and sets `stores.publication_status=published`.

## Catalog / flipbook readiness

Products expose stable UUIDs, store-scoped slugs, optional SKU, images, and draft/published status. Public `link_ref` is the contract for flipbook hotspots. See `docs/CATALOG_LINK_CONTRACT.md`.

## Commerce (Phase 3–4)

- Cart lines are client-persisted (store-scoped); checkout preview/submit revalidate prices and stock on the server.
- Cart groups separate ready-stock from each preorder batch; one group per checkout submit.
- Orders store immutable line snapshots; statuses: order / payment / fulfillment (separate dimensions).
- Preorder batches: open/close by absolute UTC timestamps; batch price overrides + quotas; purchasing aggregation.
- Business rules: handling fee / minimum / approval with scope precedence (batch > selling model > store) and version snapshots on orders.
- Manual bank transfer only in MVP; evidence files use purpose `payment_evidence` (private).
- Tracking tokens are ≥256-bit; only SHA-256 hashes are stored.
- `stores.checkout_enabled` flips on when payment methods + fulfillment are configured.
- Money is integer IDR with deterministic half-up helpers for percentage fees.

## Jobs

`email_jobs` table persists delivery attempts. Auth emails send immediately via `SendNow`. Transactional commerce emails are enqueued with idempotency keys and drained by the in-process worker (`DrainPending`).

Catalog PDF import uses a PostgreSQL `job_queue` + `outbox_events` with idempotency keys. The API process runs an in-process worker that claims jobs (`FOR UPDATE SKIP LOCKED`), renders PDF pages, calls the AI extractor adapter, and writes review items. Outbox rows are marked published by the same worker loop.

## AI adapter

`internal/ai.CatalogExtractor` with:
- `MockExtractor` — default when `AI_API_KEY` is empty (tests/CI)
- `OpenAICompatClient` — SumoPod OpenAI-compatible gateway (`AI_BASE_URL`, `AI_MODEL`, `AI_API_KEY`)

Keys are env-only; never logged. Empty AI responses are retried (max tokens 65536).

## URL generation

Centralized in Go `internal/urls` and web `lib/urls.ts`. No hardcoded production domains in business logic.
