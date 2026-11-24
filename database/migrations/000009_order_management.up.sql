-- Agent 05: Order management — admin entry, item qty architecture, revisions, drafts.
-- Additive only.

-- Order provenance + campaign association
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'customer'
        CHECK (source IN ('customer', 'admin', 'csv_import')),
    ADD COLUMN IF NOT EXISTS created_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS orders_store_source_idx
    ON orders (store_id, source, created_at DESC);
CREATE INDEX IF NOT EXISTS orders_store_campaign_idx
    ON orders (store_id, campaign_id) WHERE campaign_id IS NOT NULL;

-- Item-level quantity / availability foundation (toward ORD-05 / §11 item ops)
ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS availability_status TEXT NOT NULL DEFAULT 'available'
        CHECK (availability_status IN ('available', 'partially_unavailable', 'unavailable')),
    ADD COLUMN IF NOT EXISTS allocated_qty INT NOT NULL DEFAULT 0
        CHECK (allocated_qty >= 0),
    ADD COLUMN IF NOT EXISTS packed_qty INT NOT NULL DEFAULT 0
        CHECK (packed_qty >= 0),
    ADD COLUMN IF NOT EXISTS adjusted_line_total_idr BIGINT
        CHECK (adjusted_line_total_idr IS NULL OR adjusted_line_total_idr >= 0),
    ADD COLUMN IF NOT EXISTS original_quantity INT
        CHECK (original_quantity IS NULL OR original_quantity > 0);

UPDATE order_items
SET original_quantity = quantity
WHERE original_quantity IS NULL;

ALTER TABLE order_items
    ALTER COLUMN original_quantity SET DEFAULT NULL;

-- Append-only order revisions (ORD-09)
CREATE TABLE IF NOT EXISTS order_revisions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    revision_number     INT NOT NULL,
    change_type         TEXT NOT NULL
        CHECK (change_type IN (
            'created', 'status', 'item_qty', 'item_unavailable', 'notes', 'import', 'bulk'
        )),
    summary             TEXT NOT NULL,
    before_snapshot     JSONB,
    after_snapshot      JSONB,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    reason              TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT order_revisions_unique UNIQUE (order_id, revision_number)
);

CREATE INDEX IF NOT EXISTS order_revisions_order_idx
    ON order_revisions (order_id, revision_number DESC);
CREATE INDEX IF NOT EXISTS order_revisions_store_idx
    ON order_revisions (store_id, created_at DESC);

-- Per-item quantity / status events (ORD-05 foundation)
CREATE TABLE IF NOT EXISTS order_item_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    order_item_id       UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    revision_id         UUID REFERENCES order_revisions(id) ON DELETE SET NULL,
    event_type          TEXT NOT NULL
        CHECK (event_type IN (
            'created', 'qty_adjusted', 'availability_changed', 'allocated', 'packed'
        )),
    from_quantity       INT,
    to_quantity         INT,
    from_availability   TEXT,
    to_availability     TEXT,
    meta                JSONB NOT NULL DEFAULT '{}'::jsonb,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    reason              TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS order_item_events_item_idx
    ON order_item_events (order_item_id, created_at);
CREATE INDEX IF NOT EXISTS order_item_events_order_idx
    ON order_item_events (order_id, created_at);

-- Spreadsheet / bulk entry draft autosave (ORD-07)
CREATE TABLE IF NOT EXISTS admin_order_drafts (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    created_by_user_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title               TEXT NOT NULL DEFAULT 'Untitled draft',
    payload             JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_validated_at   TIMESTAMPTZ,
    validation_errors   JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS admin_order_drafts_store_user_idx
    ON admin_order_drafts (store_id, created_by_user_id, updated_at DESC);

-- Backfill campaign_id on existing preorder orders from batch link
UPDATE orders o
SET campaign_id = b.campaign_id
FROM preorder_batches b
WHERE o.preorder_batch_id = b.id
  AND o.campaign_id IS NULL
  AND b.campaign_id IS NOT NULL;
