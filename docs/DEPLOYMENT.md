# Deployment

## Target

VPS (same host as Emplicon): `ssh -p 2021 ubuntu@43.163.99.95`  
Checkout path on the server: `~/pikko` (separate from `~/emplicon`).

**Hard rule:** Pikko is its own Docker Compose project. Do **not** recreate, stop, or prune Emplicon containers/volumes. Nginx changes for Pikko are additive (`nginx.d` snippet + graceful `nginx -s reload` after `nginx -t` only).

| Surface | Host |
| --- | --- |
| Marketing | `https://pikko.site` |
| Dashboard | `https://app.pikko.site` |
| Storefront | `https://shop.pikko.site` |
| API | `https://api.pikko.site` |

### DNS / Cloudflare (current)

- NS: Cloudflare
- A records (DNS-only / grey cloud): `pikko.site`, `app.`, `shop.`, `api.` → `43.163.99.95`
- Keep **DNS-only** until HTTPS is verified; enable Cloudflare Proxied afterward if desired (then use Full / Full Strict)
- `www.pikko.site` is optional (no A record required for the current cert)

## Architecture on the VPS

| Piece | Detail |
| --- | --- |
| Compose project | `pikko` |
| Network | `pikko_pikko` (+ `pikko-web` / `pikko-api` also attached to `emplicon_emplicon`) |
| Volumes | `pikko_postgres_data`, `pikko_api_storage` |
| Containers | `pikko-postgres`, `pikko-api`, `pikko-web`, `pikko-proxy` (optional `:9090`), migrate one-shot |
| Public edge | `emplicon-nginx` `:80`/`:443` via `~/emplicon/deploy/nginx.d/pikko.site.conf` |
| TLS files | `~/emplicon/deploy/ssl/pikko/{fullchain,privkey}.pem` (from Let’s Encrypt) |
| ACME webroot | `~/emplicon/deploy/acme-webroot` (mounted at `/var/www/acme`) |
| Renew hook | `/etc/letsencrypt/renewal-hooks/deploy/pikko.sh` |
| DB hostname | Compose `DATABASE_URL` must use host **`pikko-postgres`** (container_name), never bare `postgres` |

### Why `DATABASE_URL` pins `pikko-postgres`

On the VPS, `pikko-api` is also attached to `emplicon_emplicon` so nginx can reach it. That shared network already has Emplicon’s Postgres advertised as hostname **`postgres`**. If Pikko’s `DATABASE_URL` uses `@postgres:5432`, Docker DNS can resolve to **emplicon-postgres** instead of Pikko’s DB — auth and email jobs then fail against the wrong database.

**Fix:** set `container_name: pikko-postgres` and use `@pikko-postgres:5432` in compose `DATABASE_URL` for `migrate` and `api` (`docker-compose.yml` + `docker-compose.deploy.yml`). Compose service key can remain `postgres`; only the connection hostname is pinned. Do not put a bare `postgres` host into server `.env` `DATABASE_URL` for the containers.

Repo examples (kept in sync with production intent):

- Current pilots: `infrastructure/nginx/emplicon-nginx.d-pikko.site.conf.example`
- Official domain (cutover): `infrastructure/nginx/emplicon-nginx.d-pikko.id.conf.example`
- Legacy redirect after cutover: `infrastructure/nginx/emplicon-nginx.d-pikko.site-redirect.conf.example`
- Env template for pikko.id: `.env.production.pikko.id.example`
- **Ordered cutover runbook:** `docs/LAUNCH.md` (do not execute without owner approval)

## First-time VPS deploy

```bash
git clone --branch cursor/phase-0-1-foundation-auth-b6f8 git@github.com:rubicahq/pikko.git ~/pikko
cd ~/pikko

umask 077
cp .env.example .env
# Set POSTGRES_PASSWORD (long random). Keep EMAIL_MODE=log until Brevo is ready.
# Production URLs:
#   MARKETING_URL=https://pikko.site
#   DASHBOARD_URL=https://app.pikko.site
#   STOREFRONT_URL=https://shop.pikko.site
#   API_BASE_URL=https://api.pikko.site
#   NEXT_PUBLIC_* matching the above
#   AUTH_COOKIE_SECURE=true
#   API_CORS_ORIGINS=https://app.pikko.site,https://pikko.site,https://shop.pikko.site
chmod 600 .env

sudo docker compose -p pikko \
  -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy up -d --build

# Let Emplicon nginx reach Pikko (additive; does not restart Emplicon)
sudo docker network connect emplicon_emplicon pikko-web
sudo docker network connect emplicon_emplicon pikko-api
```

