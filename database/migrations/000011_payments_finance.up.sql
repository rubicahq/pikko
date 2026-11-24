-- Agent 07: Payments & financial reconciliation — deposits, multi/consolidated payments,
-- allocations, adjustments, refunds, store credit ledger (feature-flagged). Additive only.

-- Preserve original order total; settlement dimension separate from payment_status (STAT-03 / FIN-*)
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS original_total_idr BIGINT
        CHECK (original_total_idr IS NULL OR original_total_idr >= 0),
    ADD COLUMN IF NOT EXISTS unavailable_delta_idr BIGINT NOT NULL DEFAULT 0
        CHECK (unavailable_delta_idr >= 0),
    ADD COLUMN IF NOT EXISTS credit_applied_idr BIGINT NOT NULL DEFAULT 0
        CHECK (credit_applied_idr >= 0),
    ADD COLUMN IF NOT EXISTS payable_idr BIGINT
        CHECK (payable_idr IS NULL OR payable_idr >= 0),
    ADD COLUMN IF NOT EXISTS settlement_status TEXT NOT NULL DEFAULT 'open'
        CHECK (settlement_status IN (
            'open', 'partially_settled', 'settled', 'refund_due', 'refunded'
        ));

UPDATE orders
SET original_total_idr = total_idr
WHERE original_total_idr IS NULL;

UPDATE orders
SET payable_idr = total_idr
WHERE payable_idr IS NULL;

CREATE INDEX IF NOT EXISTS orders_store_settlement_idx
    ON orders (store_id, settlement_status, updated_at DESC);

