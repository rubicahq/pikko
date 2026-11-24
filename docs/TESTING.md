# Testing

## Go

```bash
export TEST_DATABASE_URL=postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable
cd services/api && go test ./... -count=1 -p 1
```

Use `-p 1` so packages sharing `pikko_test` do not truncate concurrently.

Coverage highlights:

- Unit: themes (24 PRD hex), money/IDR + half-up rounding, slug validation, URLs, PDF page counting, mock AI extractor
- Integration: auth, store create, catalog CRUD, cross-store 404 on products, storefront publish → public product + `link_ref`
- Integration: PDF import (mock AI); flipbook publish + public hotspots
- Integration (Phase 3): checkout → evidence → verify; idempotent submit; tracking token hash-only; cross-store order isolation; email jobs enqueued

## Playwright

```bash
# API on :8080, web on :3000, DATABASE_URL set, migrations applied
cd apps/web && npx playwright test
```

Suites:

- Phase 1 auth/onboarding/multi-store/responsive
- Phase 2 product + publish + public storefront
- Phase 2 API cross-store isolation (separate request contexts)
- AI PDF catalog import (mock extractor)
- Flipbook editor smoke
- Phase 3: bank/fulfillment setup → cart → checkout → evidence → merchant verify

## Manual checks

- Upload logo/favicon on Settings; publish; confirm favicon on public page
- Draft product not visible on public storefront
- Unauthorized store id returns 404 (not 403)
- Checkout disabled until payment + fulfillment configured
- Tracking page shows no internal notes / cost prices
- Live AI / Brevo only when keys set in env (never commit secrets)
