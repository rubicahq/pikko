# System Health (Pikko HQ)

Internal Tim Pikko monitoring at **`https://hq.pikko.site`**. Not part of the merchant dashboard.

Lock: Project store `docs/system-health-hq.md` (2026-11-25 Asia/Jakarta).

## Access

- Host: `hq.pikko.site` (nginx + TLS SAN; same web/API stack as other pikko.site surfaces)
- Auth: magic link only, allowlisted in `hq_admins` (seeded: `rickyhilmi@gmail.com`)
- API prefix: `/api/v1/hq/*` — session + HQ allowlist on every endpoint
- Merchant UI must never deep-link System Health

## Tabs

| Tab | Priority | API |
| --- | --- | --- |
| Overview | P0 | `GET /api/v1/hq/system-health/overview` |
| Error Logs | P0 | `…/issues`, `…/issues/{id}` |
| Background Jobs | P0 | `…/jobs`, `…/jobs/stats`, `…/jobs/{id}/retry` |
| Services | P0 | `…/services?refresh=1` |
| Usage & Costs | P1 | `…/usage` |
| Alerts | P1 | `…/alerts/rules`, `…/alerts/events`, `…/alerts/evaluate` |

Date range query: `range=1h|24h|7d|30d` (default `24h`) or `range=custom&from=&to=` (RFC3339).

## Data honesty

- **Never fabricate metrics.** Missing instrumentation → `unavailable` list / UI “Unavailable”, not zero.
- API success rate + P95 come only from persisted request metrics/samples after deploy.
- Deploy version from `DEPLOY_VERSION` / `GIT_SHA` (deploy.sh exports git SHA). Empty → unavailable.
- AI/email **costs** are unavailable until billing rates exist; call/send counts are measured.
- Service checks: Operational / Degraded / Unavailable / **Unknown**. Config alone never implies Operational for email/AI/public HTTP.
- Normal business validation (`4xx`) is **not** recorded as unexpected errors. `5xx` + worker failures are.

## Persistence

Migration `000026_system_health_hq`:

- `hq_admins`, `system_error_issues`, `system_error_occurrences`
- `api_metric_minutes`, `api_latency_samples`
- `system_service_checks`, `hq_alert_rules`, `hq_alert_events`
- `ai_usage_events`, `hq_admin_audit`

Retention (cleanup ticker): occurrences 30d, resolved/ignored issues 90d, aggregates 90d, latency samples 24h. Logging failures never break merchant requests.

## Security

- Sanitize secrets/PII in messages, stacks, context, job errors (tokens, cookies, emails, keys, payment-like numbers).
- Job retry is **job_queue only** (idempotent requeue of failed/dead). Email/notification queues are not force-retried from HQ.
- Admin actions written to `hq_admin_audit`.

## Ops: tip VPS + `hq.pikko.site`

1. DNS A `hq.pikko.site` → VPS IP (DNS-only until TLS verified).
2. Expand Let’s Encrypt cert with `-d hq.pikko.site`, copy PEMs, reload nginx.
3. Install/update `~/emplicon/deploy/nginx.d/pikko.site.conf` from `infrastructure/nginx/emplicon-nginx.d-pikko.site.conf.example` (includes `hq` in `server_name`).
4. Set `.env`: `HQ_URL=https://hq.pikko.site`, `NEXT_PUBLIC_HQ_URL=…`, CORS includes HQ, `DEPLOY_VERSION` optional (deploy.sh sets SHA).
5. `./scripts/deploy.sh --ref=main` then public smoke includes HQ.

## Out of scope

Full Datadog/Grafana, host CPU/memory agents, distributed tracing stacks, auto remediation, unrestricted shell, merchant analytics.
