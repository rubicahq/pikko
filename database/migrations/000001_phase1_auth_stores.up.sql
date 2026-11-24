-- Phase 1: account, auth, stores, onboarding (PRD §32)

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           TEXT NOT NULL,
    email_normalized TEXT NOT NULL,
    display_name    TEXT,
    locale          TEXT NOT NULL DEFAULT 'id-ID',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_store_id   UUID,
    CONSTRAINT users_email_normalized_unique UNIQUE (email_normalized)
);

CREATE TABLE authentication_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email_normalized TEXT NOT NULL,
    token_hash      BYTEA NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    consumed_at     TIMESTAMPTZ,
    request_ip      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT authentication_tokens_hash_unique UNIQUE (token_hash)
);

CREATE INDEX authentication_tokens_email_created_idx
    ON authentication_tokens (email_normalized, created_at DESC);

CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      BYTEA NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    created_ip      INET,
    last_ip         INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT sessions_token_hash_unique UNIQUE (token_hash)
);

CREATE INDEX sessions_user_id_idx ON sessions (user_id);

CREATE TABLE user_preferences (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    language        TEXT NOT NULL DEFAULT 'id-ID',
    timezone        TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE stores (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            TEXT NOT NULL,
    slug            TEXT NOT NULL,
    category        TEXT NOT NULL,
    selling_model   TEXT NOT NULL CHECK (selling_model IN ('ready_stock', 'preorder', 'both')),
    currency        TEXT NOT NULL DEFAULT 'IDR',
    timezone        TEXT NOT NULL DEFAULT 'Asia/Jakarta',
    locale          TEXT NOT NULL DEFAULT 'id-ID',
    theme           TEXT NOT NULL DEFAULT 'pikko_signature',
    publication_status TEXT NOT NULL DEFAULT 'draft'
        CHECK (publication_status IN ('draft', 'published', 'archived')),
    checkout_enabled BOOLEAN NOT NULL DEFAULT false,
    archived_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stores_slug_unique UNIQUE (slug)
);

CREATE TABLE store_memberships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            TEXT NOT NULL DEFAULT 'owner' CHECK (role IN ('owner')),
    status          TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT store_memberships_user_store_unique UNIQUE (user_id, store_id)
);

CREATE INDEX store_memberships_store_id_idx ON store_memberships (store_id);
CREATE INDEX store_memberships_user_id_idx ON store_memberships (user_id);

ALTER TABLE users
    ADD CONSTRAINT users_last_store_id_fkey
    FOREIGN KEY (last_store_id) REFERENCES stores(id) ON DELETE SET NULL;

CREATE TABLE store_slug_aliases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    slug            TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT store_slug_aliases_slug_unique UNIQUE (slug)
);

CREATE TABLE onboarding_drafts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    step            INT NOT NULL DEFAULT 1 CHECK (step IN (1, 2)),
    store_name      TEXT,
    store_slug      TEXT,
    category        TEXT,
    selling_model   TEXT CHECK (selling_model IS NULL OR selling_model IN ('ready_stock', 'preorder', 'both')),
    is_first_store  BOOLEAN NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at    TIMESTAMPTZ,
    CONSTRAINT onboarding_drafts_user_active_unique UNIQUE (user_id)
);

CREATE TABLE auth_rate_limits (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bucket_key      TEXT NOT NULL,
    window_start    TIMESTAMPTZ NOT NULL,
    count           INT NOT NULL DEFAULT 1,
    CONSTRAINT auth_rate_limits_bucket_window_unique UNIQUE (bucket_key, window_start)
);

CREATE TABLE email_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key    TEXT NOT NULL,
    to_email        TEXT NOT NULL,
    from_email      TEXT NOT NULL,
    subject         TEXT NOT NULL,
    body_html       TEXT NOT NULL,
    body_text       TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sending', 'sent', 'failed')),
    provider        TEXT NOT NULL DEFAULT 'brevo',
    provider_message_id TEXT,
    idempotency_key TEXT NOT NULL,
    attempts        INT NOT NULL DEFAULT 0,
    last_error      TEXT,
    scheduled_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT email_jobs_idempotency_unique UNIQUE (idempotency_key)
);

CREATE INDEX email_jobs_status_scheduled_idx ON email_jobs (status, scheduled_at);

CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id   UUID REFERENCES users(id) ON DELETE SET NULL,
    store_id        UUID REFERENCES stores(id) ON DELETE SET NULL,
    action          TEXT NOT NULL,
    subject_type    TEXT,
    subject_id      TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_logs_actor_idx ON audit_logs (actor_user_id, created_at DESC);
CREATE INDEX audit_logs_store_idx ON audit_logs (store_id, created_at DESC);
