-- Phase 4: Flexible Commerce — preorder batches + business rules (PRD §21–22, §27, §32)

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence',
    'preorder_cover'
));

-- Preorder batches (PRD §22)
CREATE TABLE IF NOT EXISTS preorder_batches (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL,
    description         TEXT,
    cover_file_id       UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    opens_at            TIMESTAMPTZ NOT NULL,
    closes_at           TIMESTAMPTZ NOT NULL,
    purchasing_date     DATE,
    eta_start           DATE,
    eta_end             DATE,
    merchant_notes      TEXT,
    payment_policy      TEXT NOT NULL DEFAULT 'full_after_checkout'
        CHECK (payment_policy IN ('full_after_checkout', 'full_after_confirmation')),
    publication_status  TEXT NOT NULL DEFAULT 'draft'
        CHECK (publication_status IN ('draft', 'published')),
    operational_status  TEXT NOT NULL DEFAULT 'draft'
        CHECK (operational_status IN (
            'draft', 'scheduled', 'open', 'closed', 'purchasing',
            'awaiting_arrival', 'arrived', 'fulfilling', 'completed', 'cancelled'
        )),
    closed_early_at     TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT preorder_batches_store_slug_unique UNIQUE (store_id, slug),
    CONSTRAINT preorder_batches_window_check CHECK (closes_at > opens_at)
);

CREATE INDEX IF NOT EXISTS preorder_batches_store_status_idx
    ON preorder_batches (store_id, operational_status, closes_at);
CREATE INDEX IF NOT EXISTS preorder_batches_store_pub_idx
    ON preorder_batches (store_id, publication_status, opens_at);

