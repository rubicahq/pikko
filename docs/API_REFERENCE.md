# API Reference (Phase 0/1)

Base: `/api/v1`. JSON errors: `{ "error": { "code", "message", "request_id" } }`.

## Auth

| Method | Path | Auth | Notes |
| --- | --- | --- | --- |
| POST | `/auth/magic-link/request` | — | Body `{email}`. Generic success. Dev/test + `EMAIL_MODE=log` may return `dev_token`. |
| GET | `/auth/magic-link/peek?token=` | — | Validates token; **does not consume**. |
| POST | `/auth/magic-link/verify` | — | Body `{token}`. Consumes atomically; sets session cookie. |
| GET | `/auth/session` | cookie/bearer | Current user + session |
| POST | `/auth/logout` | — | Revokes current session |
| POST | `/auth/logout-all` | yes | Revokes all sessions |
| GET | `/auth/sessions` | yes | Active sessions |
| DELETE | `/auth/sessions/{sessionID}` | yes | Revoke one |

## Account

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/account` | Profile + preferences |
| PATCH | `/account` | `display_name`, `language`, `timezone` |

## Stores & onboarding

| Method | Path | Notes |
| --- | --- | --- |
| GET | `/stores` | Authorized stores (`?include_archived=true`) |
| POST | `/stores` | Create with full payload (two-step fields) |
| GET | `/stores/slug-available?slug=` | Availability |
| GET | `/stores/suggest-slug?name=` | Suggestion |
| GET | `/stores/{storeID}` | Requires membership |
| PATCH | `/stores/{storeID}` | Patch identity fields |
| POST | `/stores/{storeID}/archive` | Soft archive |
| GET | `/onboarding/draft` | Get/create resumable draft |
| POST | `/onboarding/identity` | Step 1 |
| POST | `/onboarding/selling-model` | Step 2 fields |
| POST | `/onboarding/complete` | Create store + membership |
| GET | `/meta/categories` | Business categories |

## Health

`GET /healthz` — DB ping.
