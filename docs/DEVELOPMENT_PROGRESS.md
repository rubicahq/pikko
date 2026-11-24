# Development Progress

## Scope

Phase 7 Official Go-to-Market **prep** (code + docs only): domain-agnostic configuration so pikko.site → pikko.id is env/nginx cutover, legacy redirect templates, launch runbook, and QA against PRD launch criteria. **No production DNS/VPS cutover executed.** Built on Phase 0–6 (`main` @ `2cfca6e+`). Also includes VPS backup script `sudo docker` fallback from `cursor/fix-backup-sudo-docker-86c4`.

## Completed

### Phase 0–6 (prior)

- Auth, multi-store, storefront, catalog, AI PDF/CSV import, flipbook, ready-stock + preorder checkout, bank transfer, tracking, business rules, notifications, analytics, marketing site, production ops

### Phase 7 (this release)

- Domain-agnostic: `APP_URL` alias, `LEGACY_BASE_DOMAIN`, `AUTH_COOKIE_DOMAIN`, email From defaults from `APP_BASE_DOMAIN`, redirect allowlist includes base+legacy, CORS defaults from surface URLs in prod/staging
- Web: `lib/site.ts` for marketing/canonical/sitemap/support/storefront host; middleware 308 legacy host map (strips `token`); marketing copy uses env domain
- Nginx: `emplicon-nginx.d-pikko.id.conf.example`, `emplicon-nginx.d-pikko.site-redirect.conf.example`
- Compose/Docker build args for `NEXT_PUBLIC_APP_BASE_DOMAIN`, legacy, support email
- `.env.production.pikko.id.example` + updated `.env.example`
- `docs/LAUNCH.md` ordered cutover + owner checklist (rotate keys, SPF, Full strict, magic-link test)
- Playwright Phase 7 domain smoke; Go URL/config tests for pikko.id
- Backup scripts: prefer `docker`, fall back to `sudo docker` (merged from fix branch)
- **No DB migration** for Phase 7

## Not in this release (by design)

- **Executing** DNS/TLS/env cutover to pikko.id (owner-only; needs explicit approval)
- Brevo sender domain authentication on pikko.id (owner)
- Invented subscription prices / counsel-final legal contracts
- Payment gateway / refunds money movement
- VPS SSH deploy by agent

## Production requirements

- No new migration
- New optional env: `APP_URL`, `LEGACY_BASE_DOMAIN`, `AUTH_COOKIE_DOMAIN`, `NEXT_PUBLIC_APP_BASE_DOMAIN`, `NEXT_PUBLIC_LEGACY_BASE_DOMAIN`, `NEXT_PUBLIC_SUPPORT_EMAIL`
- Cutover only via `docs/LAUNCH.md` after owner approval
- Current production may remain on pikko.site until that approval

## Test results

### Go

`cd services/api && go test ./... -count=1 -p 1`

### Frontend / Playwright

`cd apps/web && npx tsc --noEmit && npm run lint && npm run build && npx playwright test`

## Decisions / open questions for product

1. Explicit approval + date for pikko.id cutover
2. Brevo pikko.id authentication + SPF
3. Final commercial pricing and counsel legal pages
4. Confirm VPS pull of main (includes backup sudo docker fix)

## Known limitations

- Pricing remains early-access framing until business decision
- Legal pages are working notices until counsel replaces them
- Legacy magic-link emails pointing at pikko.site require a new link after redirect (tokens not forwarded)
