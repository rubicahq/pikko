-- Dev/local rollback only. Do not run against production with live data.

DROP TABLE IF EXISTS receiving_lines;
DROP TABLE IF EXISTS receiving_records;

DROP INDEX IF EXISTS preorder_purchasing_vendor_idx;

ALTER TABLE preorder_purchasing_records
    DROP CONSTRAINT IF EXISTS preorder_purchasing_unique;
ALTER TABLE preorder_purchasing_records
    DROP CONSTRAINT IF EXISTS preorder_purchasing_received_qty_check;
ALTER TABLE preorder_purchasing_records
    DROP COLUMN IF EXISTS vendor_id,
    DROP COLUMN IF EXISTS received_qty,
    DROP COLUMN IF EXISTS version;
ALTER TABLE preorder_purchasing_records
    ADD CONSTRAINT preorder_purchasing_unique
    UNIQUE NULLS NOT DISTINCT (batch_id, product_id, variant_id);

ALTER TABLE preorder_item_allocations
    DROP CONSTRAINT IF EXISTS preorder_item_allocations_allocated_qty_check;
ALTER TABLE preorder_item_allocations
    ADD CONSTRAINT preorder_item_allocations_allocated_qty_check CHECK (allocated_qty > 0);
