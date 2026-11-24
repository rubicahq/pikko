# Architecture

## Overview

Modular monolith (PRD §31). Phase 2 adds storefront + catalog on the Phase 0/1 foundation. AI PDF import and flipbook build on Phase 2 products.

```
pikko/
  apps/web              Next.js — marketing, auth, dashboard, public storefront
  services/api          Go — auth, stores, storefront, catalog, files, imports, flipbook
  packages/design-tokens
  database/migrations
  docs/
```

## Runtime

| Service | Role |
| --- | --- |
| `web` | Dashboard + public `/{slug}` storefront |
| `api` | Auth, account, stores, storefront, catalog, files, import worker |
| `postgres` | System of record |
| `migrate` | SQL migrations |

## Multi-tenancy

Store is the tenant. Every catalog/storefront/file row carries `store_id`. Routes are `/api/v1/stores/{storeId}/…` with auth + membership middleware. Storage keys: `stores/{storeId}/{purpose}/{uuid}{ext}`.

## Storefront draft vs published

`storefront_configs` has `channel` `draft` \| `published`. Editing draft never changes live. Publish copies draft (+ featured/banners) atomically and sets `stores.publication_status=published`.

## Catalog / flipbook readiness

Products expose stable UUIDs, store-scoped slugs, optional SKU, images, and draft/published status. Public `link_ref` is the contract for flipbook hotspots. See `docs/CATALOG_LINK_CONTRACT.md`.

## Jobs

`email_jobs` table persists delivery attempts. Auth emails send immediately via `SendNow`.

Catalog PDF import uses a PostgreSQL `job_queue` + `outbox_events` with idempotency keys. The API process runs an in-process worker that claims jobs (`FOR UPDATE SKIP LOCKED`), renders PDF pages, calls the AI extractor adapter, and writes review items. Outbox rows are marked published by the same worker loop.

## AI adapter

`internal/ai.CatalogExtractor` with:
- `MockExtractor` — default when `AI_API_KEY` is empty (tests/CI)
- `OpenAICompatClient` — SumoPod OpenAI-compatible gateway (`AI_BASE_URL`, `AI_MODEL`, `AI_API_KEY`)

Keys are env-only; never logged. Empty AI responses are retried (max tokens 65536).

## URL generation

Centralized in Go `internal/urls` and web `lib/urls.ts`. No hardcoded production domains in business logic.
