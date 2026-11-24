-- Agent 04: Campaign unification + catalog foundations (PRD §7–8)
-- Additive only: brands, vendors, custom attributes, campaigns, campaign_products.

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence',
    'preorder_cover',
    'brand_logo', 'campaign_cover'
));

-- ---------------------------------------------------------------------------
-- Brands as first-class entities (CAT-04)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS brands (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL,
    description         TEXT,
    logo_file_id        UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    status              TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT brands_store_slug_unique UNIQUE (store_id, slug)
);

CREATE INDEX IF NOT EXISTS brands_store_status_idx ON brands (store_id, status, name);

ALTER TABLE products
    ADD COLUMN IF NOT EXISTS brand_id UUID REFERENCES brands(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS products_store_brand_idx
    ON products (store_id, brand_id) WHERE brand_id IS NOT NULL;

-- Backfill brand entities from distinct product.brand text (preserve text column).
INSERT INTO brands (store_id, name, slug, status)
SELECT DISTINCT
    p.store_id,
    trim(p.brand),
    lower(regexp_replace(regexp_replace(trim(p.brand), '[^a-zA-Z0-9]+', '-', 'g'), '(^-|-$)', '', 'g')),
    'active'
FROM products p
WHERE p.brand IS NOT NULL AND trim(p.brand) <> ''
ON CONFLICT (store_id, slug) DO NOTHING;

UPDATE products p
SET brand_id = b.id
FROM brands b
WHERE p.store_id = b.store_id
  AND p.brand IS NOT NULL
  AND lower(trim(p.brand)) = lower(b.name)
  AND p.brand_id IS NULL;

-- ---------------------------------------------------------------------------
-- Vendors + product links (CAT-05 foundation; Agent 06 expands purchasing)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS vendors (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL,
    contact_name        TEXT,
    contact_email       TEXT,
    contact_phone       TEXT,
    notes               TEXT,
    status              TEXT NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT vendors_store_slug_unique UNIQUE (store_id, slug)
);

CREATE INDEX IF NOT EXISTS vendors_store_status_idx ON vendors (store_id, status, name);

CREATE TABLE IF NOT EXISTS product_vendors (
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    vendor_id           UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    is_preferred        BOOLEAN NOT NULL DEFAULT false,
    notes               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (product_id, vendor_id)
);

CREATE INDEX IF NOT EXISTS product_vendors_store_vendor_idx
    ON product_vendors (store_id, vendor_id);

-- ---------------------------------------------------------------------------
-- Custom attributes foundation (CAT-06)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS attribute_definitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    key                 TEXT NOT NULL,
    label               TEXT NOT NULL,
    data_type           TEXT NOT NULL
        CHECK (data_type IN ('text', 'number', 'boolean', 'select')),
    options             JSONB NOT NULL DEFAULT '[]'::jsonb,
    applies_to          TEXT NOT NULL DEFAULT 'product'
        CHECK (applies_to IN ('product', 'variant')),
    is_required         BOOLEAN NOT NULL DEFAULT false,
    display_order       INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT attribute_definitions_store_key_unique UNIQUE (store_id, key),
    CONSTRAINT attribute_definitions_key_format CHECK (key ~ '^[a-z][a-z0-9_]{0,63}$')
);

CREATE INDEX IF NOT EXISTS attribute_definitions_store_idx
    ON attribute_definitions (store_id, display_order, label);

CREATE TABLE IF NOT EXISTS product_attribute_values (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    attribute_id        UUID NOT NULL REFERENCES attribute_definitions(id) ON DELETE CASCADE,
    value_text          TEXT,
    value_number        NUMERIC,
    value_boolean       BOOLEAN,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_attribute_values_unique UNIQUE (product_id, attribute_id),
    CONSTRAINT product_attribute_values_has_value CHECK (
        value_text IS NOT NULL OR value_number IS NOT NULL OR value_boolean IS NOT NULL
    )
);

CREATE INDEX IF NOT EXISTS product_attribute_values_store_idx
    ON product_attribute_values (store_id, product_id);

