# Pikko

Simple commerce platform for independent sellers. **Sell your way.**

## Stack

- **Web:** Next.js, TypeScript, Tailwind, Radix/shadcn-style UI, Lucide, Inter
- **API:** Go, PostgreSQL, golang-migrate
- **Email:** Brevo adapter (`EMAIL_MODE=log` for local/CI; `brevo` for production)
- **Deploy:** Docker Compose on VPS behind a reverse proxy

## Quick start (local)

```bash
cp .env.example .env
# ensure Postgres is running and DATABASE_URL is set
make migrate
make api          # :8080
cd apps/web && npm install && npm run dev   # :3000
```

Web proxies `/api/v1/*` to the API so session cookies stay same-origin.

## Docker

```bash
docker compose up --build
# optional nginx front-door:
docker compose --profile proxy up --build
```

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md). VPS deploy is documented there but not executed until an SSH key is provided.

## Docs

| Doc | Purpose |
| --- | --- |
| [PRODUCT_REQUIREMENTS.md](docs/PRODUCT_REQUIREMENTS.md) | Source of truth PRD |
| [DESIGN_SYSTEM.md](docs/DESIGN_SYSTEM.md) | Tokens & UI |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design |
| [DATABASE_SCHEMA.md](docs/DATABASE_SCHEMA.md) | Schema |
| [API_REFERENCE.md](docs/API_REFERENCE.md) | HTTP API |
| [DEVELOPMENT_PROGRESS.md](docs/DEVELOPMENT_PROGRESS.md) | Phase status + tests |
| [REQUIREMENTS_MATRIX.md](docs/REQUIREMENTS_MATRIX.md) | Requirement tracking |
| [TESTING.md](docs/TESTING.md) | How to test |
| [DEPLOYMENT.md](docs/DEPLOYMENT.md) | Docker / VPS |

## License

Proprietary — Rubica HQ.
