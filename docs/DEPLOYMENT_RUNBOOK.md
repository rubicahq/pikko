# Deployment Runbook — Pikko (PRD v3.0 / Agent 15)

**Audience:** owner / operator on the production VPS  
**Active production domain:** `pikko.site` only  
**Official domain cutover:** `pikko.id` is **not automatic** — see §7 and `docs/LAUNCH.md` (explicit written approval required).  
**Evolves:** `docs/DEPLOYMENT.md`, `docs/RUNBOOK.md`, `docs/LAUNCH.md` (those files remain; **this is the day-2 ops entrypoint**).

---

## 0. Owner-ready checklist (no cutover)

Use this before/after every production deploy on `pikko.site`:

- [ ] SSH works: `ssh -p 2021 ubuntu@43.163.99.95`
- [ ] Checkout is `~/pikko` on `main` (or intended ref)
- [ ] `~/pikko/.env` mode `600`; `DATABASE_URL` host is **`pikko-postgres`**
- [ ] Emplicon stack healthy (do not recreate/stop it)
- [ ] Prefer `./scripts/deploy.sh` (BuildKit, selective rebuild, backup-on-migrate, smoke, rollback tags)
- [ ] Post-deploy: local smoke + optional `--public-smoke`
- [ ] Magic-link request smoke (`EMAIL_MODE=brevo`) when mail path changed
- [ ] **Do not** flip DNS/env to `pikko.id` as part of feature deploys

---

## 1. Surfaces (current)

| Surface | Host |
| --- | --- |
| Marketing | `https://pikko.site` |
| Merchant app | `https://app.pikko.site` |
| Storefront | `https://shop.pikko.site/{store-slug}` |
| API | `https://api.pikko.site` |

URLs are **env-driven** (`MARKETING_URL`, `DASHBOARD_URL`, `STOREFRONT_URL`, `API_BASE_URL`, `APP_BASE_DOMAIN`, matching `NEXT_PUBLIC_*`). Smoke public checks read the same env (E2E-N). Defaults today match `pikko.site` — changing them without DNS/TLS/Brevo is not a cutover.

---

## 2. VPS topology

| Item | Value |
| --- | --- |
| SSH | `ssh -p 2021 ubuntu@43.163.99.95` |
| Checkout | `~/pikko` |
| Compose project | `pikko` (**never** recreate/stop Emplicon) |
| Containers | `pikko-postgres`, `pikko-api`, `pikko-web`, optional `pikko-proxy` |
| Volumes | `pikko_postgres_data`, `pikko_api_storage` |
| Host ports (loopback) | API `127.0.0.1:8100` → 8080; Web `127.0.0.1:3100` → 3000 |
| Edge | Emplicon nginx `:80`/`:443` via additive `nginx.d` snippet |
| TLS | `~/emplicon/deploy/ssl/pikko/` |
| DB hostname in compose | **`pikko-postgres`** (not bare `postgres` — Emplicon collision) |

**Hard rule:** Do not prune Emplicon containers/volumes. Nginx changes: add snippet → `nginx -t` → `nginx -s reload`. Deploy script only **attaches** `pikko-web` / `pikko-api` to `emplicon_emplicon` additively.

Compose always:

```bash
docker compose -p pikko \
  -f docker-compose.yml -f docker-compose.deploy.yml \
  --profile proxy …
```

(`scripts/deploy.sh` embeds this; never omit the deploy overlay or host ports `:8100`/`:3100` disappear.)

---

## 3. Environment

Templates:

- `.env.example` — local/dev + smoke/backup knobs
- `.env.production.pikko.id.example` — **future** cutover only (do not apply without §7)

Critical vars: `DATABASE_URL` (→ `pikko-postgres`), surface URLs, `AUTH_COOKIE_*`, `API_CORS_ORIGINS`, `BREVO_API_KEY` / `EMAIL_MODE`, `AI_*`, storage, rate limits, `LEGACY_BASE_DOMAIN` (cutover only).

