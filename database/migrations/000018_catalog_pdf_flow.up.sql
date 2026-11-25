-- Catalog PDF locked UX: progress stages, dual/null prices, catalogs + links, product relations.
-- Additive only.

-- Slice 1: named progress stages on import jobs
ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS progress_stage TEXT NOT NULL DEFAULT 'queued';

ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS progress_page INT
        CHECK (progress_page IS NULL OR progress_page >= 1);

ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS progress_pages INT
        CHECK (progress_pages IS NULL OR progress_pages >= 0);

ALTER TABLE catalog_import_jobs
    DROP CONSTRAINT IF EXISTS catalog_import_jobs_progress_stage_check;

ALTER TABLE catalog_import_jobs
    ADD CONSTRAINT catalog_import_jobs_progress_stage_check
        CHECK (progress_stage IN (
            'queued', 'uploading', 'splitting_pages', 'analyzing',
            'ready_for_review', 'completed', 'failed'
        ));

UPDATE catalog_import_jobs
SET progress_stage = CASE
    WHEN status = 'failed' THEN 'failed'
    WHEN status = 'completed' THEN 'completed'
    WHEN status = 'review_required' THEN 'ready_for_review'
    WHEN status = 'processing' AND progress_pct >= 55 THEN 'analyzing'
    WHEN status = 'processing' AND progress_pct >= 30 THEN 'splitting_pages'
    WHEN status = 'processing' THEN 'uploading'
    ELSE 'queued'
END
WHERE progress_stage = 'queued' OR progress_stage IS NULL OR progress_stage = '';

-- Slice 2: normal + discount prices on import items (price_idr kept as effective/legacy)
ALTER TABLE catalog_import_items
    ADD COLUMN IF NOT EXISTS price_normal_idr BIGINT
        CHECK (price_normal_idr IS NULL OR price_normal_idr >= 0);

ALTER TABLE catalog_import_items
    ADD COLUMN IF NOT EXISTS price_discount_idr BIGINT
        CHECK (price_discount_idr IS NULL OR price_discount_idr >= 0);

UPDATE catalog_import_items
SET price_normal_idr = price_idr
WHERE price_normal_idr IS NULL AND price_idr IS NOT NULL;

-- Products: contact-for-price when catalog item has no price
ALTER TABLE products
    ADD COLUMN IF NOT EXISTS contact_for_price BOOLEAN NOT NULL DEFAULT false;

-- Slice 4: first-class catalog document + optional campaign/preorder links
CREATE TABLE IF NOT EXISTS catalogs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id                UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    title                   TEXT NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'published', 'archived')),
    source_import_job_id    UUID REFERENCES catalog_import_jobs(id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS catalogs_store_idx ON catalogs (store_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS catalogs_store_import_unique
    ON catalogs (store_id, source_import_job_id)
    WHERE source_import_job_id IS NOT NULL;

ALTER TABLE catalog_pages
    ADD COLUMN IF NOT EXISTS catalog_id UUID REFERENCES catalogs(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS catalog_pages_catalog_idx ON catalog_pages (catalog_id, display_order);

CREATE TABLE IF NOT EXISTS catalog_campaigns (
    catalog_id      UUID NOT NULL REFERENCES catalogs(id) ON DELETE CASCADE,
    campaign_id     UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (catalog_id, campaign_id)
);

CREATE INDEX IF NOT EXISTS catalog_campaigns_campaign_idx ON catalog_campaigns (campaign_id);
CREATE INDEX IF NOT EXISTS catalog_campaigns_store_idx ON catalog_campaigns (store_id, campaign_id);

CREATE TABLE IF NOT EXISTS catalog_preorder_batches (
    catalog_id      UUID NOT NULL REFERENCES catalogs(id) ON DELETE CASCADE,
    batch_id        UUID NOT NULL REFERENCES preorder_batches(id) ON DELETE CASCADE,
    store_id        UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (catalog_id, batch_id)
);

CREATE INDEX IF NOT EXISTS catalog_preorder_batches_batch_idx ON catalog_preorder_batches (batch_id);
CREATE INDEX IF NOT EXISTS catalog_preorder_batches_store_idx ON catalog_preorder_batches (store_id, batch_id);

-- List-only hotspots (no bbox from AI) still bind products to a page for the storefront list
ALTER TABLE catalog_page_hotspots
    ADD COLUMN IF NOT EXISTS list_only BOOLEAN NOT NULL DEFAULT false;

-- Slice 3: soft product relate / merge across catalogs
CREATE TABLE IF NOT EXISTS product_relations (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id                UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    product_id              UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    related_product_id      UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    relation_type           TEXT NOT NULL
        CHECK (relation_type IN ('same_item', 'merged_into')),
    notes                   TEXT,
    created_by              UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT product_relations_distinct CHECK (product_id <> related_product_id),
    CONSTRAINT product_relations_unique UNIQUE (store_id, product_id, related_product_id, relation_type)
);

CREATE INDEX IF NOT EXISTS product_relations_product_idx
    ON product_relations (store_id, product_id);
CREATE INDEX IF NOT EXISTS product_relations_related_idx
    ON product_relations (store_id, related_product_id);