CREATE TABLE IF NOT EXISTS preorder_batch_products (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    batch_id            UUID NOT NULL REFERENCES preorder_batches(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    price_override_idr  BIGINT CHECK (price_override_idr IS NULL OR price_override_idr >= 0),
    quantity_limit      INT CHECK (quantity_limit IS NULL OR quantity_limit > 0),
    availability        TEXT NOT NULL DEFAULT 'available'
        CHECK (availability IN ('available', 'unavailable')),
    notes               TEXT,
    display_order       INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT preorder_batch_products_unique UNIQUE (batch_id, product_id)
);

CREATE INDEX IF NOT EXISTS preorder_batch_products_store_idx
    ON preorder_batch_products (store_id, batch_id, display_order);

CREATE TABLE IF NOT EXISTS preorder_batch_variants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    batch_id            UUID NOT NULL REFERENCES preorder_batches(id) ON DELETE CASCADE,
    batch_product_id    UUID NOT NULL REFERENCES preorder_batch_products(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id          UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    price_override_idr  BIGINT CHECK (price_override_idr IS NULL OR price_override_idr >= 0),
    quantity_limit      INT CHECK (quantity_limit IS NULL OR quantity_limit > 0),
    availability        TEXT NOT NULL DEFAULT 'available'
        CHECK (availability IN ('available', 'unavailable')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT preorder_batch_variants_unique UNIQUE (batch_id, variant_id)
);

CREATE INDEX IF NOT EXISTS preorder_batch_variants_product_idx
    ON preorder_batch_variants (store_id, batch_product_id);

-- Quota reservations use inventory_reservations with optional batch_id
ALTER TABLE inventory_reservations
    ADD COLUMN IF NOT EXISTS batch_id UUID REFERENCES preorder_batches(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS inventory_reservations_batch_idx
    ON inventory_reservations (store_id, batch_id, product_id, status)
    WHERE batch_id IS NOT NULL;

-- Purchasing aggregation + allocations (PRD §22)
CREATE TABLE IF NOT EXISTS preorder_purchasing_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    batch_id            UUID NOT NULL REFERENCES preorder_batches(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id          UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    purchased_qty       INT NOT NULL DEFAULT 0 CHECK (purchased_qty >= 0),
    unavailable_qty     INT NOT NULL DEFAULT 0 CHECK (unavailable_qty >= 0),
    notes               TEXT,
    updated_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT preorder_purchasing_unique UNIQUE NULLS NOT DISTINCT (batch_id, product_id, variant_id)
);

CREATE INDEX IF NOT EXISTS preorder_purchasing_batch_idx
    ON preorder_purchasing_records (store_id, batch_id);

CREATE TABLE IF NOT EXISTS preorder_item_allocations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    batch_id            UUID NOT NULL REFERENCES preorder_batches(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    order_item_id       UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id          UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    allocated_qty       INT NOT NULL CHECK (allocated_qty > 0),
    unavailable_qty     INT NOT NULL DEFAULT 0 CHECK (unavailable_qty >= 0),
    resolution          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT preorder_item_allocations_item_unique UNIQUE (order_item_id)
);

CREATE INDEX IF NOT EXISTS preorder_item_allocations_batch_idx
    ON preorder_item_allocations (store_id, batch_id, order_id);

-- Order / item batch snapshots
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS preorder_batch_id UUID REFERENCES preorder_batches(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS batch_name_snapshot TEXT,
    ADD COLUMN IF NOT EXISTS batch_slug_snapshot TEXT,
    ADD COLUMN IF NOT EXISTS batch_eta_start_snapshot DATE,
    ADD COLUMN IF NOT EXISTS batch_eta_end_snapshot DATE,
    ADD COLUMN IF NOT EXISTS requires_approval BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS approval_deadline_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS orders_store_batch_idx
    ON orders (store_id, preorder_batch_id) WHERE preorder_batch_id IS NOT NULL;

ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS batch_id UUID REFERENCES preorder_batches(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS batch_price_idr BIGINT CHECK (batch_price_idr IS NULL OR batch_price_idr >= 0);

-- Business rules with versioning (PRD §27)
CREATE TABLE IF NOT EXISTS business_rules (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    rule_type           TEXT NOT NULL
        CHECK (rule_type IN ('handling_fee', 'minimum_order', 'order_approval')),
    name                TEXT NOT NULL,
    scope_type          TEXT NOT NULL
        CHECK (scope_type IN ('store', 'selling_model', 'batch')),
    scope_selling_model TEXT
        CHECK (scope_selling_model IS NULL OR scope_selling_model IN ('ready_stock', 'preorder')),
    scope_batch_id      UUID REFERENCES preorder_batches(id) ON DELETE CASCADE,
    is_enabled          BOOLEAN NOT NULL DEFAULT true,
    current_version     INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT business_rules_scope_batch_ck CHECK (
        (scope_type = 'batch' AND scope_batch_id IS NOT NULL AND scope_selling_model IS NULL)
        OR (scope_type = 'selling_model' AND scope_selling_model IS NOT NULL AND scope_batch_id IS NULL)
        OR (scope_type = 'store' AND scope_batch_id IS NULL AND scope_selling_model IS NULL)
    )
);

-- One enabled rule per type+scope (prevent ambiguous duplicates)
CREATE UNIQUE INDEX IF NOT EXISTS business_rules_scope_unique
    ON business_rules (
        store_id, rule_type, scope_type,
        COALESCE(scope_selling_model, ''),
        COALESCE(scope_batch_id, '00000000-0000-0000-0000-000000000000'::uuid)
    )
    WHERE is_enabled;

CREATE INDEX IF NOT EXISTS business_rules_store_idx
    ON business_rules (store_id, rule_type, is_enabled);

CREATE TABLE IF NOT EXISTS business_rule_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    rule_id             UUID NOT NULL REFERENCES business_rules(id) ON DELETE CASCADE,
    version             INT NOT NULL CHECK (version > 0),
    -- handling_fee: fee_mode fixed|percent, fee_fixed_idr, fee_percent
    -- minimum_order: min_subtotal_idr, min_quantity
    -- order_approval: require_approval bool (always true when present)
    config              JSONB NOT NULL DEFAULT '{}'::jsonb,
    published_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_by        UUID REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT business_rule_versions_unique UNIQUE (rule_id, version)
);

CREATE INDEX IF NOT EXISTS business_rule_versions_store_idx
    ON business_rule_versions (store_id, rule_id, version DESC);

CREATE TABLE IF NOT EXISTS order_rule_applications (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    rule_id             UUID NOT NULL REFERENCES business_rules(id) ON DELETE RESTRICT,
    rule_version_id     UUID NOT NULL REFERENCES business_rule_versions(id) ON DELETE RESTRICT,
    rule_type           TEXT NOT NULL,
    scope_type          TEXT NOT NULL,
    config_snapshot     JSONB NOT NULL,
    result_snapshot     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT order_rule_applications_order_type_unique UNIQUE (order_id, rule_type)
);

CREATE INDEX IF NOT EXISTS order_rule_applications_store_idx
    ON order_rule_applications (store_id, order_id);
