# Launch cutover: pikko.site → pikko.id

**Do not execute this cutover without explicit owner approval.**  
This document is the ordered operator runbook. Code is domain-agnostic; switching hosts is configuration + DNS + TLS + Brevo — not a schema migration.

Related templates:

| Artifact | Path |
| --- | --- |
| Env for pikko.id | `.env.production.pikko.id.example` |
| Nginx serve pikko.id | `infrastructure/nginx/emplicon-nginx.d-pikko.id.conf.example` |
| Nginx redirect pikko.site | `infrastructure/nginx/emplicon-nginx.d-pikko.site-redirect.conf.example` |
| Deploy compose | `docker-compose.deploy.yml` |
| Day-2 ops | `docs/DEPLOYMENT.md`, `docs/RUNBOOK.md` |

Target hosts after cutover:

| Surface | Host |
| --- | --- |
| Marketing | `https://pikko.id` |
| Dashboard | `https://app.pikko.id` |
| Storefront | `https://shop.pikko.id` |
| API | `https://api.pikko.id` |

Legacy hosts keep **308/301** redirects to the matching new host until bookmarks and emails age out.

---

## 0. Preconditions (owner)

Complete before flipping production traffic:

1. Cloudflare zone for **pikko.id** under Rubica control; NS live.
2. Brevo: authenticate **pikko.id** (DKIM, DMARC, SPF, Brevo verification code). Verified senders: `account@`, `notifications@`, `support@`.
3. Rotate secrets that may have leaked in pilots: **SumoPod / AI_API_KEY**, **BREVO_API_KEY**, `POSTGRES_PASSWORD` if shared casually.
4. Counsel-approved privacy/terms if replacing pilot notices (optional but recommended).
5. Commercial pricing decision (or keep early-access framing).
6. Support mailbox / process for `support@pikko.id`.
7. Backup cron already working on VPS (`infrastructure/backup/backup.sh` — uses `sudo docker` when needed).
8. Explicit written approval: “cut over production to pikko.id”.

---

## 1. Ordered cutover steps

### 1.1 Pre-backup

```bash
ssh -p 2021 ubuntu@43.163.99.95
cd ~/pikko
BACKUP_DIR=~/backups/pikko ./infrastructure/backup/backup.sh
```

Record stamp path. Optionally run `restore-test.sh` against that dump.

### 1.2 DNS (Cloudflare) — DNS-only first

Create **A** records (grey cloud / DNS-only) for:

- `pikko.id`, `app.pikko.id`, `shop.pikko.id`, `api.pikko.id` → VPS `43.163.99.95`

Keep **pikko.site** records as-is until redirect vhost is live.

Verify:

```bash
dig +short pikko.id app.pikko.id shop.pikko.id api.pikko.id
```

### 1.3 TLS certificate (Let’s Encrypt)

Issue without stopping Emplicon nginx (webroot):

```bash
sudo certbot certonly --webroot \
  -w /home/ubuntu/emplicon/deploy/acme-webroot \
  -d pikko.id -d app.pikko.id -d shop.pikko.id -d api.pikko.id \
  --agree-tos -m kya@rubica.co.id
```

Copy into nginx SSL mount:

```bash
sudo mkdir -p /home/ubuntu/emplicon/deploy/ssl/pikko-id
sudo cp /etc/letsencrypt/live/pikko.id/fullchain.pem /home/ubuntu/emplicon/deploy/ssl/pikko-id/
sudo cp /etc/letsencrypt/live/pikko.id/privkey.pem /home/ubuntu/emplicon/deploy/ssl/pikko-id/
sudo chown ubuntu:ubuntu /home/ubuntu/emplicon/deploy/ssl/pikko-id/*.pem
sudo chmod 644 /home/ubuntu/emplicon/deploy/ssl/pikko-id/fullchain.pem
sudo chmod 600 /home/ubuntu/emplicon/deploy/ssl/pikko-id/privkey.pem
```

Update renew hook (or add sibling) to copy `pikko.id` PEMs the same way as `pikko.site`.

### 1.4 Install pikko.id nginx vhost (serve)

```bash
cp ~/pikko/infrastructure/nginx/emplicon-nginx.d-pikko.id.conf.example \
  ~/emplicon/deploy/nginx.d/pikko.id.conf
sudo docker exec emplicon-nginx nginx -t
sudo docker exec emplicon-nginx nginx -s reload
```

Smoke **before** env switch (pages may still advertise old URLs in HTML until rebuild):

```bash
curl -sS https://pikko.id/ -o /dev/null -w '%{http_code}\n'
curl -sS https://api.pikko.id/healthz
```

### 1.5 Env + rebuild (app config)

Merge `.env.production.pikko.id.example` into `~/pikko/.env` (mode `600`):

- `APP_BASE_DOMAIN=pikko.id`
- `LEGACY_BASE_DOMAIN=pikko.site`
- `APP_URL` / `MARKETING_URL` / `DASHBOARD_URL` / `STOREFRONT_URL` / `API_BASE_URL`
- Matching `NEXT_PUBLIC_*` including `NEXT_PUBLIC_APP_BASE_DOMAIN` and `NEXT_PUBLIC_LEGACY_BASE_DOMAIN`
- Email From addresses `@pikko.id`
- `API_CORS_ORIGINS` for pikko.id hosts
- `AUTH_REDIRECT_ALLOWLIST` includes **both** `pikko.id` and `pikko.site`
- `AUTH_COOKIE_DOMAIN=` empty (host-only via Next `/api` proxy) unless you deliberately share cookies across subdomains
- `EMAIL_MODE=brevo` only after Brevo verifies pikko.id senders
- Rotated `BREVO_API_KEY` / `AI_API_KEY`

