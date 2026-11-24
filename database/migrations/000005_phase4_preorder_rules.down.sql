-- Phase 4 down (destructive; not for production without approval)

DROP TABLE IF EXISTS order_rule_applications;
DROP TABLE IF EXISTS business_rule_versions;
DROP TABLE IF EXISTS business_rules;

ALTER TABLE order_items
    DROP COLUMN IF EXISTS batch_id,
    DROP COLUMN IF EXISTS batch_price_idr;

ALTER TABLE orders
    DROP COLUMN IF EXISTS preorder_batch_id,
    DROP COLUMN IF EXISTS batch_name_snapshot,
    DROP COLUMN IF EXISTS batch_slug_snapshot,
    DROP COLUMN IF EXISTS batch_eta_start_snapshot,
    DROP COLUMN IF EXISTS batch_eta_end_snapshot,
    DROP COLUMN IF EXISTS requires_approval,
    DROP COLUMN IF EXISTS approval_deadline_at;

DROP TABLE IF EXISTS preorder_item_allocations;
DROP TABLE IF EXISTS preorder_purchasing_records;

DROP INDEX IF EXISTS inventory_reservations_batch_idx;
ALTER TABLE inventory_reservations DROP COLUMN IF EXISTS batch_id;

DROP TABLE IF EXISTS preorder_batch_variants;
DROP TABLE IF EXISTS preorder_batch_products;
DROP TABLE IF EXISTS preorder_batches;

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence'
));
