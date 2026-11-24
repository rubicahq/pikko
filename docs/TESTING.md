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
- Unit (Phase 4): rule precedence, percent rounding, cart group isolation
- Integration (Phase 4): batch pricing/quota, closed-batch reject, approval + rule snapshots, cross-store batch isolation, concurrent oversell protection

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
- Phase 4: preorder batch → cart group → checkout → merchant confirm
- Phase 5: notifications / analytics / CSV / customers smoke
- Phase 6: marketing pages, FAQ keyboard, robots/sitemap, security headers

## Ops scripts

```bash
./infrastructure/ops/dependency-audit.sh
./infrastructure/ops/load-smoke.sh https://pikko.site https://api.pikko.site
./infrastructure/backup/restore-test.sh ~/backups/pikko/<stamp>/postgres.dump
```

## Manual checks

- Upload logo/favicon on Settings; publish; confirm favicon on public page
- Draft product not visible on public storefront
- Unauthorized store id returns 404 (not 403)
- Checkout disabled until payment + fulfillment configured
- Tracking page shows no internal notes / cost prices
- Live AI / Brevo only when keys set in env (never commit secrets)
