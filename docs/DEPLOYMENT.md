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

Repo example (kept in sync with production intent):  
`infrastructure/nginx/emplicon-nginx.d-pikko.site.conf.example`

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

```bash
sudo docker exec pikko-postgres pg_dump -U pikko -d pikko -Fc \
  > ~/backups/pikko-$(date -u +%Y%m%dT%H%M%SZ).dump
```

Volumes: `pikko_postgres_data`, `pikko_api_storage`. Keep dumps outside the compose tree.

## Redeploy / update

```bash
cd ~/pikko
git pull
sudo docker compose -p pikko \
  -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy up -d --build
sudo docker network connect emplicon_emplicon pikko-web || true
sudo docker network connect emplicon_emplicon pikko-api || true
```

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
