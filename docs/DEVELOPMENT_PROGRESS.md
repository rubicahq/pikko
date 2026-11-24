# Development Progress

## Scope

Phase 5 Merchant Efficiency: in-app notifications, transactional email polish (including unavailable allocation), merchant analytics (Asia/Jakarta), CSV product import, customers dashboard, plus Phase 4 leftovers (batch cover editor + variant pairing UI). AI PDF import was already complete and not re-implemented. Built on Phase 0–4 (`main` @ `01ff5d8+`).

## Completed

### Phase 0–4 (prior)

- Auth, multi-store, storefront, catalog, AI PDF import, flipbook, ready-stock + preorder checkout, bank transfer, tracking, business rules, emails

### Phase 5

- Migration `000006`: `notifications`, `notification_preferences`, `analytics_events`; CSV support on `catalog_import_jobs` (`source_type`, `csv_file_id`); `catalog_csv` file purpose
- In-app notifications API + UI (list, unread badge, mark read / mark all) wired from new order, payment evidence, batch close, purchasing issues
- Customer email on unavailable allocation via existing `email_jobs` / Brevo drain; batch-closed merchant email
- Analytics overview API + UI (sales, orders, top products, conversion) with Asia/Jakarta day boundaries, store-scoped queries
- CSV catalog import (template download, validation, duplicate SKU warnings, review → draft products)
- Customers list/detail + internal notes editor
- Preorder batch cover upload/clear UI; variant pairing UI on batch products
- Tests: Go `./...`; Playwright Phase 5 smoke; docs updated

## Not in this release (by design)

- Payment gateway / refunds money movement
- Marketing site / Phase 6 production readiness
- VPS deploy / Emplicon (out of scope for this agent)
- Visual charts library (tables only for analytics MVP)

## Production requirements

- Run migration `000006` (idempotent migrate up from empty or existing)
- No new required secrets; Brevo + existing env remain (`EMAIL_MODE`, `BREVO_API_KEY`, `EMAIL_NOTIFICATION_FROM`)
- Worker process should keep draining `email_jobs` and batch sync

## Test results

### Go

`cd services/api && go test ./... -count=1 -p 1`

### Frontend / Playwright

`cd apps/web && npx tsc --noEmit && npm run lint && npm run build && npx playwright test`

## Decisions / open questions for product

1. Brevo sender domain readiness (carry-over)
2. Paid-order cancel/refund UX (carry-over)
3. Confirm merge → then separate deploy / migrate `000006` on production

## Known limitations

- Unavailable-item email mints a fresh tracking token when the original raw token is unavailable (hash-only storage)
- Analytics conversion = paid orders / total non-cancelled orders in range (not storefront visit conversion)
- CSV import does not create variants from columns yet (flat products)
