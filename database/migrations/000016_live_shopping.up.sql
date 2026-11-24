-- Agent 12: Live shopping ops — featured SKUs, quick-add provenance, revision/event types.
-- Additive only. No video infrastructure.

-- Featured / quick-add SKUs on campaign products (LIVE-01)
ALTER TABLE campaign_products
    ADD COLUMN IF NOT EXISTS featured BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS campaign_products_featured_idx
    ON campaign_products (campaign_id, featured, display_order)
    WHERE featured = true;

-- Order provenance for merchant quick-add during live (LIVE-02)
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_source_check;
ALTER TABLE orders ADD CONSTRAINT orders_source_check
    CHECK (source IN ('customer', 'admin', 'csv_import', 'live_quick_add'));

-- Revisions + item events for live quick-add
ALTER TABLE order_revisions DROP CONSTRAINT IF EXISTS order_revisions_change_type_check;
ALTER TABLE order_revisions ADD CONSTRAINT order_revisions_change_type_check
    CHECK (change_type IN (
        'created', 'status', 'item_qty', 'item_unavailable', 'notes', 'import', 'bulk', 'live_quick_add'
    ));

ALTER TABLE order_item_events DROP CONSTRAINT IF EXISTS order_item_events_event_type_check;
ALTER TABLE order_item_events ADD CONSTRAINT order_item_events_event_type_check
    CHECK (event_type IN (
        'created', 'qty_adjusted', 'availability_changed', 'allocated', 'packed', 'live_added'
    ));

-- Feed lookup by campaign + recency (LIVE-03)
CREATE INDEX IF NOT EXISTS orders_store_campaign_created_idx
    ON orders (store_id, campaign_id, created_at DESC)
    WHERE campaign_id IS NOT NULL;

-- Audit trail for live quick-adds (concurrency / ops forensics)
CREATE TABLE IF NOT EXISTS live_quick_adds (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    campaign_id         UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    order_item_id       UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    product_id          UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT,
    variant_id          UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    quantity            INT NOT NULL CHECK (quantity > 0),
    unit_price_idr      BIGINT NOT NULL CHECK (unit_price_idr >= 0),
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    idempotency_key     TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT live_quick_adds_idempotency_unique UNIQUE (store_id, idempotency_key)
);

CREATE INDEX IF NOT EXISTS live_quick_adds_campaign_idx
    ON live_quick_adds (store_id, campaign_id, created_at DESC);
CREATE INDEX IF NOT EXISTS live_quick_adds_order_idx
    ON live_quick_adds (store_id, order_id, created_at DESC);
