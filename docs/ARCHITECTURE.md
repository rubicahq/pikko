# Architecture

## Overview

Modular monolith (PRD §31). Phase 0/1 delivers foundation + passwordless auth + multi-store onboarding.

```
pikko/
  apps/web              Next.js (App Router) + Tailwind + Radix/shadcn-style UI
  services/api          Go HTTP API (stdlib ServeMux)
  packages/design-tokens
  packages/ui           Pointer to web UI primitives
  database/migrations   golang-migrate SQL
  infrastructure/nginx  Sample reverse proxy
  docs/
  tests/e2e             Playwright (also apps/web/e2e)
```

## Runtime

| Service | Role |
| --- | --- |
| `web` | Marketing, auth UI, onboarding, dashboard shell |
| `api` | Auth, account, stores, onboarding, authorization |
| `postgres` | System of record |
| `migrate` | One-shot migration job in Compose |
| `proxy` (optional) | nginx in front of web + api |

Email delivery uses a Brevo adapter. `EMAIL_MODE=log` (default) writes messages to logs — safe for local/CI. `EMAIL_MODE=brevo` requires `BREVO_API_KEY`.

## URL generation

Centralized in:

- Go: `services/api/internal/urls`
- Web: `apps/web/src/lib/urls.ts`

Business logic stores slugs/IDs/relative paths only. Hosts come from env (`MARKETING_URL`, `DASHBOARD_URL`, `STOREFRONT_URL`, `API_BASE_URL`).

## Auth model

1. `POST /api/v1/auth/magic-link/request` — hash stored, email sent
2. `GET /api/v1/auth/magic-link/peek` — validates without consuming
3. User confirms via UI (`POST .../verify`) — atomic consume + session
4. Session cookie `pikko_session` (HttpOnly); Bearer also accepted for SSR/tests

## Multi-tenancy

Store is the tenant. Protected store routes are always `/api/v1/stores/{store-id}/...` with middleware that checks membership. Dashboard routes mirror that with `/stores/{store-id}/...` so tabs keep independent store context.

## Storage

`STORAGE_DRIVER=local` writes under `STORAGE_LOCAL_PATH`. S3 driver stubbed for later.

## Jobs

`email_jobs` table persists delivery attempts. Auth emails send immediately via `SendNow`. A dedicated worker process can be added when async volume requires it (Compose-ready).
