# Development Progress

## Scope

Phase 0 Foundation + Phase 1 Auth & Multi-Store Onboarding (PRD §40, §47).

## Completed

- Initial `main` commit with `docs/PRODUCT_REQUIREMENTS.md` + README
- Monorepo, design tokens, UI primitives, Inter, Lucide
- Go API: auth, account, stores, onboarding, middleware, email jobs
- Next.js: marketing home, auth screens A–D, onboarding, manage stores, switcher, account settings, store overview
- Dockerfiles + Compose + nginx sample
- Docs set (9 files)
- Unit, integration, and Playwright suites for Phase 0/1

## Not in this release (by design)

- Catalog, storefront editor, checkout, orders, preorders, payments, AI import, marketing solution pages, pricing content
- Real Brevo production sender verification / DNS
- VPS deploy (waiting on SSH key)

## Test results

### Go unit + integration (local Postgres `pikko_test`)

Ran `cd services/api && go test ./... -count=1` (Asia/Jakarta):

```
ok  internal/auth     (peek/consume, concurrent verify, session revoke)
ok  internal/stores   (create, cross-store 404, onboarding resume)
ok  internal/urls
ok  internal/validate
```

### Playwright

Ran `cd apps/web && npx playwright test` against local API + web (Asia/Jakarta):

```
13 passed (11.2s)
- auth → onboarding → first store
- GET peek does not consume; POST confirm consumes
- additional store + manage stores
- responsive: 375, 390, 768, 1024, 1440 (home + auth, no horizontal overflow)
```

### Responsive

Verified via Playwright viewport suite on marketing home and auth at 375 / 390 / 768 / 1024 / 1440.

## Decisions / open questions for product

1. Confirm Brevo sender domains (`account@pikko.site`) are verified before enabling `EMAIL_MODE=brevo` on the VPS.
2. Confirm SSH key secret name for deploy automation.
3. Store limit policy remains configurable / unlimited until subscription pricing is decided (§46).
4. Cookie domain strategy for `app.` vs `api.` in production — currently web same-origin proxies `/api/v1` so cookies stay on the dashboard host.

## Known limitations

- Phase 2 nav destinations show an honest “not built yet” panel (not fake commerce UI).
- Component preview blocked in production; available in development only.
- S3 storage driver is stubbed; local filesystem used by default.
- API healthcheck in Compose is process-level only (distroless image has no curl).