Never commit secrets. Rotate Brevo/AI keys on owner schedule.

---

## 4. Migrations policy

| Rule | Detail |
| --- | --- |
| Apply path | Compose `migrate` one-shot (api waits on `service_completed_successfully`) via `./scripts/deploy.sh` |
| Production policy | **Additive `*.up.sql` only** |
| Backup | Automatic before migration-bearing deploys; force with `--backup` |
| Down migrations | **Do not** run `*.down.sql` against production |
| Rollback of code | Image `pre-<stamp>` tags / prior SHA rebuild — see §8 |
| DB restore | Explicit approval only; prefer restore-test into ephemeral DB first |

**Current migration head:** `000016_live_shopping` (tree `000001`–`000016`).  
Agents 13–15 added **no** new migrations (analytics/security/deploy are code/docs only).

| Version | Name | Notes |
| --- | --- | --- |
| 000001–000006 | Phase 1–5 foundation | Auth → merchant efficiency |
| 000007 | `team_rbac` | Roles + invites |
| 000008 | `campaign_catalog` | Campaigns / brands / attrs |
| 000009 | `order_management` | Admin orders / revisions |
| 000010 | `purchasing_allocation` | Vendors / receiving |
| 000011 | `payments_finance` | Ledger / refunds / credit |
| 000012 | `customer_portal` | Portal identities |
| 000013 | `packing_shipment` | Pack / ship |
| 000014 | `whatsapp_templates` | WA registry |
| 000015 | `pdf_import_hardening` | Import review fields |
| 000016 | `live_shopping` | Featured SKUs / quick-add |

Deploy logs `schema_migrations` clean head after `compose_up`. Dirty migration flag → stop and investigate; do not force.

---

## 5. Day-2 deploy (pikko.site)

```bash
ssh -p 2021 ubuntu@43.163.99.95
cd ~/pikko
./scripts/deploy.sh                 # pull main, selective rebuild, smoke
./scripts/deploy.sh --force-all     # full api+web(+migrate image) rebuild
./scripts/deploy.sh --backup        # force DB dump even without new migrations
./scripts/deploy.sh --public-smoke  # + public HTTPS via env URLs
./scripts/deploy.sh --dry-run       # plan only (no docker mutations)
./scripts/deploy.sh --skip-smoke    # deploy without smoke (not recommended)
./scripts/deploy.sh --no-pull       # current checkout only
```

### What the script guarantees

1. **BuildKit** (`DOCKER_BUILDKIT=1`) + parallel selective rebuild  
2. **Always** both compose files (loopback ports preserved)  
3. **Backup** before migrations (or `--backup`) → `~/backups/pikko/deploy/pikko-<stamp>.dump` + `.sha256`  
4. **Postgres healthy** wait before backup/migrate  
5. Rebuild **migrate image** with api when API/migrations change (avoids stale SQL)  
6. Tag `pikko-api|web:pre-<stamp>` and **rollback** on port/smoke/`healthz` failure  
7. Additive Emplicon network attach  
8. Local smoke (+ optional public); **never** flips hosts to `pikko.id`

Warm force-deploy (unchanged HEAD, cached BuildKit) historically ~29s wall clock; no-op same-SHA skip is ~1s + smoke.

Manual equivalent: `docs/DEPLOYMENT.md`.

---

## 6. Health & smoke

```bash
# From VPS (loopback — what deploy uses)
./scripts/smoke.sh
SMOKE_PUBLIC=1 ./scripts/smoke.sh

# Public (or any env-configured hosts)
curl -sS https://api.pikko.site/livez
curl -sS https://api.pikko.site/healthz
curl -sS https://api.pikko.site/readyz
```

| Probe | Meaning |
| --- | --- |
| livez | Process up |
| healthz | DB ping |
| readyz | DB + storage writable (503 if fail) |

