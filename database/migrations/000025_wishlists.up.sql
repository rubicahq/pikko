-- Scheduled Live Shopping wishlists: phone-gated public save + merchant visibility.
-- Additive only.

CREATE TABLE IF NOT EXISTS wishlist_items (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    customer_id     UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    campaign_id     UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id      UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    -- Denormalized key so UNIQUE works with nullable variant_id (nil → zero UUID).
    variant_key     UUID NOT NULL DEFAULT '00000000-0000-0000-0000-000000000000',
    catalog_id      UUID REFERENCES catalogs(id) ON DELETE SET NULL,
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT wishlist_items_unique UNIQUE (customer_id, campaign_id, product_id, variant_key)
);

CREATE INDEX IF NOT EXISTS wishlist_items_store_customer_idx
    ON wishlist_items (store_id, customer_id, created_at DESC);

CREATE INDEX IF NOT EXISTS wishlist_items_store_campaign_idx
    ON wishlist_items (store_id, campaign_id, created_at DESC);

CREATE INDEX IF NOT EXISTS wishlist_items_customer_campaign_idx
    ON wishlist_items (customer_id, campaign_id, created_at DESC);
