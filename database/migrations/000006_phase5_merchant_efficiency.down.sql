DROP TABLE IF EXISTS analytics_events;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;

DROP INDEX IF EXISTS catalog_import_jobs_source_idx;
ALTER TABLE catalog_import_jobs DROP CONSTRAINT IF EXISTS catalog_import_jobs_source_file_check;
ALTER TABLE catalog_import_jobs DROP COLUMN IF EXISTS csv_file_id;
ALTER TABLE catalog_import_jobs DROP COLUMN IF EXISTS source_type;
-- Restore pdf_file_id NOT NULL only when safe; leave nullable on down to avoid data loss.

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence',
    'preorder_cover'
));
