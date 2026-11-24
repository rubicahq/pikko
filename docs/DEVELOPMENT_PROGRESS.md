# Development Progress

## Scope

Phase 3 First Transaction: cart, guest checkout, ready-stock orders, manual bank transfer, evidence, verification, order management, tracking, essential emails (PRD §20–26, §28, §40). Built on Phase 0–2 + AI import + flipbook (merged on `main`).

## Completed

### Phase 0–2 + accelerators (prior)

- Auth, multi-store, storefront, catalog, AI PDF import, flipbook

### Phase 3

- Migration `000004`: payment methods, fulfillment settings, customers, orders/items/history, tracking tokens (hash-only), payments, evidence, inventory reservations, public rate limits, order counters
- Real cart module (`apps/web/src/lib/cart.ts`) — store-scoped localStorage lines; server revalidates on preview/submit
- Public checkout preview/submit with integer IDR totals, price-change guard, idempotency
- Manual bank transfer + private evidence upload (`payment_evidence`) with tracking/merchant file access
- Merchant payment verify/reject; order list/detail + fulfillment transitions
- Public tracking page `/track/{token}` (noindex); token ≥256-bit, hash stored only
- Transactional emails via Brevo adapter + `email_jobs` drain (order confirmation customer, new order merchant; payment verified/rejected)
- Customers auto-created store-scoped from checkout
- Stock reservations for finite inventory under transaction
- Checkout enabled only when ≥1 bank account and ≥1 fulfillment method configured
- Go unit/integration tests; Playwright Phase 3 buy → evidence → verify
- Docs updated

## Not in this release (by design)

- Preorder batches / multi-group carts / handling fees / approval rules (Phase 4)
- Gateway payments, refunds money movement, deposits
- In-app notification center
- VPS deploy / Emplicon (separate after merge)
- Live Brevo domain verification

## Test results

### Go

`cd services/api && go test ./... -count=1 -p 1`

Includes commerce checkout + payment verification + cross-store isolation; money rounding; prior suites.

### Playwright

`cd apps/web && npx playwright test`

Phase 3: configure bank/fulfillment → publish product/storefront → cart → checkout → upload proof → merchant verify.

## Decisions / open questions for product

1. Brevo sender domain readiness (carry-over)
2. Whether payment evidence signed URLs should use time-limited HMAC in addition to tracking-token auth (MVP uses tracking-scoped + merchant-auth file routes)
3. Paid-order cancellation / refund UX when merchants need to unwind verified payments
4. SSH key / VPS deploy (out of scope here)

## Known limitations

- Cart lines persist client-side; inventory is not reserved until order submit
- Handling fees / minimum order / approval rules deferred to Phase 4
- S3 signed URLs still stubbed; private evidence served via authorized API routes
- Customer list UI is API-ready; dedicated customers dashboard page still later-phase polish
