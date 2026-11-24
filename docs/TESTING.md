# Testing

## Unit (Go)

```bash
cd services/api && go test ./internal/validate ./internal/urls ./internal/auth -count=1
```

Covers token entropy/hashing, slug/email validation, URL generation, open-redirect allowlist.

## Integration (Go + Postgres)

Requires `pikko_test` database migrated:

```bash
MIGRATIONS_PATH=file:///workspace/database/migrations \
  DATABASE_URL=postgres://pikko:pikko@localhost:5432/pikko_test?sslmode=disable \
  go run ./services/api/cmd/migrate up

cd services/api && go test ./internal/auth ./internal/stores -count=1
```

Covers:

- Magic link peek does not consume; verify consumes once
- Concurrent verify → single account
- Session revocation
- Store create + ownership
- Cross-store HTTP access denied (404)
- Onboarding resume + idempotent create

## E2E (Playwright)

```bash
# API + web must be running
cd apps/web && npx playwright test
```

Uses `EMAIL_MODE=log` `dev_token` path — no real Brevo calls, no production auth bypass.

Scenarios:

- Marketing home loads
- Auth → confirm → onboarding → create store → overview
- Additional store + switcher navigation
- Responsive checks at 375, 390, 768, 1024, 1440

## Visual / responsive

Playwright viewport suite in `apps/web/e2e/responsive.spec.ts`.
