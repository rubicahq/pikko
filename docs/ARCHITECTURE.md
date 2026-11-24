# Architecture

## Overview

Modular monolith (PRD §31). Phase 2 adds storefront + catalog on the Phase 0/1 foundation.

```
pikko/
  apps/web              Next.js — marketing, auth, dashboard, public storefront
  services/api          Go — auth, stores, storefront, catalog, files
  packages/design-tokens
  database/migrations
  docs/
```

## Runtime

| Service | Role |
| --- | --- |
| `web` | Dashboard + public `/{slug}` storefront |
| `api` | Auth, account, stores, storefront, catalog, files |
| `postgres` | System of record |
| `migrate` | SQL migrations |

## Multi-tenancy

Store is the tenant. Every catalog/storefront/file row carries `store_id`. Routes are `/api/v1/stores/{storeId}/…` with auth + membership middleware. Storage keys: `stores/{storeId}/{purpose}/{uuid}{ext}`.

## Storefront draft vs published

`storefront_configs` has `channel` `draft` \| `published`. Editing draft never changes live. Publish copies draft (+ featured/banners) atomically and sets `stores.publication_status=published`.

## Catalog / flipbook readiness

Products expose stable UUIDs, store-scoped slugs, optional SKU, images, and draft/published status. Public `link_ref` is the contract for future flipbook hotspots. See `docs/CATALOG_LINK_CONTRACT.md`.

## URL generation

Centralized in Go `internal/urls` and web `lib/urls.ts`. No hardcoded production domains in business logic.
