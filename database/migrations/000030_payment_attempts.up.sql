-- Provider attempts hold inquiry references. Provider webhooks/reconciliation
-- own their status changes; Connie only reads this table.
CREATE TABLE payment_attempts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id                UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id                UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    payment_id              UUID REFERENCES payments(id) ON DELETE SET NULL,
    provider                TEXT NOT NULL,
    provider_invoice_number TEXT,
    amount_idr              BIGINT NOT NULL CHECK (amount_idr >= 0),
    status                  TEXT NOT NULL DEFAULT 'pending',
    request_id              TEXT,
    checkout_url            TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE store_payment_methods DROP CONSTRAINT IF EXISTS store_payment_methods_method_type_check;
ALTER TABLE store_payment_methods ADD CONSTRAINT store_payment_methods_method_type_check
    CHECK (method_type IN ('bank_transfer', 'doku_checkout'));
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_type_check;
ALTER TABLE payments ADD CONSTRAINT payments_method_type_check
    CHECK (method_type IN ('bank_transfer', 'paid_by_admin', 'doku_checkout'));

CREATE TABLE store_doku_credentials (
    store_id                    UUID PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
    checkout_client_id          TEXT NOT NULL,
    checkout_secret_ciphertext  BYTEA NOT NULL,
    mcp_client_id               TEXT NOT NULL,
    mcp_api_key_ciphertext      BYTEA NOT NULL,
    environment                 TEXT NOT NULL DEFAULT 'sandbox' CHECK (environment IN ('sandbox')),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT store_doku_checkout_client_unique UNIQUE (checkout_client_id)
);

CREATE INDEX payment_attempts_store_order_idx
    ON payment_attempts (store_id, order_id, created_at DESC);
CREATE UNIQUE INDEX payment_attempts_provider_invoice_idx
    ON payment_attempts (provider, provider_invoice_number)
    WHERE provider_invoice_number IS NOT NULL;
