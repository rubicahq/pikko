# Deployment

## Target

VPS: `ssh -p 2021 ubuntu@43.163.99.95`  
Run as separate Docker containers behind a reverse proxy. **Do not deploy from this repo task until the SSH key secret is provided.**

Intended hostnames (current domain `pikko.site`):

| Surface | Host |
| --- | --- |
| Marketing | `pikko.site` |
| Dashboard | `app.pikko.site` |
| Storefront | `shop.pikko.site` |
| API | `api.pikko.site` |

## Docker Compose

```bash
cp .env.example .env
# set BREVO_API_KEY and EMAIL_MODE=brevo for real mail; keep EMAIL_MODE=log otherwise
docker compose up --build -d
# optional nginx front-door:
docker compose --profile proxy up --build -d
```

Services: `postgres`, `migrate` (one-shot), `api`, `web`, optional `proxy`.

Dockerfiles:

- `Dockerfile.api` — Go API + migrate binary
- `Dockerfile.web` — Next.js standalone

## Local without Docker

1. PostgreSQL 16+ with databases `pikko` / `pikko_test`
2. `cp .env.example .env` and set `DATABASE_URL`
3. `make migrate`
4. Terminal A: `make api`
5. Terminal B: `cd apps/web && npm run dev`

Web proxies `/api/v1/*` to the API so session cookies stay same-origin on localhost:3000.

## Secrets

Never commit `.env` or API keys. Required production secrets:

- `DATABASE_URL`
- `BREVO_API_KEY` (when `EMAIL_MODE=brevo`)
- storage credentials when using S3

## TLS / proxy

Terminate TLS at nginx/Caddy/Traefik. Forward `X-Forwarded-For` and `X-Forwarded-Proto`. Set `AUTH_COOKIE_SECURE=true` in production.

## Migrations

```bash
MIGRATIONS_PATH=file:///database/migrations DATABASE_URL=... ./migrate up
```

Compose runs migrate before api starts.

## Rollback posture

- Keep previous image tags
- Prefer forward-fix migrations
- Restore Postgres from backup before destructive ops (requires explicit approval)
