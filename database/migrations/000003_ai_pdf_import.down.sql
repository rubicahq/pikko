DROP TABLE IF EXISTS catalog_page_hotspots;
DROP TABLE IF EXISTS catalog_pages;
DROP TABLE IF EXISTS catalog_import_items;
DROP TABLE IF EXISTS catalog_import_jobs;
DROP TABLE IF EXISTS outbox_events;
DROP TABLE IF EXISTS job_queue;

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other'
));
