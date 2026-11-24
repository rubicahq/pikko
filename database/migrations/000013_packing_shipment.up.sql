-- Agent 09: Packing queues + shipments (PACK-01–03, SHIP-01–03).
-- Additive only. Reuses order_items.packed_qty and fulfillment statuses.

CREATE TABLE IF NOT EXISTS packing_tasks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'in_progress', 'packed', 'cancelled')),
    assigned_user_id    UUID REFERENCES users(id) ON DELETE SET NULL,
    notes               TEXT,
    packed_at           TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT packing_tasks_store_order_unique UNIQUE (store_id, order_id)
);

CREATE INDEX IF NOT EXISTS packing_tasks_store_status_idx
    ON packing_tasks (store_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS packing_tasks_order_idx
    ON packing_tasks (order_id);

CREATE TABLE IF NOT EXISTS packing_task_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    packing_task_id     UUID NOT NULL REFERENCES packing_tasks(id) ON DELETE CASCADE,
    order_item_id       UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    qty_to_pack         INT NOT NULL DEFAULT 0 CHECK (qty_to_pack >= 0),
    qty_packed          INT NOT NULL DEFAULT 0 CHECK (qty_packed >= 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT packing_task_items_unique UNIQUE (packing_task_id, order_item_id),
    CONSTRAINT packing_task_items_qty_check CHECK (qty_packed <= qty_to_pack)
);

CREATE INDEX IF NOT EXISTS packing_task_items_task_idx
    ON packing_task_items (packing_task_id);

CREATE TABLE IF NOT EXISTS shipments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    order_id            UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    carrier             TEXT NOT NULL,
    tracking_number     TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'label_created'
        CHECK (status IN ('label_created', 'shipped', 'in_transit', 'delivered', 'cancelled')),
    notes               TEXT,
    shipped_at          TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    notify_customer     BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS shipments_store_created_idx
    ON shipments (store_id, created_at DESC);
CREATE INDEX IF NOT EXISTS shipments_order_idx
    ON shipments (store_id, order_id);
CREATE INDEX IF NOT EXISTS shipments_tracking_idx
    ON shipments (store_id, tracking_number);

CREATE TABLE IF NOT EXISTS shipment_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    shipment_id         UUID NOT NULL REFERENCES shipments(id) ON DELETE CASCADE,
    order_item_id       UUID NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    quantity            INT NOT NULL CHECK (quantity > 0),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT shipment_items_unique UNIQUE (shipment_id, order_item_id)
);

CREATE INDEX IF NOT EXISTS shipment_items_shipment_idx
    ON shipment_items (shipment_id);
CREATE INDEX IF NOT EXISTS shipment_items_order_item_idx
    ON shipment_items (order_item_id);
