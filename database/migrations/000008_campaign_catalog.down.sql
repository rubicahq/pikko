-- Reverse Agent 04 campaign/catalog foundations (destructive — for local rollback only).

DROP TABLE IF EXISTS campaign_products;
ALTER TABLE preorder_batches DROP COLUMN IF EXISTS campaign_id;
DROP TABLE IF EXISTS campaigns;

DROP TABLE IF EXISTS product_attribute_values;
DROP TABLE IF EXISTS attribute_definitions;

DROP TABLE IF EXISTS product_vendors;
DROP TABLE IF EXISTS vendors;

ALTER TABLE products DROP COLUMN IF EXISTS brand_id;
DROP TABLE IF EXISTS brands;

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence',
    'preorder_cover'
));