Services: `postgres`, `migrate` (one-shot), `api`, `web`, optional `proxy` on `:9090`.

### TLS (Let’s Encrypt, Emplicon pattern)

1. Install HTTP+ACME vhost first (or full HTTPS config from the example once certs exist).
2. Issue cert **without stopping nginx** (webroot):

```bash
sudo certbot certonly --webroot \
  -w /home/ubuntu/emplicon/deploy/acme-webroot \
  -d pikko.site -d app.pikko.site -d shop.pikko.site -d api.pikko.site \
  --agree-tos -m kya@rubica.co.id
```

3. Copy into the nginx SSL mount and install renew hook (already present on the VPS as `pikko.sh`):

```bash
sudo mkdir -p /home/ubuntu/emplicon/deploy/ssl/pikko
sudo cp /etc/letsencrypt/live/pikko.site/fullchain.pem /home/ubuntu/emplicon/deploy/ssl/pikko/
sudo cp /etc/letsencrypt/live/pikko.site/privkey.pem /home/ubuntu/emplicon/deploy/ssl/pikko/
sudo chown ubuntu:ubuntu /home/ubuntu/emplicon/deploy/ssl/pikko/*.pem
sudo chmod 644 /home/ubuntu/emplicon/deploy/ssl/pikko/fullchain.pem
sudo chmod 600 /home/ubuntu/emplicon/deploy/ssl/pikko/privkey.pem
```

4. Install HTTPS vhost → `nginx -t` → `nginx -s reload` (never recreate `emplicon-nginx` for a config change).

Auto-renew uses `certbot.timer` + deploy hook that copies PEMs and reloads nginx. Verify with:

```bash
sudo certbot renew --dry-run --cert-name pikko.site
```

### Rollback (nginx vhost only)

1. Remove `~/emplicon/deploy/nginx.d/pikko.site.conf` (or restore previous copy from `~/backups/pikko-tls-*`).
2. `sudo docker exec emplicon-nginx nginx -t`
3. `sudo docker exec emplicon-nginx nginx -s reload`

Emplicon server blocks are unchanged by this rollback.

## Verify

```bash
dig +short pikko.site app.pikko.site shop.pikko.site api.pikko.site
curl -sSI http://pikko.site/ | head    # expect 301 → https
curl -sS https://pikko.site/ -o /dev/null -w '%{http_code}\n'
curl -sS https://app.pikko.site/ -o /dev/null -w '%{http_code}\n'
curl -sS https://shop.pikko.site/ -o /dev/null -w '%{http_code}\n'
curl -sS https://api.pikko.site/healthz
# Emplicon smoke (local, bypass Cloudflare):
curl -sk --resolve emplicon.com:443:127.0.0.1 https://emplicon.com/ -o /dev/null -w '%{http_code}\n'
```

## Secrets

Never commit `.env`. On the VPS: `~/pikko/.env` mode `600`.

- `POSTGRES_PASSWORD` / DB URL
- Brevo (transactional magic-link email):
  - `BREVO_API_KEY` (server `.env` only)
  - `EMAIL_MODE=brevo` when a verified sender exists; otherwise keep `log`
  - `EMAIL_FROM_NAME`, `EMAIL_ACCOUNT_FROM`, `EMAIL_NOTIFICATION_FROM`, `EMAIL_REPLY_TO`
  - Production uses verified sender `account@pikko.site` (from-name `Pikko`)
  - Recreate **only** `pikko-api` after changing Brevo/email env (`docker compose … up -d --no-deps --force-recreate api`), then re-attach to `emplicon_emplicon` if needed