```bash
cd ~/pikko
git fetch origin main && git checkout main && git pull origin main
sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy up -d --build
sudo docker network connect emplicon_emplicon pikko-web || true
sudo docker network connect emplicon_emplicon pikko-api || true
```

No DB migration is required for domain cutover. Store slugs and business rows stay unchanged.

### 1.6 Legacy redirect (pikko.site → pikko.id)

Replace the **serving** `pikko.site.conf` with the redirect snippet (keep ACME):

```bash
# Backup current vhost first
cp ~/emplicon/deploy/nginx.d/pikko.site.conf ~/backups/pikko-tls-$(date +%Y%m%d)/ || true
cp ~/pikko/infrastructure/nginx/emplicon-nginx.d-pikko.site-redirect.conf.example \
  ~/emplicon/deploy/nginx.d/pikko.site-redirect.conf
# Remove or disable the old serving conf so server_names do not collide
rm -f ~/emplicon/deploy/nginx.d/pikko.site.conf
sudo docker exec emplicon-nginx nginx -t
sudo docker exec emplicon-nginx nginx -s reload
```

Expect:

```bash
curl -sSI https://pikko.site/ | head   # 301 → https://pikko.id/
curl -sSI https://app.pikko.site/auth | head
curl -sSI https://shop.pikko.site/demo | head
curl -sSI https://api.pikko.site/healthz | head
```

App middleware also maps legacy Host → new host (308) and **strips `token` query** so magic-link secrets are not bounced through redirects. Users with an open confirm tab should request a **new** magic link on `app.pikko.id`.

### 1.7 Cloudflare Proxied + Full (strict)

Only after HTTPS works end-to-end on origin:

1. Orange-cloud (Proxied) A records for pikko.id hosts.
2. SSL/TLS mode **Full (strict)**.
3. Confirm no Flexible SSL.

### 1.8 Smoke test

```bash
curl -sS https://pikko.id/ -o /dev/null -w '%{http_code}\n'
curl -sS https://app.pikko.id/ -o /dev/null -w '%{http_code}\n'
curl -sS https://shop.pikko.id/ -o /dev/null -w '%{http_code}\n'
curl -sS https://api.pikko.id/{livez,healthz,readyz}
./infrastructure/ops/load-smoke.sh https://pikko.id https://api.pikko.id
```

Manual:

1. Request magic link → inbox From `account@pikko.id` → confirm on `app.pikko.id` → session cookie set.
2. Open existing store overview; storefront URL shows `shop.pikko.id/{slug}`.
3. Place test checkout; tracking + notification links use new hosts.
4. Confirm sitemap/canonicals on `https://pikko.id/sitemap.xml`.
5. Confirm `https://pikko.site/...` redirects permanently.

### 1.9 Komunikasi

- Notify pilot merchants: new primary domain, re-auth if session drops, update shared storefront links.
- Update public bios / WhatsApp / Instagram highlights to `pikko.id` / `shop.pikko.id/{slug}`.
- Internal: support mailbox, status page if any, Brevo sender names.

---

## 2. Rollback

1. Restore previous `~/pikko/.env` (pikko.site URLs) and rebuild compose.
2. Reinstall serving `pikko.site.conf`; remove or disable `pikko.site-redirect.conf` and optionally `pikko.id.conf`.
3. `nginx -t` → reload.
4. DNS: keep pikko.id records or grey-cloud them; traffic returns to pikko.site.
5. Re-verify `https://pikko.site` and magic link on `app.pikko.site`.

Do **not** `docker compose down -v` or drop the database.

---

## 3. Owner-only checklist (must do outside this agent)

| # | Action | Why |
| --- | --- | --- |
| 1 | Rotate SumoPod / `AI_API_KEY` | Pilot keys may be long-lived |
| 2 | Rotate Brevo API key; authenticate **pikko.id** senders | Magic links + transactional mail |
| 3 | Add SPF (+ confirm DKIM/DMARC) for pikko.id on Cloudflare DNS-only | Deliverability |
| 4 | After TLS OK: Proxied + **Full (strict)** | Edge security |
| 5 | End-to-end magic-link login on `app.pikko.id` with real inbox | Auth acceptance |
| 6 | Explicit cutover approval timestamp | PRD §7 / Phase 7 gate |
| 7 | Optional: counsel legal pages; final pricing | Open business decisions |

---

## 4. What code already guarantees

- Absolute URLs for magic links, email, storefront, tracking, SEO canonicals, sitemap come from env (`MARKETING_URL` / `APP_URL`, `DASHBOARD_URL`, `STOREFRONT_URL`, `API_BASE_URL`, `NEXT_PUBLIC_*`).
- `AUTH_COOKIE_DOMAIN` configurable; default host-only.
- CORS and redirect allowlist derive from / include `APP_BASE_DOMAIN` (+ `LEGACY_BASE_DOMAIN`).
- Nginx templates for serve + permanent legacy redirect.
- **No automatic cutover** — production stays on current env until the owner runs this runbook.
