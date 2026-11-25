-- Catalog storefront cover banners: optional cover image per catalog + upload purpose.

ALTER TABLE catalogs
    ADD COLUMN IF NOT EXISTS cover_file_id UUID REFERENCES stored_files(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS catalogs_store_published_idx
    ON catalogs (store_id, created_at DESC)
    WHERE status = 'published';

-- Restore purposes dropped by 000011 and add catalog_cover for storefront banners.
ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence', 'refund_evidence',
    'preorder_cover', 'catalog_csv',
    'brand_logo', 'campaign_cover',
    'catalog_cover'
));
