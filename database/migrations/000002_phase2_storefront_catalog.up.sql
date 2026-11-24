-- Phase 2: storefront, branding assets, catalog (PRD §16–17, §32)

CREATE TABLE stored_files (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    purpose             TEXT NOT NULL CHECK (purpose IN (
        'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other'
    )),
    storage_key         TEXT NOT NULL,
    content_type        TEXT NOT NULL,
    size_bytes          BIGINT NOT NULL CHECK (size_bytes > 0),
    original_filename   TEXT,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT stored_files_key_unique UNIQUE (storage_key)
);

CREATE INDEX stored_files_store_purpose_idx ON stored_files (store_id, purpose, created_at DESC);

-- Draft vs published storefront configuration (PRD §16)
CREATE TABLE storefront_configs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    channel             TEXT NOT NULL CHECK (channel IN ('draft', 'published')),
    theme_id            TEXT NOT NULL DEFAULT 'pikko_signature',
    background_color    TEXT NOT NULL DEFAULT '#FFFFFF',
    text_color          TEXT NOT NULL DEFAULT '#15191E',
    primary_color       TEXT NOT NULL DEFAULT '#FF3D00',
    logo_file_id        UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    favicon_file_id     UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    store_icon_file_id  UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    description         TEXT,
    contact_email       TEXT,
    whatsapp            TEXT,
    social_links        JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at        TIMESTAMPTZ,
    CONSTRAINT storefront_configs_store_channel_unique UNIQUE (store_id, channel)
);

CREATE INDEX storefront_configs_store_idx ON storefront_configs (store_id);

CREATE TABLE storefront_banners (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    channel             TEXT NOT NULL CHECK (channel IN ('draft', 'published')),
    desktop_file_id     UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    mobile_file_id      UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    headline            TEXT,
    supporting_text     TEXT,
    cta_label           TEXT,
    destination_type    TEXT CHECK (destination_type IS NULL OR destination_type IN ('product', 'collection', 'url')),
    destination_ref     TEXT,
    display_order       INT NOT NULL DEFAULT 0,
    is_active           BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX storefront_banners_store_channel_idx
    ON storefront_banners (store_id, channel, display_order);

CREATE TABLE categories (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL,
    description         TEXT,
    sort_order          INT NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_store_slug_unique UNIQUE (store_id, slug)
);

CREATE INDEX categories_store_idx ON categories (store_id, sort_order, name);

-- Products are the stable link target for future flipbook catalog pages/hotspots:
-- hotspot.product_id -> products.id; public URL via store slug + product slug.
-- status draft|published|archived; optional sku; images via product_images + stored_files.
CREATE TABLE products (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL,
    description         TEXT,
    sku                 TEXT,
    brand               TEXT,
    category_id         UUID REFERENCES categories(id) ON DELETE SET NULL,
    base_price_idr      BIGINT NOT NULL CHECK (base_price_idr >= 0),
    compare_at_price_idr BIGINT CHECK (compare_at_price_idr IS NULL OR compare_at_price_idr >= 0),
    cost_price_idr      BIGINT CHECK (cost_price_idr IS NULL OR cost_price_idr >= 0),
    ready_stock_eligible BOOLEAN NOT NULL DEFAULT true,
    preorder_eligible   BOOLEAN NOT NULL DEFAULT false,
    availability        TEXT NOT NULL DEFAULT 'available'
        CHECK (availability IN ('available', 'unavailable')),
    inventory_mode      TEXT NOT NULL DEFAULT 'unlimited'
        CHECK (inventory_mode IN ('finite', 'unlimited')),
    stock_quantity      INT CHECK (stock_quantity IS NULL OR stock_quantity >= 0),
    weight_grams        INT CHECK (weight_grams IS NULL OR weight_grams >= 0),
    status              TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'published', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT products_store_slug_unique UNIQUE (store_id, slug)
);

CREATE INDEX products_store_status_idx ON products (store_id, status, updated_at DESC);
CREATE INDEX products_store_category_idx ON products (store_id, category_id);
CREATE INDEX products_store_name_idx ON products (store_id, name);

CREATE TABLE product_images (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    file_id             UUID NOT NULL REFERENCES stored_files(id) ON DELETE CASCADE,
    alt_text            TEXT,
    display_order       INT NOT NULL DEFAULT 0,
    is_primary          BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX product_images_product_idx ON product_images (product_id, display_order);
CREATE INDEX product_images_store_idx ON product_images (store_id);

CREATE TABLE product_options (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    display_order       INT NOT NULL DEFAULT 0,
    CONSTRAINT product_options_product_name_unique UNIQUE (product_id, name)
);

CREATE TABLE product_option_values (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    option_id           UUID NOT NULL REFERENCES product_options(id) ON DELETE CASCADE,
    value               TEXT NOT NULL,
    display_order       INT NOT NULL DEFAULT 0,
    CONSTRAINT product_option_values_option_value_unique UNIQUE (option_id, value)
);

CREATE TABLE product_variants (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    sku                 TEXT,
    price_override_idr  BIGINT CHECK (price_override_idr IS NULL OR price_override_idr >= 0),
    stock_quantity      INT CHECK (stock_quantity IS NULL OR stock_quantity >= 0),
    availability        TEXT NOT NULL DEFAULT 'available'
        CHECK (availability IN ('available', 'unavailable')),
    image_file_id       UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    option_key          TEXT NOT NULL DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_variants_product_option_key_unique UNIQUE (product_id, option_key)
);

CREATE INDEX product_variants_store_product_idx ON product_variants (store_id, product_id);

CREATE TABLE variant_option_values (
    variant_id          UUID NOT NULL REFERENCES product_variants(id) ON DELETE CASCADE,
    option_value_id     UUID NOT NULL REFERENCES product_option_values(id) ON DELETE CASCADE,
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    PRIMARY KEY (variant_id, option_value_id)
);

CREATE TABLE collections (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    slug                TEXT NOT NULL,
    description         TEXT,
    cover_file_id       UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    display_order       INT NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'published', 'archived')),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT collections_store_slug_unique UNIQUE (store_id, slug)
);

CREATE INDEX collections_store_status_idx ON collections (store_id, status, display_order);

CREATE TABLE collection_products (
    collection_id       UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    display_order       INT NOT NULL DEFAULT 0,
    PRIMARY KEY (collection_id, product_id)
);

CREATE TABLE storefront_featured_products (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    channel             TEXT NOT NULL CHECK (channel IN ('draft', 'published')),
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    display_order       INT NOT NULL DEFAULT 0,
    CONSTRAINT storefront_featured_products_unique UNIQUE (store_id, channel, product_id)
);

CREATE TABLE storefront_featured_collections (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    channel             TEXT NOT NULL CHECK (channel IN ('draft', 'published')),
    collection_id       UUID NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    display_order       INT NOT NULL DEFAULT 0,
    CONSTRAINT storefront_featured_collections_unique UNIQUE (store_id, channel, collection_id)
);

-- Seed draft storefront config for existing stores
INSERT INTO storefront_configs (store_id, channel, theme_id, background_color, text_color, primary_color)
SELECT id, 'draft', COALESCE(NULLIF(theme, ''), 'pikko_signature'), '#FFFFFF', '#15191E', '#FF3D00'
FROM stores
ON CONFLICT (store_id, channel) DO NOTHING;