- storage credentials when using S3
- AI gateway (OpenAI-compatible; server `.env` only):
  - `AI_API_KEY`
  - `AI_BASE_URL` (use `…/v1` chat/completions + models under that base)
  - `AI_MODEL` (production currently uses `gemini/gemini-3.5-flash`)

Do not put API keys in the repo, docs with real values, logs, or agent stores.

### Brevo domain DNS (Cloudflare DNS-only)

`pikko.site` is authenticated in Brevo. Expected records (names relative to `pikko.site`):

| Type | Name | Value |
| --- | --- | --- |
| TXT | `@` | `brevo-code:…` (Brevo domain verification code) |
| CNAME | `brevo1._domainkey` | `b1.pikko-site.dkim.brevo.com` |
| CNAME | `brevo2._domainkey` | `b2.pikko-site.dkim.brevo.com` |
| TXT | `_dmarc` | `v=DMARC1; p=none; rua=mailto:rua@dmarc.brevo.com` |
| TXT | `@` | `v=spf1 include:spf.brevo.com ~all` |

DKIM / DMARC / Brevo code were present at cutover. **SPF was missing** on the apex — add the SPF TXT above (DNS-only) so all `@pikko.site` senders stay SPF-clean. Do not invent unverified From addresses.

## Migrations

Compose runs migrate before api. Manual:

```bash
sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml run --rm migrate
```

## App rollback

```bash
cd ~/pikko
git fetch && git checkout <previous-sha>
sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml --profile proxy up -d --build
sudo docker network connect emplicon_emplicon pikko-web || true
sudo docker network connect emplicon_emplicon pikko-api || true
```

Do **not** `docker compose down -v`, prune volumes, or drop the database without explicit approval.

## Backup notes

Scheduled script (Phase 6): `infrastructure/backup/backup.sh`  
Crontab example: `infrastructure/backup/crontab.example`  
Restore validation (ephemeral container): `infrastructure/backup/restore-test.sh`  

Scripts prefer unprivileged `docker`, then fall back to `sudo docker` when the invoking user is not in the docker group (typical on the VPS). Storage volume archival uses `sudo test` / `sudo tar` because mountpoints under `/var/lib/docker` are root-owned.

```bash
# Manual one-shot (legacy)
sudo docker exec pikko-postgres pg_dump -U pikko -d pikko -Fc \
  > ~/backups/pikko-$(date -u +%Y%m%dT%H%M%SZ).dump

# Preferred
BACKUP_DIR=~/backups/pikko ~/pikko/infrastructure/backup/backup.sh
```

Volumes: `pikko_postgres_data`, `pikko_api_storage`. Keep dumps outside the compose tree. See `docs/RUNBOOK.md`.

Optional env: `API_RATE_LIMIT_IP_PER_HOUR` (default 1200). Health: `/livez`, `/healthz`, `/readyz`.

### Session cookie across marketing + app

Production nginx serves `pikko.site` and `app.pikko.site` from the same web/API stack and proxies `/api/` **directly to Go**. Magic-link verify therefore sets the cookie for whichever `Host` confirmed the link (`app.pikko.site` by default). With `AUTH_COOKIE_DOMAIN` empty (host-only), revisiting `pikko.site` looks logged-out.

Set:

```bash
AUTH_COOKIE_DOMAIN=pikko.site   # or pikko.id after cutover
AUTH_SESSION_TTL_DAYS=30        # default; meets “≥24h stay signed in”
AUTH_COOKIE_SECURE=true
AUTH_COOKIE_SAMESITE=lax
```

Then recreate `api` (env) and redeploy `web` (marketing `/` and `/auth` redirect signed-in users to `https://app…/stores`). Existing host-only cookies are not upgraded — one fresh magic-link sign-in issues the shared cookie.

## Redeploy / update (main)

**Preferred:** idempotent script (keeps deploy overlay ports, selective rebuild, optional backup, smoke, rollback):

```bash
cd ~/pikko
./scripts/deploy.sh                 # pull main, rebuild only what changed, smoke
./scripts/deploy.sh --force-all     # rebuild api+web anyway
./scripts/deploy.sh --backup        # force pg_dump even without new migrations
./scripts/deploy.sh --skip-smoke    # deploy without smoke
./scripts/smoke.sh                  # smoke only (loopback)
SMOKE_PUBLIC=1 ./scripts/smoke.sh   # + public HTTPS checks
```

