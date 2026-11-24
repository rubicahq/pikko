DROP INDEX IF EXISTS catalog_import_items_job_review_idx;

ALTER TABLE catalog_import_items
    DROP CONSTRAINT IF EXISTS catalog_import_items_review_status_check;

ALTER TABLE catalog_import_items
    DROP COLUMN IF EXISTS review_status;

ALTER TABLE catalog_import_items
    DROP COLUMN IF EXISTS confidence_pct;

ALTER TABLE catalog_import_jobs
    DROP COLUMN IF EXISTS warning_count;

ALTER TABLE catalog_import_jobs
    DROP COLUMN IF EXISTS ai_provider;
