# Pikko operational runbook (Phase 6)

Audience: operators on the Emplicon VPS. Keep Emplicon untouched.

## Surfaces

| Host | Role |
| --- | --- |
| `https://pikko.site` | Marketing |
| `https://app.pikko.site` | Merchant dashboard |
| `https://shop.pikko.site` | Public storefront |
| `https://api.pikko.site` | Go API |

## Health

```bash
curl -sS https://api.pikko.site/livez
curl -sS https://api.pikko.site/healthz
curl -sS https://api.pikko.site/readyz
```

- `livez` — process up
- `healthz` — DB ping
- `readyz` — DB query + local storage path writable

## Deploy / roll forward

See `DEPLOYMENT.md` redeploy section. Summary:

1. Tag current images; take `pg_dump`
2. `git pull origin main`
3. `docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml --profile proxy up -d --build`
4. Re-attach `pikko-web` / `pikko-api` to `emplicon_emplicon` if needed
5. Smoke checklist in `DEPLOYMENT.md`

## Backups

Script: `infrastructure/backup/backup.sh`  
Crontab example: `infrastructure/backup/crontab.example`  
Restore validation: `infrastructure/backup/restore-test.sh /path/to/postgres.dump`

```bash
# Manual
BACKUP_DIR=~/backups/pikko ~/pikko/infrastructure/backup/backup.sh
# Validate newest dump (ephemeral container; does not touch production DB)
LATEST=$(ls -1d ~/backups/pikko/*/postgres.dump | tail -1)
~/pikko/infrastructure/backup/restore-test.sh "$LATEST"
```

Retention default: 14 days (`BACKUP_RETENTION_DAYS`).

## Rate limits

| Scope | Env | Default |
| --- | --- | --- |
| Magic-link per email / hour | `AUTH_RATE_LIMIT_EMAIL_PER_HOUR` | 5 |
| Magic-link per IP / hour | `AUTH_RATE_LIMIT_IP_PER_HOUR` | 20 |
| Coarse API per IP / hour | `API_RATE_LIMIT_IP_PER_HOUR` | 1200 |

429 responses include `Retry-After` where applicable.

## Logging

API emits JSON structured logs (`http_request` with method, path, status, duration_ms, request_id, ip). Do not log magic-link tokens, session secrets, Brevo keys, or payment evidence bodies.

```bash
sudo docker logs --since 1h pikko-api 2>&1 | tail -200
```

## Incidents

| Symptom | First checks |
| --- | --- |
| 5xx on API | `readyz`, `docker logs pikko-api`, disk space, postgres health |
| Blank marketing | `curl -sSI https://pikko.site/`, nginx config, `pikko-web` logs |
| Auth email missing | `EMAIL_MODE`, Brevo key, `email_jobs` rows, Brevo DNS |
| Upload failures | storage volume ownership UID 65532 (see DEPLOYMENT.md) |
| Emplicon affected | Stop. Pikko changes must stay additive; never recreate Emplicon compose |

## Rollback

```bash
cd ~/pikko
git checkout <previous-main-sha>
sudo docker compose -p pikko -f docker-compose.yml -f docker-compose.deploy.yml --profile proxy up -d --build
```

DB restore only with explicit approval — restore into a new volume/DB first when possible.

## Security hardening checklist

- [ ] `AUTH_COOKIE_SECURE=true` in production
- [ ] CORS origins limited to pikko hosts
- [ ] Secrets only in `~/pikko/.env` mode `600`
- [ ] Security headers on API + web middleware
- [ ] `npm audit` / `go mod verify` via `infrastructure/ops/dependency-audit.sh`
- [ ] Dev routes (`/dev/*`) rewritten away in production
- [ ] Auth / dashboard / tracking remain `noindex`

## Light load smoke

```bash
~/pikko/infrastructure/ops/load-smoke.sh https://pikko.site https://api.pikko.site
```

## Accessibility

Marketing pages use semantic landmarks, visible focus rings (design tokens), keyboardable nav/disclosure FAQ, and `prefers-reduced-motion` for reveal animations. Re-check with keyboard + axe after content edits.