### What the script optimizes

Measured on the VPS (wall clock, BuildKit on; times in Asia/Jakarta session 2026-11-24):

| Step | Before (naive docs / wrong flags) | After (`./scripts/deploy.sh`) |
| --- | --- | --- |
| `git fetch/pull` | ~3s | ~3s |
| DB backup | always ~0.5–1s | **skipped** unless migrations (or `--backup`) |
| Web rebuild (source change, deps cached) | **~37s** (`npm run build` ~28s) | ~37s only if web paths changed |
| API rebuild | sequential after web | **parallel** with web when both needed |
| Full force rebuild both (cache cold after Dockerfile change) | ~60s+ and risk lost ports | **87s** once (npm ci + go + next); then cached |
| Force rebuild both (warm BuildKit / Next cache) | ~40–60s + port recovery | **~29s** (build 26s + up 2s + smoke 1s); ports kept |
| No-op redeploy (same SHA) | still `--build` everything | **~1s** skip; smoke optional |
| Omit deploy overlay | **ports :3100/:8100 lost**, +10–20s recreate | always both `-f` files |
| Smoke | mixed into deploy | `<1s` via `scripts/smoke.sh` |

Naive sequential `up -d --build` for a web-only change often spent ~40–60s and sometimes recreated Postgres / dropped host ports. The script:

1. Always uses `-f docker-compose.yml -f docker-compose.deploy.yml` (ports stay on `127.0.0.1:8100` / `:3100`)
2. Enables BuildKit + cache mounts (`go mod` / `go-build` / `.npm` / Next `.next/cache`)
3. Builds **web + api in parallel** when both changed; **skips** unchanged services
4. Backs up DB **only** when `database/migrations/*` changed (override with `--backup`)
5. Tags `pikko-*:pre-<stamp>` and **rolls back** if smoke/`healthz` fails
6. Attaches to `emplicon_emplicon` additively (never recreates Emplicon)

Manual equivalent (legacy):

```bash
cd ~/pikko
# Keep .env (secrets). Prefer ./scripts/deploy.sh instead of this block.
sudo docker tag pikko-api:latest pikko-api:pre-$(date -u +%Y%m%dT%H%M%SZ)
sudo docker tag pikko-web:latest pikko-web:pre-$(date -u +%Y%m%dT%H%M%SZ)

git fetch origin main && git checkout main && git pull origin main

sudo docker compose -p pikko \
  -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy build --parallel api web
sudo docker compose -p pikko \
  -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy up -d --no-build

sudo docker network connect emplicon_emplicon pikko-web || true
sudo docker network connect emplicon_emplicon pikko-api || true
./scripts/smoke.sh
```

### Storage volume ownership

`pikko-api` runs as distroless `nonroot` (UID/GID **65532**). After the first volume create, if uploads return `upload_failed`, fix ownership **without deleting the volume**:

```bash
sudo chown -R 65532:65532 /var/lib/docker/volumes/pikko_api_storage/_data
```

### Smoke checklist (post-deploy)

- `https://pikko.site`, `app.`, `shop.` → 200; `https://api.pikko.site/healthz` → 200
- Magic-link request OK (or expected 429 cooldown)
- Create store → upload logo/favicon → product → publish → `https://shop.pikko.site/{slug}`
- PDF import job reaches `review_required` / completed; public flipbook `/ {slug}/catalog`
- Cleanup test users/stores afterward

Note: `GET /api/v1/auth/session` currently reads the session **cookie** only; Bearer works on other authenticated routes via middleware.

## Local without Docker

1. PostgreSQL 16+ with `pikko` / `pikko_test`
2. `cp .env.example .env` and set `DATABASE_URL`
3. `make migrate` → `make api` → `cd apps/web && npm run dev`

## Local Docker

```bash
cp .env.example .env
docker compose up --build -d
# optional proxy: publish a free host port in an override, then --profile proxy
```
