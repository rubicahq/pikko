# Development Progress

## Scope

Phase 2 Beautiful Storefront: store identity/branding, themes, catalog, public storefront (PRD §16–17, §40). Built on Phase 0/1 (PR #1).

## Completed

### Phase 0/1 (prior)

- Monorepo, design tokens, auth, multi-store onboarding, docs, Docker foundation

### Phase 2

- Migration `000002`: `stored_files`, `storefront_configs` (draft/published), banners/featured scaffolding, categories, products, images, options/variants, collections
- Store-scoped object keys + upload validation (logo/favicon/store icon/product)
- 24 curated themes with PRD hex values + accessible button foreground
- Merchant APIs: storefront draft/patch/publish/discard, files, categories, products, collections, setup checklist
- Public APIs: store by slug, products, categories, public files
- `link_ref` on public product responses for flipbook hotspot linking (`docs/CATALOG_LINK_CONTRACT.md`)
- Merchant UI: Settings (identity + assets), Storefront editor + draft preview, Catalog list/categories/product editor
- Public storefront pages `/{slug}` and `/{slug}/products/{productSlug}` with basic SEO (title, description, canonical, favicon)
- Overview checklist derived from persisted state
- Unit/integration tests; Playwright Phase 2 flows + cross-store isolation

## Not in this release (by design)

- Flipbook catalog pages / hotspots (follow-on worker)
- AI PDF import (moved earlier in roadmap; follow-on worker)
- Banners UI, featured collections UI polish beyond API scaffolding
- Checkout, orders, payments, preorders (Phase 3+)
- VPS deploy / Emplicon
- Real Brevo domain verification

## Test results

### Go

`cd services/api && go test ./... -count=1 -p 1`

Includes themes (24 hex), money/IDR, catalog CRUD + cross-store 404, stores, auth.

### Playwright

`cd apps/web && npx playwright test`

Phase 1 suites retained; Phase 2: product create → publish storefront → public browse; API cross-store isolation.

## Decisions / open questions for product

1. Brevo sender domain readiness (carry-over)
2. SSH key secret name for deploy (carry-over)
3. Production cookie/host strategy (carry-over)
4. Flipbook page model details (owned by follow-on worker; product link contract frozen in `CATALOG_LINK_CONTRACT.md`)

## Known limitations

- S3 driver still stubbed; local filesystem default with store-scoped keys ready for S3
- Banner/featured editor UI is API-ready but not a full visual merchandising board yet
- Checkout CTA on public PDP notes Phase 3
