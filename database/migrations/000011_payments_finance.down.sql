-- Reverse Agent 07 finance migration (destructive; not for production use).

DROP TABLE IF EXISTS store_credit_ledger CASCADE;
DROP TABLE IF EXISTS store_credits CASCADE;
DROP TABLE IF EXISTS financial_adjustments CASCADE;
ALTER TABLE IF EXISTS financial_events DROP CONSTRAINT IF EXISTS financial_events_refund_id_fkey;
DROP TABLE IF EXISTS refunds CASCADE;
DROP TABLE IF EXISTS financial_events CASCADE;
DROP TABLE IF EXISTS payment_allocations CASCADE;
DROP TABLE IF EXISTS store_deposit_policies CASCADE;

ALTER TABLE stores DROP COLUMN IF EXISTS store_credit_enabled;

ALTER TABLE payments
    DROP COLUMN IF EXISTS payment_kind,
    DROP COLUMN IF EXISTS idempotency_key,
    DROP COLUMN IF EXISTS note,
    DROP COLUMN IF EXISTS version;

ALTER TABLE orders
    DROP COLUMN IF EXISTS original_total_idr,
    DROP COLUMN IF EXISTS unavailable_delta_idr,
    DROP COLUMN IF EXISTS credit_applied_idr,
    DROP COLUMN IF EXISTS payable_idr,
    DROP COLUMN IF EXISTS settlement_status;

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence'
));