-- Per-store deposit policy (ORD-10)
CREATE TABLE IF NOT EXISTS store_deposit_policies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL DEFAULT 'Default deposit',
    applies_to          TEXT NOT NULL DEFAULT 'preorder'
        CHECK (applies_to IN ('all', 'preorder', 'ready_stock')),
    deposit_type        TEXT NOT NULL DEFAULT 'percent'
        CHECK (deposit_type IN ('percent', 'fixed')),
    percent_bps         INT CHECK (percent_bps IS NULL OR (percent_bps >= 0 AND percent_bps <= 10000)),
    fixed_amount_idr    BIGINT CHECK (fixed_amount_idr IS NULL OR fixed_amount_idr >= 0),
    is_enabled          BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT store_deposit_policies_type_check CHECK (
        (deposit_type = 'percent' AND percent_bps IS NOT NULL)
        OR (deposit_type = 'fixed' AND fixed_amount_idr IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS store_deposit_policies_store_idx
    ON store_deposit_policies (store_id, is_enabled);

-- Extend payments for deposit/final/partial/consolidated + idempotency (FIN-07)
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS payment_kind TEXT NOT NULL DEFAULT 'full'
        CHECK (payment_kind IN ('full', 'deposit', 'final', 'partial', 'consolidated')),
    ADD COLUMN IF NOT EXISTS idempotency_key TEXT,
    ADD COLUMN IF NOT EXISTS note TEXT,
    ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1;

CREATE UNIQUE INDEX IF NOT EXISTS payments_store_idempotency_uidx
    ON payments (store_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- Multi-order allocations for consolidated / split payments (E2E-H)
CREATE TABLE IF NOT EXISTS payment_allocations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    payment_id          UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    amount_idr          BIGINT NOT NULL CHECK (amount_idr > 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT payment_allocations_payment_order_unique UNIQUE (payment_id, order_id)
);

CREATE INDEX IF NOT EXISTS payment_allocations_order_idx
    ON payment_allocations (store_id, order_id);
CREATE INDEX IF NOT EXISTS payment_allocations_payment_idx
    ON payment_allocations (payment_id);

-- Append-only financial event history (FIN-08)
CREATE TABLE IF NOT EXISTS financial_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID REFERENCES customers(id) ON DELETE SET NULL,
    order_id            UUID REFERENCES orders(id) ON DELETE SET NULL,
    payment_id          UUID REFERENCES payments(id) ON DELETE SET NULL,
    refund_id           UUID,
    event_type          TEXT NOT NULL
        CHECK (event_type IN (
            'payment_recorded', 'payment_verified', 'payment_rejected',
            'allocation', 'adjustment_unavailable', 'adjustment_manual',
            'refund_entitled', 'refund_issued', 'refund_failed',
            'credit_granted', 'credit_applied', 'credit_revoked'
        )),
    direction           TEXT NOT NULL CHECK (direction IN ('debit', 'credit')),
    amount_idr          BIGINT NOT NULL CHECK (amount_idr >= 0),
    idempotency_key     TEXT NOT NULL,
    meta                JSONB NOT NULL DEFAULT '{}'::jsonb,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT financial_events_store_idempotency_unique UNIQUE (store_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS financial_events_order_idx
    ON financial_events (order_id, created_at);
CREATE INDEX IF NOT EXISTS financial_events_store_idx
    ON financial_events (store_id, created_at DESC);
CREATE INDEX IF NOT EXISTS financial_events_customer_idx
    ON financial_events (store_id, customer_id, created_at DESC)
    WHERE customer_id IS NOT NULL;

-- Financial adjustments from unavailable / manual (UNAV-03 / FIN-04)
CREATE TABLE IF NOT EXISTS financial_adjustments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    order_item_id       UUID REFERENCES order_items(id) ON DELETE SET NULL,
    revision_id         UUID REFERENCES order_revisions(id) ON DELETE SET NULL,
    financial_event_id  UUID REFERENCES financial_events(id) ON DELETE SET NULL,
    adjustment_type     TEXT NOT NULL
        CHECK (adjustment_type IN ('unavailable', 'manual', 'fee', 'credit_applied')),
    amount_idr          BIGINT NOT NULL CHECK (amount_idr >= 0),
    reason              TEXT,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    idempotency_key     TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT financial_adjustments_store_idempotency_unique UNIQUE (store_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS financial_adjustments_order_idx
    ON financial_adjustments (order_id, created_at);

-- Refunds workflow (REF-01, REF-04)
CREATE TABLE IF NOT EXISTS refunds (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    customer_id         UUID REFERENCES customers(id) ON DELETE SET NULL,
    amount_idr          BIGINT NOT NULL CHECK (amount_idr > 0),
    method              TEXT NOT NULL DEFAULT 'bank_transfer'
        CHECK (method IN ('bank_transfer', 'store_credit')),
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    reason              TEXT,
    evidence_file_id    UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    financial_event_id  UUID REFERENCES financial_events(id) ON DELETE SET NULL,
    idempotency_key     TEXT NOT NULL,
    version             INT NOT NULL DEFAULT 1,
    processed_at        TIMESTAMPTZ,
    processed_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT refunds_store_idempotency_unique UNIQUE (store_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS refunds_order_idx ON refunds (order_id, created_at DESC);
CREATE INDEX IF NOT EXISTS refunds_store_status_idx ON refunds (store_id, status, created_at DESC);

ALTER TABLE financial_events
    DROP CONSTRAINT IF EXISTS financial_events_refund_id_fkey;
ALTER TABLE financial_events
    ADD CONSTRAINT financial_events_refund_id_fkey
    FOREIGN KEY (refund_id) REFERENCES refunds(id) ON DELETE SET NULL;

-- Widen payment_evidence purpose already includes payment_evidence; add refund_evidence
ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence', 'refund_evidence'
));

-- Store credit ledger behind feature flags (REF-02/03)
ALTER TABLE stores
    ADD COLUMN IF NOT EXISTS store_credit_enabled BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE IF NOT EXISTS store_credits (
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    balance_idr         BIGINT NOT NULL DEFAULT 0 CHECK (balance_idr >= 0),
    version             INT NOT NULL DEFAULT 1,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (store_id, customer_id)
);

CREATE TABLE IF NOT EXISTS store_credit_ledger (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    entry_type          TEXT NOT NULL
        CHECK (entry_type IN ('credit', 'debit', 'hold', 'release')),
    amount_idr          BIGINT NOT NULL CHECK (amount_idr > 0),
    balance_after_idr   BIGINT NOT NULL CHECK (balance_after_idr >= 0),
    order_id            UUID REFERENCES orders(id) ON DELETE SET NULL,
    refund_id           UUID REFERENCES refunds(id) ON DELETE SET NULL,
    financial_event_id  UUID REFERENCES financial_events(id) ON DELETE SET NULL,
    idempotency_key     TEXT NOT NULL,
    note                TEXT,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT store_credit_ledger_store_idempotency_unique UNIQUE (store_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS store_credit_ledger_customer_idx
    ON store_credit_ledger (store_id, customer_id, created_at DESC);
