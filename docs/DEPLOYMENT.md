# Deployment

## Target

VPS (same host as Emplicon): `ssh -p 2021 ubuntu@43.163.99.95`  
Checkout path on the server: `~/pikko` (separate from `~/emplicon`).

**Hard rule:** Pikko is its own Docker Compose project. Do **not** modify, restart, remove, or prune Emplicon containers, volumes, networks, or nginx config while deploying Pikko.

Intended hostnames (PRD / production DNS):

| Surface | Host |
| --- | --- |
| Marketing | `pikko.site` |
| Dashboard | `app.pikko.site` |
| Storefront | `shop.pikko.site` |
| API | `api.pikko.site` |

### DNS status (checked 2026-11-24)

- `pikko.site` NS: Cloudflare (`mona.ns.cloudflare.com`, `jaime.ns.cloudflare.com`)
- **No A/AAAA records** yet for `pikko.site`, `app.`, `shop.`, `api.`, or `www.`
- Until DNS + TLS exist, Pikko is exposed on a free host port via its own nginx proxy (see Interim access below). Do not point Emplicon’s `:80`/`:443` nginx at Pikko until DNS is ready.

## Emplicon pattern (read-only reference)

Emplicon runs as compose project services on network `emplicon_emplicon`, with `emplicon-nginx` publishing host `:80`/`:443` and including `deploy/nginx.d/*.conf`. Extra apps (e.g. Rubica Engage) join that network and add an nginx.d snippet.

Pikko mirrors the isolation idea with:

- Compose project name: `pikko`
- Network: `pikko_pikko`
- Volumes: `pikko_postgres_data`, `pikko_api_storage`
- Containers: `pikko-postgres`, `pikko-api`, `pikko-web`, `pikko-proxy` (+ one-shot `pikko-migrate-*`)
- Own reverse proxy on host **`:9090`** (does not conflict with Emplicon)

## First-time VPS deploy

```bash
# On the VPS
git clone --branch cursor/phase-0-1-foundation-auth-b6f8 git@github.com:rubicahq/pikko.git ~/pikko
cd ~/pikko

# Generate production secrets on the server only (never commit or print)
umask 077
cp .env.example .env
# Set POSTGRES_PASSWORD to a long random value, keep EMAIL_MODE=log
# until a Brevo API key is supplied. For interim HTTP port access set:
#   AUTH_COOKIE_SECURE=false
#   MARKETING_URL / DASHBOARD_URL / STOREFRONT_URL / API_BASE_URL
#     and NEXT_PUBLIC_* counterparts to http://43.163.99.95:9090
#   API_CORS_ORIGINS to match those origins
chmod 600 .env

sudo docker compose -p pikko \
  -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy up -d --build
```

Services: `postgres`, `migrate` (one-shot), `api`, `web`, `proxy`.

Dockerfiles:

- `Dockerfile.api` — Go API + migrate binary
- `Dockerfile.web` — Next.js standalone

### Interim access (DNS not ready)

| Entry | URL |
| --- | --- |
| App + API (via pikko-proxy) | http://43.163.99.95:9090/ |
| Health | http://43.163.99.95:9090/healthz |
| Loopback web | http://127.0.0.1:3100/ |
| Loopback API | http://127.0.0.1:8100/healthz |

`pikko-proxy` is the VPS front door for now. Host `:80`/`:443` remain Emplicon’s.

### After DNS + TLS are ready

1. Create Cloudflare A records for `pikko.site`, `www`, `app`, `shop`, `api` → `43.163.99.95` (proxied or DNS-only per TLS plan).
2. Issue/install TLS certs (Cloudflare origin cert or Let’s Encrypt) under a path Emplicon nginx can mount, e.g. `~/emplicon/deploy/ssl/pikko/`.
3. Attach Pikko app containers to Emplicon’s network (**additive**, does not restart Emplicon):

   ```bash
   sudo docker network connect emplicon_emplicon pikko-web
   sudo docker network connect emplicon_emplicon pikko-api
   ```

4. Install `infrastructure/nginx/emplicon-nginx.d-pikko.site.conf.example` as `~/emplicon/deploy/nginx.d/pikko.site.conf`, adjust TLS paths, then `nginx -t` + `nginx -s reload` inside `emplicon-nginx` (reload only — do not recreate the container).
5. Update `~/pikko/.env` URLs to `https://…pikko.site`, set `AUTH_COOKIE_SECURE=true`, rebuild/recreate `api` + `web`, and optionally stop publishing `:9090` once hostname routing works.

## Local without Docker

1. PostgreSQL 16+ with databases `pikko` / `pikko_test`
2. `cp .env.example .env` and set `DATABASE_URL`
3. `make migrate`
4. Terminal A: `make api`
5. Terminal B: `cd apps/web && npm run dev`

Web proxies `/api/v1/*` to the API so session cookies stay same-origin on localhost:3000.

## Secrets

Never commit `.env` or API keys. Required production secrets (generated on the VPS):

- `POSTGRES_PASSWORD` / `DATABASE_URL`
- `BREVO_API_KEY` (only when `EMAIL_MODE=brevo`)
- storage credentials when using S3

Keep `EMAIL_MODE=log` until Brevo sender domains are verified and a key is provided.

## TLS / proxy

Terminate TLS at nginx (Emplicon front door or a future dedicated edge). Forward `X-Forwarded-For` and `X-Forwarded-Proto`. Set `AUTH_COOKIE_SECURE=true` only when the browser reaches the app over HTTPS.

## Migrations

Compose runs `migrate` before `api` starts:

```bash
sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml run --rm migrate
```

## Verify

```bash
sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml --profile proxy ps
curl -sS http://127.0.0.1:9090/healthz
curl -sS -o /dev/null -w '%{http_code}\n' http://127.0.0.1:9090/
curl -sS http://127.0.0.1:9090/api/v1/auth/session   # expect 401 JSON when logged out
```

Confirm Emplicon containers were not restarted (compare `docker inspect … StartedAt` / container IDs before and after).

## Rollback

1. **App-only rollback (preferred):** rebuild/recreate from a previous known-good git SHA / image tag:

   ```bash
   cd ~/pikko
   git fetch && git checkout <previous-sha>
   sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml --profile proxy up -d --build
   ```

2. **Stop Pikko without deleting data:**

   ```bash
   sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml --profile proxy stop
   ```

3. **Do not** run `docker compose down -v`, `docker volume rm`, `docker system prune`, or drop the Postgres database. Volume deletion requires explicit operator approval.

4. Prefer forward-fix migrations. If a bad migration ships, restore from backup (below) before any destructive SQL.

## Backup notes

- Postgres data lives in Docker volume `pikko_postgres_data`.
- Object/local uploads live in `pikko_api_storage`.
- Logical dump example (non-destructive):

  ```bash
  sudo docker exec pikko-postgres pg_dump -U pikko -d pikko -Fc \
    > ~/backups/pikko-$(date -u +%Y%m%dT%H%M%SZ).dump
  ```

- Keep dumps off the compose project directory; encrypt/store per your normal VPS backup routine.
- Restoring a dump is a deliberate ops action — never automate restore into a live volume without approval.

## Redeploy / update

```bash
cd ~/pikko
git pull
sudo docker compose -p pikko \
  -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy up -d --build
```

## Local Docker (developer machine)

```bash
cp .env.example .env
docker compose up --build -d
# optional proxy: add a compose override that publishes e.g. "8088:80", then:
docker compose --profile proxy up --build -d
```
