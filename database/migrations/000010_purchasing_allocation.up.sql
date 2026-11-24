-- Agent 06: Purchasing aggregation, multi-vendor splits, allocation concurrency, receiving.
-- Additive only.

-- Allow unavailable-only allocations (allocated_qty may be 0).
ALTER TABLE preorder_item_allocations
    DROP CONSTRAINT IF EXISTS preorder_item_allocations_allocated_qty_check;
ALTER TABLE preorder_item_allocations
    ADD CONSTRAINT preorder_item_allocations_allocated_qty_check CHECK (allocated_qty >= 0);

-- Multi-vendor purchase records + optimistic version + received qty handoff
ALTER TABLE preorder_purchasing_records
    ADD COLUMN IF NOT EXISTS vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS received_qty INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS version INT NOT NULL DEFAULT 1;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'preorder_purchasing_received_qty_check'
    ) THEN
        ALTER TABLE preorder_purchasing_records
            ADD CONSTRAINT preorder_purchasing_received_qty_check CHECK (received_qty >= 0);
    END IF;
END $$;

ALTER TABLE preorder_purchasing_records
    DROP CONSTRAINT IF EXISTS preorder_purchasing_unique;
ALTER TABLE preorder_purchasing_records
    ADD CONSTRAINT preorder_purchasing_unique
    UNIQUE NULLS NOT DISTINCT (batch_id, product_id, variant_id, vendor_id);

CREATE INDEX IF NOT EXISTS preorder_purchasing_vendor_idx
    ON preorder_purchasing_records (store_id, vendor_id)
    WHERE vendor_id IS NOT NULL;

-- Receiving foundation (PUR-06 / RCV-01–03)
CREATE TABLE IF NOT EXISTS receiving_records (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    batch_id            UUID REFERENCES preorder_batches(id) ON DELETE SET NULL,
    vendor_id           UUID REFERENCES vendors(id) ON DELETE SET NULL,
    status              TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'partial', 'complete', 'cancelled')),
    notes               TEXT,
    received_at         TIMESTAMPTZ,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS receiving_records_store_idx
    ON receiving_records (store_id, created_at DESC);
CREATE INDEX IF NOT EXISTS receiving_records_batch_idx
    ON receiving_records (store_id, batch_id)
    WHERE batch_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS receiving_lines (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id                UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    receiving_id            UUID NOT NULL REFERENCES receiving_records(id) ON DELETE CASCADE,
    product_id              UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    variant_id              UUID REFERENCES product_variants(id) ON DELETE SET NULL,
    purchasing_record_id    UUID REFERENCES preorder_purchasing_records(id) ON DELETE SET NULL,
    expected_qty            INT NOT NULL DEFAULT 0 CHECK (expected_qty >= 0),
    received_qty            INT NOT NULL DEFAULT 0 CHECK (received_qty >= 0),
    discrepancy_notes       TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS receiving_lines_receiving_idx
    ON receiving_lines (receiving_id);
CREATE INDEX IF NOT EXISTS receiving_lines_store_product_idx
    ON receiving_lines (store_id, product_id);