**Env knobs (E2E-N):** `SMOKE_API_URL`, `SMOKE_WEB_URL`, `SMOKE_PUBLIC`, `SMOKE_READYZ`, `SMOKE_TIMEOUT_SEC`, `SMOKE_PUBLIC_*`, or surface URLs from `.env`.

Post-deploy checklist: magic-link request, store list, public storefront slug, checkout sanity, **no Emplicon regression**.

---

## 7. pikko.id cutover (owner-gated)

**Do not execute without explicit written approval.**  
Agents and `deploy.sh` **must not** flip DNS/env as part of feature work.

Full ordered steps: **`docs/LAUNCH.md`**.

Summary: preconditions → pre-backup → DNS A (DNS-only) → TLS → env + web rebuild → nginx vhost + legacy redirect → smoke → Cloudflare Proxied Full Strict → rollback plan in LAUNCH.md.

---

## 8. Backups & rollback

### Scheduled / full backup

```bash
BACKUP_DIR=~/backups/pikko ~/pikko/infrastructure/backup/backup.sh
LATEST=$(ls -1d ~/backups/pikko/*/postgres.dump | tail -1)
~/pikko/infrastructure/backup/restore-test.sh "$LATEST"
```

Retention default 14 days. Cron: `infrastructure/backup/crontab.example`.

### Deploy-time dump

Written under `~/backups/pikko/deploy/` by `deploy.sh` when migrations change (or `--backup`). Keep for the release window; prune manually if disk tight (cron retention targets dated **directories** from `backup.sh`, not flat deploy dumps).

### Image / code rollback

```bash
# Prefer automatic: failed smoke already retags pre-<stamp>
# Manual image rollback example:
sudo docker tag pikko-api:pre-YYYYMMDDThhmmssZ pikko-api:latest
sudo docker tag pikko-web:pre-YYYYMMDDThhmmssZ pikko-web:latest
cd ~/pikko
./scripts/deploy.sh --no-pull --skip-smoke   # or compose up --no-build
./scripts/smoke.sh
```

### DB rollback

- Prefer **forward fix** (additive migration) over down SQL.  
- Restoring a dump onto production requires **explicit approval**.  
- Always `restore-test.sh` against an ephemeral container first.

---

## 9. Rate limits & ops knobs

| Scope | Env | Default |
| --- | --- | --- |
| Magic-link email/hour | `AUTH_RATE_LIMIT_EMAIL_PER_HOUR` | 5 |
| Magic-link IP/hour | `AUTH_RATE_LIMIT_IP_PER_HOUR` | 20 |
| Coarse API IP/hour | `API_RATE_LIMIT_IP_PER_HOUR` | 1200 |

Access logs on API; dependency audit / load smoke under `infrastructure/ops/`. Nginx must keep `X-Real-IP` (Agent 14 rate-limit trust).

---

## 10. Incident notes

| Symptom | Check |
| --- | --- |
| Auth against wrong DB | `DATABASE_URL` host must be `pikko-postgres` |
| CORS failures | `API_CORS_ORIGINS` includes app/marketing/shop |
| Magic link wrong host | Surface URL env + Brevo From domain |
| Web stale public env | Rebuild web image (`NEXT_PUBLIC_*` bake) |
| Nginx 502 | `docker network connect emplicon_emplicon pikko-web\|api` |
| Ports 8100/3100 missing | Deploy overlay not applied — re-run `deploy.sh` |
| Migrate dirty / stuck | Inspect `schema_migrations`; do not `down` on prod |
| Upload failures | Storage volume UID **65532** (see `DEPLOYMENT.md`) |
| Emplicon impacted | **Stop.** Pikko changes stay additive only |

---

## 11. Agent 15 completion

Validated for production ops on current VPS (`ssh -p 2021`, `pikko.site` stack):

- Scripts: BuildKit, backup-on-migrate, postgres wait, health wait, rollback, env-only public smoke  
- Migration docs through **`000016`**  
- Emplicon coexistence preserved (`pikko-postgres`, project `pikko`)  
- **No** automatic `pikko.id` cutover  
