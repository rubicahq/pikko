DROP TABLE IF EXISTS product_relations;

ALTER TABLE catalog_page_hotspots DROP COLUMN IF EXISTS list_only;

DROP TABLE IF EXISTS catalog_preorder_batches;
DROP TABLE IF EXISTS catalog_campaigns;

ALTER TABLE catalog_pages DROP COLUMN IF EXISTS catalog_id;
DROP TABLE IF EXISTS catalogs;

ALTER TABLE products DROP COLUMN IF EXISTS contact_for_price;

ALTER TABLE catalog_import_items DROP COLUMN IF EXISTS price_discount_idr;
ALTER TABLE catalog_import_items DROP COLUMN IF EXISTS price_normal_idr;

ALTER TABLE catalog_import_jobs DROP CONSTRAINT IF EXISTS catalog_import_jobs_progress_stage_check;
ALTER TABLE catalog_import_jobs DROP COLUMN IF EXISTS progress_pages;
ALTER TABLE catalog_import_jobs DROP COLUMN IF EXISTS progress_page;
ALTER TABLE catalog_import_jobs DROP COLUMN IF EXISTS progress_stage;
