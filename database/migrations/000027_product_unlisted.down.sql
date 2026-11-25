UPDATE products SET status='draft' WHERE status='unlisted';
ALTER TABLE products DROP CONSTRAINT products_status_check;
ALTER TABLE products ADD CONSTRAINT products_status_check CHECK (status IN ('draft', 'published', 'archived'));
ALTER TABLE catalog_import_jobs DROP COLUMN image_mode;
