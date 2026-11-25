ALTER TABLE products DROP CONSTRAINT products_status_check;
ALTER TABLE products ADD CONSTRAINT products_status_check CHECK (status IN ('draft', 'published', 'unlisted', 'archived'));
ALTER TABLE catalog_import_jobs ADD COLUMN image_mode TEXT NOT NULL DEFAULT 'none' CHECK (image_mode IN ('none', 'first_page', 'crop'));
