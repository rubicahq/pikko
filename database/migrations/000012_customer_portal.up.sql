-- Agent 08: customer portal auth, refund preferences, address change requests.
-- Additive only. Guest checkout + order_tracking_tokens unchanged.

-- Portal magic-link tokens (email + store scoped; SHA-256 hash only; ≥256-bit raw via auth.NewToken)
CREATE TABLE IF NOT EXISTS customer_portal_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    email_normalized    TEXT NOT NULL,
    token_hash          BYTEA NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    consumed_at         TIMESTAMPTZ,
    request_ip          INET,
    user_agent          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT customer_portal_tokens_hash_unique UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS customer_portal_tokens_store_email_idx
    ON customer_portal_tokens (store_id, email_normalized, created_at DESC);

-- Portal sessions (separate from merchant sessions)
CREATE TABLE IF NOT EXISTS customer_portal_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    token_hash          BYTEA NOT NULL,
    expires_at          TIMESTAMPTZ NOT NULL,
    revoked_at          TIMESTAMPTZ,
    last_seen_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    user_agent          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT customer_portal_sessions_hash_unique UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS customer_portal_sessions_customer_idx
    ON customer_portal_sessions (store_id, customer_id, expires_at DESC);

-- Customer refund method preference + bank details (after verified magic link)
CREATE TABLE IF NOT EXISTS customer_refund_preferences (
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    preferred_method    TEXT NOT NULL DEFAULT 'bank_transfer'
        CHECK (preferred_method IN ('bank_transfer', 'store_credit')),
    bank_name           TEXT,
    account_holder      TEXT,
    account_number      TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (store_id, customer_id)
);

-- Saved addresses: add updated_at for portal edit rules
ALTER TABLE customer_addresses
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

-- Post-order address change requests (merchant approval required)
CREATE TABLE IF NOT EXISTS portal_address_change_requests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'cancelled')),
    recipient_name      TEXT NOT NULL,
    phone               TEXT,
    line1               TEXT NOT NULL,
    line2               TEXT,
    city                TEXT NOT NULL,
    province            TEXT,
    postal_code         TEXT,
    country             TEXT NOT NULL DEFAULT 'ID',
    customer_note       TEXT,
    merchant_note       TEXT,
    reviewed_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS portal_address_change_requests_store_status_idx
    ON portal_address_change_requests (store_id, status, created_at DESC);

CREATE INDEX IF NOT EXISTS portal_address_change_requests_order_idx
    ON portal_address_change_requests (order_id, created_at DESC);

CREATE INDEX IF NOT EXISTS portal_address_change_requests_customer_idx
    ON portal_address_change_requests (store_id, customer_id, created_at DESC);

-- At most one pending address change per order
CREATE UNIQUE INDEX IF NOT EXISTS portal_address_change_requests_order_pending_unique
    ON portal_address_change_requests (order_id)
    WHERE status = 'pending';
