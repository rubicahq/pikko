-- Phase 3: First Transaction — cart/checkout, orders, bank transfer, tracking (PRD §20–26, §32)

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence'
));

-- Manual bank transfer accounts per store (PRD §24)
CREATE TABLE store_payment_methods (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    method_type         TEXT NOT NULL DEFAULT 'bank_transfer'
        CHECK (method_type IN ('bank_transfer')),
    bank_name           TEXT NOT NULL,
    account_holder      TEXT NOT NULL,
    account_number      TEXT NOT NULL,
    instructions        TEXT,
    payment_deadline_hours INT NOT NULL DEFAULT 48 CHECK (payment_deadline_hours > 0),
    is_enabled          BOOLEAN NOT NULL DEFAULT true,
    display_order       INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX store_payment_methods_store_idx
    ON store_payment_methods (store_id, is_enabled, display_order);

-- Fulfillment config: flat-rate shipping, free shipping, pickup (PRD §21)
CREATE TABLE store_fulfillment_settings (
    store_id            UUID PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
    shipping_enabled    BOOLEAN NOT NULL DEFAULT false,
    shipping_fee_idr    BIGINT NOT NULL DEFAULT 0 CHECK (shipping_fee_idr >= 0),
    free_shipping_enabled BOOLEAN NOT NULL DEFAULT false,
    pickup_enabled      BOOLEAN NOT NULL DEFAULT false,
    pickup_instructions TEXT,
    service_area_notes  TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Store-scoped customers auto-created from orders (PRD §25)
CREATE TABLE customers (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    email               TEXT NOT NULL,
    phone               TEXT,
    first_order_at      TIMESTAMPTZ,
    last_order_at       TIMESTAMPTZ,
    total_orders        INT NOT NULL DEFAULT 0,
    completed_order_value_idr BIGINT NOT NULL DEFAULT 0 CHECK (completed_order_value_idr >= 0),
    internal_notes      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT customers_store_email_unique UNIQUE (store_id, email)
);

CREATE INDEX customers_store_idx ON customers (store_id, last_order_at DESC NULLS LAST);
CREATE INDEX customers_store_phone_idx ON customers (store_id, phone) WHERE phone IS NOT NULL;

CREATE TABLE customer_addresses (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    label               TEXT,
    recipient_name      TEXT NOT NULL,
    phone               TEXT,
    line1               TEXT NOT NULL,
    line2               TEXT,
    city                TEXT NOT NULL,
    province            TEXT,
    postal_code         TEXT,
    country             TEXT NOT NULL DEFAULT 'ID',
    is_default          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX customer_addresses_customer_idx ON customer_addresses (customer_id);

-- Orders with immutable price snapshots (PRD §23)
CREATE TABLE orders (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id         UUID REFERENCES customers(id) ON DELETE SET NULL,
    order_number        TEXT NOT NULL,
    idempotency_key     TEXT NOT NULL,
    selling_model       TEXT NOT NULL DEFAULT 'ready_stock'
        CHECK (selling_model IN ('ready_stock', 'preorder')),
    order_status        TEXT NOT NULL DEFAULT 'confirmed'
        CHECK (order_status IN ('pending_confirmation', 'confirmed', 'completed', 'cancelled')),
    payment_status      TEXT NOT NULL DEFAULT 'unpaid'
        CHECK (payment_status IN (
            'unpaid', 'awaiting_verification', 'paid', 'failed', 'refunded', 'partially_refunded'
        )),
    fulfillment_status  TEXT NOT NULL DEFAULT 'unfulfilled'
        CHECK (fulfillment_status IN (
            'unfulfilled', 'purchasing', 'awaiting_arrival', 'ready_to_ship',
            'ready_for_pickup', 'shipped', 'delivered', 'collected', 'cancelled'
        )),
    currency            TEXT NOT NULL DEFAULT 'IDR',
    subtotal_idr        BIGINT NOT NULL CHECK (subtotal_idr >= 0),
    shipping_fee_idr    BIGINT NOT NULL DEFAULT 0 CHECK (shipping_fee_idr >= 0),
    handling_fee_idr    BIGINT NOT NULL DEFAULT 0 CHECK (handling_fee_idr >= 0),
    total_idr           BIGINT NOT NULL CHECK (total_idr >= 0),
    fulfillment_method  TEXT NOT NULL
        CHECK (fulfillment_method IN ('shipping', 'free_shipping', 'pickup')),
    customer_name       TEXT NOT NULL,
    customer_email      TEXT NOT NULL,
    customer_phone      TEXT,
    shipping_recipient  TEXT,
    shipping_phone      TEXT,
    shipping_line1      TEXT,
    shipping_line2      TEXT,
    shipping_city       TEXT,
    shipping_province   TEXT,
    shipping_postal_code TEXT,
    shipping_country    TEXT,
    customer_notes      TEXT,
    internal_notes      TEXT,
    cancel_reason       TEXT,
    payment_deadline_at TIMESTAMPTZ,
    confirmed_at        TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT orders_store_number_unique UNIQUE (store_id, order_number),
    CONSTRAINT orders_idempotency_unique UNIQUE (idempotency_key)
);

CREATE INDEX orders_store_created_idx ON orders (store_id, created_at DESC);
CREATE INDEX orders_store_status_idx ON orders (store_id, order_status, payment_status, fulfillment_status);
CREATE INDEX orders_store_customer_idx ON orders (store_id, customer_id);
CREATE INDEX orders_store_email_idx ON orders (store_id, customer_email);

CREATE TABLE order_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id          UUID REFERENCES products(id) ON DELETE SET NULL,
    variant_id          UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    product_name        TEXT NOT NULL,
    product_slug        TEXT,
    sku                 TEXT,
    variant_label       TEXT,
    unit_price_idr      BIGINT NOT NULL CHECK (unit_price_idr >= 0),
    quantity            INT NOT NULL CHECK (quantity > 0),
    line_total_idr      BIGINT NOT NULL CHECK (line_total_idr >= 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX order_items_order_idx ON order_items (order_id);
CREATE INDEX order_items_store_idx ON order_items (store_id, order_id);

CREATE TABLE order_status_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    dimension           TEXT NOT NULL
        CHECK (dimension IN ('order', 'payment', 'fulfillment')),
    from_status         TEXT,
    to_status           TEXT NOT NULL,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    reason              TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX order_status_history_order_idx ON order_status_history (order_id, created_at);

-- Tracking tokens: only hash stored; raw token emailed (PRD §26)
CREATE TABLE order_tracking_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    token_hash          BYTEA NOT NULL,
    expires_at          TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT order_tracking_tokens_hash_unique UNIQUE (token_hash)
);

CREATE INDEX order_tracking_tokens_order_idx ON order_tracking_tokens (order_id)
    WHERE revoked_at IS NULL;

CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    payment_method_id   UUID REFERENCES store_payment_methods(id) ON DELETE SET NULL,
    method_type         TEXT NOT NULL DEFAULT 'bank_transfer'
        CHECK (method_type IN ('bank_transfer')),
    amount_idr          BIGINT NOT NULL CHECK (amount_idr >= 0),
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'awaiting_verification', 'verified', 'rejected', 'refunded')),
    bank_name_snapshot  TEXT,
    account_holder_snapshot TEXT,
    account_number_snapshot TEXT,
    instructions_snapshot TEXT,
    verified_at         TIMESTAMPTZ,
    verified_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    rejected_at         TIMESTAMPTZ,
    rejected_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    reject_reason       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX payments_order_idx ON payments (order_id, created_at DESC);
CREATE INDEX payments_store_status_idx ON payments (store_id, status, created_at DESC);

CREATE TABLE payment_evidence (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    payment_id          UUID NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    file_id             UUID NOT NULL REFERENCES stored_files(id) ON DELETE RESTRICT,
    submitted_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    note                TEXT
);

CREATE INDEX payment_evidence_payment_idx ON payment_evidence (payment_id, submitted_at DESC);
CREATE INDEX payment_evidence_store_idx ON payment_evidence (store_id, order_id);

-- Inventory reservations for finite stock (PRD §21)
CREATE TABLE inventory_reservations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    order_item_id       UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id          UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    quantity            INT NOT NULL CHECK (quantity > 0),
    status              TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'released', 'consumed')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at         TIMESTAMPTZ,
    CONSTRAINT inventory_reservations_item_unique UNIQUE (order_item_id)
);

CREATE INDEX inventory_reservations_product_idx
    ON inventory_reservations (store_id, product_id, status);

CREATE TABLE inventory_adjustments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id          UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    delta               INT NOT NULL,
    reason              TEXT NOT NULL,
    order_id            UUID REFERENCES orders(id) ON DELETE SET NULL,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX inventory_adjustments_product_idx ON inventory_adjustments (store_id, product_id, created_at DESC);

-- Public endpoint rate limits (checkout, tracking, evidence)
CREATE TABLE public_rate_limits (
    bucket_key          TEXT NOT NULL,
    window_start        TIMESTAMPTZ NOT NULL,
    count               INT NOT NULL DEFAULT 0,
    CONSTRAINT public_rate_limits_bucket_window_unique UNIQUE (bucket_key, window_start)
);

-- Per-store order number sequence helper
CREATE TABLE store_order_counters (
    store_id            UUID PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
    last_number         BIGINT NOT NULL DEFAULT 0
);
