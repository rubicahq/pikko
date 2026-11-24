# Database Schema (Phase 0/1)

Migration: `database/migrations/000001_phase1_auth_stores.up.sql`

## Tables

| Table | Purpose |
| --- | --- |
| `users` | Verified accounts (email unique normalized) |
| `authentication_tokens` | Magic-link hashes, expiry, atomic `consumed_at` |
| `sessions` | Session token hashes, revocation, expiry |
| `user_preferences` | Language, timezone |
| `stores` | Store identity, selling model, publication, archive |
| `store_memberships` | User↔store role (`owner`) |
| `store_slug_aliases` | Historical slugs (foundation) |
| `onboarding_drafts` | Resumable two-step onboarding (one row per user) |
| `auth_rate_limits` | Email/IP hourly buckets |
| `email_jobs` | Persistent email delivery + idempotency |
| `audit_logs` | Sensitive store actions |

## Key constraints

- `users.email_normalized` UNIQUE
- `stores.slug` UNIQUE
- `store_memberships (user_id, store_id)` UNIQUE
- Token and session raw values are never stored — SHA-256 hashes only

## Defaults on store create

Theme `pikko_signature`, currency `IDR`, timezone `Asia/Jakarta`, locale `id-ID`, publication `draft`, checkout `false`. Store + owner membership created in one transaction.
