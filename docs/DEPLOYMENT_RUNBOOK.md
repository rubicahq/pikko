# Deployment Runbook — Pikko (PRD v3.0)

**Active production domain:** `pikko.site` only  
**Official domain cutover:** `pikko.id` is **not automatic** — see §7 and `docs/LAUNCH.md` (owner approval required).  
**Evolves:** `docs/DEPLOYMENT.md`, `docs/RUNBOOK.md`, `docs/LAUNCH.md` (those files remain; this is the v3 operator entrypoint).

---

## 1. Surfaces (current)

| Surface | Host |
| --- | --- |
| Marketing | `https://pikko.site` |
| Merchant app | `https://app.pikko.site` |
| Storefront | `https://shop.pikko.site` |
| API | `https://api.pikko.site` |

Storefront path pattern: `https://shop.pikko.site/{store-slug}`.

---

## 2. VPS topology

| Item | Value |
| --- | --- |
| SSH | `ssh -p 2021 ubuntu@43.163.99.95` |
| Checkout | `~/pikko` |
| Compose project | `pikko` (never recreate/stop Emplicon) |
| Containers | `pikko-postgres`, `pikko-api`, `pikko-web`, optional `pikko-proxy` |
| Volumes | `pikko_postgres_data`, `pikko_api_storage` |
| Edge | Emplicon nginx `:80`/`:443` via additive `nginx.d` snippet |
| TLS | `~/emplicon/deploy/ssl/pikko/` |
| DB host in compose | **`pikko-postgres`** (not bare `postgres` — Emplicon collision) |

**Hard rule:** Do not prune Emplicon containers/volumes. Nginx changes: add snippet → `nginx -t` → `nginx -s reload`.

---

## 3. Environment

Templates:

- `.env.example` — local/dev
- `.env.production.pikko.id.example` — **future** cutover only

Critical vars: `DATABASE_URL` (→ `pikko-postgres`), surface URLs, `AUTH_COOKIE_*`, `API_CORS_ORIGINS`, `BREVO_API_KEY` / `EMAIL_MODE`, `AI_*`, storage, `LEGACY_BASE_DOMAIN` (cutover only).

Never commit secrets. Rotate Brevo/AI keys on owner schedule.

---

## 4. Migrations policy

- Apply via compose `migrate` service / `scripts/deploy.sh` when new `database/migrations/00000N_*.up.sql` lands.
- **Additive only** on production.
- Always backup before migration-bearing deploys (`./scripts/deploy.sh --backup` or `infrastructure/backup/backup.sh`).
- Do **not** run down migrations against production.

Current head: **`000006`**.

---

## 5. Day-2 deploy (pikko.site)

Preferred:

```bash
ssh -p 2021 ubuntu@43.163.99.95
cd ~/pikko
./scripts/deploy.sh                 # pull main, selective rebuild, smoke
# or
./scripts/deploy.sh --force-all     # full api+web rebuild
./scripts/deploy.sh --backup        # force DB backup
./scripts/deploy.sh --public-smoke  # hit public HTTPS
./scripts/deploy.sh --dry-run       # plan only
```

Manual equivalent remains in `docs/DEPLOYMENT.md` (compose up, re-attach `emplicon_emplicon` network if needed).

Warm force-deploy timing note: ~29s when HEAD unchanged (see recent deploy script docs).

---

## 6. Health & smoke

```bash
curl -sS https://api.pikko.site/livez
curl -sS https://api.pikko.site/healthz
curl -sS https://api.pikko.site/readyz
./scripts/smoke.sh   # local/container smoke when on VPS
```

| Probe | Meaning |
| --- | --- |
| livez | Process up |
| healthz | DB ping |
| readyz | DB + storage readiness (503 if fail) |

Post-deploy checklist: magic-link request (EMAIL_MODE=brevo), store list, public storefront slug, checkout disabled/enabled sanity, no Emplicon regression.

---

## 7. pikko.id cutover (owner-gated)

**Do not execute without explicit written approval.**

Full ordered steps live in **`docs/LAUNCH.md`**:

1. Preconditions (Cloudflare zone, Brevo pikko.id auth, key rotation, backup cron, approval)
2. Pre-backup
3. DNS A records DNS-only
4. TLS certbot webroot
5. Env switch + rebuild web (`NEXT_PUBLIC_*` bake)
6. Nginx pikko.id vhost + pikko.site redirect
7. Smoke + magic-link real send
8. Cloudflare Proxied Full Strict (after HTTPS verified)
9. Rollback plan in LAUNCH.md

Code is already domain-agnostic (Phase 7). Agents **must not** flip DNS/env as part of feature work.

---

## 8. Backups

```bash
BACKUP_DIR=~/backups/pikko ~/pikko/infrastructure/backup/backup.sh
# Uses docker, falls back to sudo docker
LATEST=$(ls -1d ~/backups/pikko/*/postgres.dump | tail -1)
~/pikko/infrastructure/backup/restore-test.sh "$LATEST"
```

Retention default 14 days. Cron: `infrastructure/backup/crontab.example`.

---

## 9. Rate limits & ops knobs

| Scope | Env | Default |
| --- | --- | --- |
| Magic-link email/hour | `AUTH_RATE_LIMIT_EMAIL_PER_HOUR` | 5 |
| Magic-link IP/hour | `AUTH_RATE_LIMIT_IP_PER_HOUR` | 20 |
| Coarse API IP/hour | `API_RATE_LIMIT_IP_PER_HOUR` | 1200 |

Access logs on API; dependency audit / load smoke under `infrastructure/ops/`.

---

## 10. Incident notes

| Symptom | Check |
| --- | --- |
| Auth against wrong DB | `DATABASE_URL` host must be `pikko-postgres` |
| CORS failures | `API_CORS_ORIGINS` includes app/marketing/shop |
| Magic link wrong host | Surface URL env + Brevo From domain |
| Web stale public env | Rebuild web image (`NEXT_PUBLIC_*`) |
| Nginx 502 | `docker network connect emplicon_emplicon pikko-web|api` |

---

## 11. Agent 15 expectations

Validate this runbook against VPS reality; update timings/commands; still **never** auto-cutover to pikko.id. Keep Emplicon isolation invariants.
