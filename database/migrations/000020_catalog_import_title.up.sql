-- Require/store catalog name on PDF import jobs (name-first Add Catalog).
ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS catalog_title TEXT;
