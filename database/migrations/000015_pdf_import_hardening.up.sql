-- Agent 11: PDF/AI catalog import review hardening (additive only)

ALTER TABLE catalog_import_items
    ADD COLUMN IF NOT EXISTS confidence_pct NUMERIC(5,2)
        CHECK (confidence_pct IS NULL OR (confidence_pct >= 0 AND confidence_pct <= 100));

ALTER TABLE catalog_import_items
    ADD COLUMN IF NOT EXISTS review_status TEXT NOT NULL DEFAULT 'pending';

-- Backfill before CHECK (existing rows already have DEFAULT pending)
UPDATE catalog_import_items
SET review_status = CASE
    WHEN created_product_id IS NOT NULL THEN 'accepted'
    WHEN selected = false THEN 'rejected'
    ELSE COALESCE(NULLIF(review_status, ''), 'pending')
END
WHERE review_status IS NULL OR review_status = 'pending';

ALTER TABLE catalog_import_items
    DROP CONSTRAINT IF EXISTS catalog_import_items_review_status_check;

ALTER TABLE catalog_import_items
    ADD CONSTRAINT catalog_import_items_review_status_check
        CHECK (review_status IN ('pending', 'accepted', 'rejected'));

CREATE INDEX IF NOT EXISTS catalog_import_items_job_review_idx
    ON catalog_import_items (job_id, review_status, display_order);

ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS warning_count INT NOT NULL DEFAULT 0;

ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS ai_provider TEXT;
