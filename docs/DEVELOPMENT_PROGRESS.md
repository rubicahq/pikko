# Development Progress

## Scope

Phase 6 Marketing and Production Readiness: final marketing site for pikko.site (landing, solutions, pricing, FAQ, legal), SEO (sitemap, robots, OG), accessibility polish, and production ops (health, logging, rate limits, security headers, scheduled backups, runbook, light load smoke, dependency audit). Built on Phase 0–5 (`main` @ `b963682+`).

## Completed

### Phase 0–5 (prior)

- Auth, multi-store, storefront, catalog, AI PDF/CSV import, flipbook, ready-stock + preorder checkout, bank transfer, tracking, business rules, notifications, analytics

### Phase 6

- Marketing routes: `/`, `/pricing`, `/faq`, `/privacy`, `/terms`, `/solutions`, `/solutions/jastip`, `/solutions/social-sellers`
- Marketing layout (header/footer), Plus Jakarta Sans display + Inter body, design tokens only
- SEO: `sitemap.xml`, `robots.txt`, Open Graph image, per-page canonicals; auth/dashboard remain noindex
- API: `/livez`, `/healthz`, `/readyz`; structured access logs; coarse IP rate limit; hardened security headers
- Ops: `infrastructure/backup/*`, `infrastructure/ops/load-smoke.sh`, `dependency-audit.sh`, `docs/RUNBOOK.md`
- Nginx example edge headers; Playwright Phase 6 smoke; docs matrix updated
- **No DB migration** in Phase 6

## Not in this release (by design)

- Phase 7 pikko.id migration / commercial GTM (needs explicit approval)
- Invented subscription prices (open business decision — pilot framing only)
- Payment gateway / refunds money movement
- VPS deploy execution (documented for operator; agent does not SSH)

## Production requirements

- No new migration
- Optional new env: `API_RATE_LIMIT_IP_PER_HOUR` (default 1200)
- Install backup cron from `infrastructure/backup/crontab.example`
- Reload nginx if adopting updated `pikko.site.conf` header snippets
- Set `NEXT_PUBLIC_APP_ENV=production` so marketing is indexable

## Test results

### Go

`cd services/api && go test ./... -count=1 -p 1`

### Frontend / Playwright

`cd apps/web && npx tsc --noEmit && npm run lint && npm run build && npx playwright test`

## Decisions / open questions for product

1. Final commercial pricing (carry-over; legal pages are working notices)
2. Confirm VPS: pull main, install backup cron, optional nginx header reload
3. Brevo sender domain readiness (carry-over)

## Known limitations

- Pricing page describes pilot access — not a paid plan catalog
- Legal pages are operational notices for pilots, not counsel-approved launch contracts
- Light load smoke is a probe script, not a full capacity benchmark
