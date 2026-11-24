# Development Progress

## Scope

Phase 4 Flexible Commerce: preorder batches, batch pricing/quotas, preorder-aware checkout with cart groups, business rules (handling fee, minimum, approval) with precedence + versioning, purchasing list, extended fulfillment (PRD §21–22, §27, §40). Built on Phase 0–3 (live on `pikko.site`).

## Completed

### Phase 0–3 (prior)

- Auth, multi-store, storefront, catalog, AI import, flipbook, ready-stock checkout, bank transfer, tracking, emails

### Phase 4

- Migration `000005`: preorder batches/products/variants, purchasing + allocations, business rules + versions + order rule applications, order/batch snapshots, quota reservations via `inventory_reservations.batch_id`
- Merchant APIs: preorder CRUD, product assign, purchasing, allocations, batch orders; business rules CRUD + preview
- Public APIs: list/get batches, product open batches; checkout preview/submit preorder-aware (batch price, quota, rules, approval)
- Cart groups (`pikko_cart_v3`): ready-stock vs per-batch preorder checked out separately
- Storefront: active preorders section, batch page, PDP batch selector/labels
- Merchant UI: Preorders + Business Rules (sidebar live); order confirm/reject; preorder fulfillment chain
- Rules: precedence batch > selling model > store; versioned snapshots on each order; deterministic IDR percent rounding
- Batch auto open/close via timestamp sync (checkout + worker tick)
- Emails: order needs confirmation; existing Brevo/outbox idempotency retained
- Tests: Go unit (rules/precedence/cart groups) + integration (quota, closed batch, cross-store, approval); Playwright Phase 4 flow
- Docs updated

## Not in this release (by design)

- Full CSV purchasing export polish / unavailable-item customer notification UX depth
- Variant-level batch assignment UI (API table exists)
- Customers dashboard UI (API list exists)
- Payment gateway / refunds money movement
- VPS deploy / Emplicon (out of scope for this agent)

## Production requirements

- Run migration `000005` (idempotent migrate up from empty or existing)
- No new required secrets; Brevo + existing env remain
- Optional: ensure worker process runs so batch open/close sync ticks periodically (checkout still enforces timestamps)

## Test results

### Go

`cd services/api && go test ./... -count=1 -p 1`

### Playwright

`cd apps/web && npx playwright test`

Includes Phase 4: batch → cart group → checkout → merchant confirm.

## Decisions / open questions for product

1. Brevo sender domain readiness (carry-over)
2. Paid-order cancel/refund UX when merchants need to unwind verified payments
3. Whether unavailable preorder allocations should auto-email customers in MVP or stay merchant-driven
4. Confirm merge → then separate deploy

## Known limitations

- Batch cover image upload purpose supported (`preorder_cover`); merchant UI uses defaults without cover editor yet
- Purchasing allocation UI is API-backed; list save is primary merchant path in this release
- Multi-batch product on PDP requires selecting one open batch at add-to-cart