-- ---------------------------------------------------------------------------
-- Campaigns — unify preorder + live shopping flags (CAMP-05)
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS campaigns (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id                UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    campaign_type           TEXT NOT NULL
        CHECK (campaign_type IN ('preorder', 'live')),
    name                    TEXT NOT NULL,
    slug                    TEXT NOT NULL,
    description             TEXT,
    cover_file_id           UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    opens_at                TIMESTAMPTZ NOT NULL,
    closes_at               TIMESTAMPTZ NOT NULL,
    purchasing_date         DATE,
    eta_start               DATE,
    eta_end                 DATE,
    merchant_notes          TEXT,
    payment_policy          TEXT NOT NULL DEFAULT 'full_after_checkout'
        CHECK (payment_policy IN ('full_after_checkout', 'full_after_confirmation')),
    publication_status      TEXT NOT NULL DEFAULT 'draft'
        CHECK (publication_status IN ('draft', 'published')),
    operational_status      TEXT NOT NULL DEFAULT 'draft'
        CHECK (operational_status IN (
            'draft', 'scheduled', 'open', 'closed', 'purchasing',
            'awaiting_arrival', 'arrived', 'fulfilling', 'completed', 'cancelled'
        )),
    closed_early_at         TIMESTAMPTZ,
    -- Live shopping prep (behavior owned by Agent 12)
    live_quick_add_enabled  BOOLEAN NOT NULL DEFAULT false,
    live_show_order_feed    BOOLEAN NOT NULL DEFAULT true,
    preorder_batch_id       UUID UNIQUE REFERENCES preorder_batches(id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT campaigns_store_slug_unique UNIQUE (store_id, slug),
    CONSTRAINT campaigns_window_check CHECK (closes_at > opens_at),
    CONSTRAINT campaigns_type_link_check CHECK (
        campaign_type = 'preorder'
        OR (campaign_type = 'live' AND preorder_batch_id IS NULL)
    )
);

CREATE INDEX IF NOT EXISTS campaigns_store_type_idx
    ON campaigns (store_id, campaign_type, opens_at DESC);
CREATE INDEX IF NOT EXISTS campaigns_store_status_idx
    ON campaigns (store_id, operational_status, closes_at);
CREATE INDEX IF NOT EXISTS campaigns_store_pub_idx
    ON campaigns (store_id, publication_status, opens_at);

ALTER TABLE preorder_batches
    ADD COLUMN IF NOT EXISTS campaign_id UUID UNIQUE REFERENCES campaigns(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS preorder_batches_campaign_idx
    ON preorder_batches (store_id, campaign_id) WHERE campaign_id IS NOT NULL;

-- Backfill campaigns from existing preorder batches
INSERT INTO campaigns (
    store_id, campaign_type, name, slug, description, cover_file_id,
    opens_at, closes_at, purchasing_date, eta_start, eta_end, merchant_notes,
    payment_policy, publication_status, operational_status, closed_early_at,
    preorder_batch_id, created_at, updated_at
)
SELECT
    b.store_id, 'preorder', b.name, b.slug, b.description, b.cover_file_id,
    b.opens_at, b.closes_at, b.purchasing_date, b.eta_start, b.eta_end, b.merchant_notes,
    b.payment_policy, b.publication_status, b.operational_status, b.closed_early_at,
    b.id, b.created_at, b.updated_at
FROM preorder_batches b
WHERE NOT EXISTS (
    SELECT 1 FROM campaigns c WHERE c.preorder_batch_id = b.id
);

UPDATE preorder_batches b
SET campaign_id = c.id
FROM campaigns c
WHERE c.preorder_batch_id = b.id
  AND b.campaign_id IS NULL;

-- Campaign-product config (price / availability / phase overrides)
CREATE TABLE IF NOT EXISTS campaign_products (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    campaign_id         UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    price_override_idr  BIGINT CHECK (price_override_idr IS NULL OR price_override_idr >= 0),
    quantity_limit      INT CHECK (quantity_limit IS NULL OR quantity_limit > 0),
    availability        TEXT NOT NULL DEFAULT 'available'
        CHECK (availability IN ('available', 'unavailable')),
    selling_phase       TEXT
        CHECK (selling_phase IS NULL OR selling_phase IN ('preorder', 'live', 'ready_stock')),
    notes               TEXT,
    display_order       INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT campaign_products_unique UNIQUE (campaign_id, product_id)
);

CREATE INDEX IF NOT EXISTS campaign_products_store_idx
    ON campaign_products (store_id, campaign_id, display_order);

-- Mirror existing preorder batch product assignments into campaign_products
INSERT INTO campaign_products (
    store_id, campaign_id, product_id, price_override_idr, quantity_limit,
    availability, selling_phase, notes, display_order, created_at, updated_at
)
SELECT
    bp.store_id, b.campaign_id, bp.product_id, bp.price_override_idr, bp.quantity_limit,
    bp.availability, 'preorder', bp.notes, bp.display_order, bp.created_at, bp.updated_at
FROM preorder_batch_products bp
JOIN preorder_batches b ON b.id = bp.batch_id
WHERE b.campaign_id IS NOT NULL
ON CONFLICT (campaign_id, product_id) DO NOTHING;
