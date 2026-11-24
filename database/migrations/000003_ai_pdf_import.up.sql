-- Phase 5 accelerator / AI catalog PDF import + catalog pages for flipbook (PRD §18, §32)

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop'
));

-- Generic PostgreSQL-backed job queue with idempotency (PRD §31)
CREATE TABLE job_queue (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID REFERENCES stores(id) ON DELETE CASCADE,
    job_type            TEXT NOT NULL,
    payload             JSONB NOT NULL DEFAULT '{}'::jsonb,
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'dead')),
    idempotency_key     TEXT NOT NULL,
    attempts            INT NOT NULL DEFAULT 0,
    max_attempts        INT NOT NULL DEFAULT 5,
    last_error          TEXT,
    available_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    locked_at           TIMESTAMPTZ,
    locked_by           TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    CONSTRAINT job_queue_idempotency_unique UNIQUE (idempotency_key)
);

CREATE INDEX job_queue_claim_idx ON job_queue (status, available_at)
    WHERE status IN ('pending', 'failed');

CREATE INDEX job_queue_store_idx ON job_queue (store_id, created_at DESC);

-- Transactional outbox for domain events
CREATE TABLE outbox_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID REFERENCES stores(id) ON DELETE CASCADE,
    event_type          TEXT NOT NULL,
    payload             JSONB NOT NULL DEFAULT '{}'::jsonb,
    idempotency_key     TEXT NOT NULL,
    published_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT outbox_events_idempotency_unique UNIQUE (idempotency_key)
);

CREATE INDEX outbox_events_pending_idx ON outbox_events (created_at)
    WHERE published_at IS NULL;

CREATE TABLE catalog_import_jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    created_by          UUID REFERENCES users(id) ON DELETE SET NULL,
    pdf_file_id         UUID NOT NULL REFERENCES stored_files(id) ON DELETE RESTRICT,
    extract_images      BOOLEAN NOT NULL DEFAULT true,
    status              TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'processing', 'review_required', 'completed', 'failed')),
    page_count          INT,
    item_count          INT NOT NULL DEFAULT 0,
    progress_pct        INT NOT NULL DEFAULT 0 CHECK (progress_pct >= 0 AND progress_pct <= 100),
    error_message       TEXT,
    job_queue_id        UUID REFERENCES job_queue(id) ON DELETE SET NULL,
    idempotency_key     TEXT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    CONSTRAINT catalog_import_jobs_idempotency_unique UNIQUE (idempotency_key)
);

CREATE INDEX catalog_import_jobs_store_idx ON catalog_import_jobs (store_id, created_at DESC);

CREATE TABLE catalog_import_items (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    job_id              UUID NOT NULL REFERENCES catalog_import_jobs(id) ON DELETE CASCADE,
    display_order       INT NOT NULL DEFAULT 0,
    selected            BOOLEAN NOT NULL DEFAULT true,
    uncertain           BOOLEAN NOT NULL DEFAULT false,
    warning             TEXT,
    name                TEXT,
    description         TEXT,
    brand               TEXT,
    price_idr           BIGINT CHECK (price_idr IS NULL OR price_idr >= 0),
    sku                 TEXT,
    category_name       TEXT,
    variant_info        JSONB NOT NULL DEFAULT '{}'::jsonb,
    image_file_id       UUID REFERENCES stored_files(id) ON DELETE SET NULL,
    page_number         INT CHECK (page_number IS NULL OR page_number >= 1),
    bbox_x_pct          NUMERIC(6,3) CHECK (bbox_x_pct IS NULL OR (bbox_x_pct >= 0 AND bbox_x_pct <= 100)),
    bbox_y_pct          NUMERIC(6,3) CHECK (bbox_y_pct IS NULL OR (bbox_y_pct >= 0 AND bbox_y_pct <= 100)),
    bbox_width_pct      NUMERIC(6,3) CHECK (bbox_width_pct IS NULL OR (bbox_width_pct >= 0 AND bbox_width_pct <= 100)),
    bbox_height_pct     NUMERIC(6,3) CHECK (bbox_height_pct IS NULL OR (bbox_height_pct >= 0 AND bbox_height_pct <= 100)),
    raw_extraction      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_product_id  UUID REFERENCES products(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX catalog_import_items_job_idx ON catalog_import_items (job_id, display_order);
CREATE INDEX catalog_import_items_store_idx ON catalog_import_items (store_id, job_id);

-- Catalog flipbook pages (draft from PDF import; publish in flipbook editor)
CREATE TABLE catalog_pages (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    image_file_id       UUID NOT NULL REFERENCES stored_files(id) ON DELETE RESTRICT,
    display_order       INT NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'draft'
        CHECK (status IN ('draft', 'published')),
    source_import_job_id UUID REFERENCES catalog_import_jobs(id) ON DELETE SET NULL,
    page_number         INT CHECK (page_number IS NULL OR page_number >= 1),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX catalog_pages_store_status_idx ON catalog_pages (store_id, status, display_order);
CREATE INDEX catalog_pages_store_order_idx ON catalog_pages (store_id, display_order);

-- Hotspots: product FK per CATALOG_LINK_CONTRACT; import candidates may set product later
CREATE TABLE catalog_page_hotspots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    catalog_page_id     UUID NOT NULL REFERENCES catalog_pages(id) ON DELETE CASCADE,
    product_id          UUID REFERENCES products(id) ON DELETE CASCADE,
    import_item_id      UUID REFERENCES catalog_import_items(id) ON DELETE SET NULL,
    x_pct               NUMERIC(6,3) NOT NULL CHECK (x_pct >= 0 AND x_pct <= 100),
    y_pct               NUMERIC(6,3) NOT NULL CHECK (y_pct >= 0 AND y_pct <= 100),
    width_pct           NUMERIC(6,3) NOT NULL CHECK (width_pct > 0 AND width_pct <= 100),
    height_pct          NUMERIC(6,3) NOT NULL CHECK (height_pct > 0 AND height_pct <= 100),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX catalog_page_hotspots_page_idx ON catalog_page_hotspots (catalog_page_id);
CREATE INDEX catalog_page_hotspots_store_idx ON catalog_page_hotspots (store_id, catalog_page_id);
CREATE INDEX catalog_page_hotspots_product_idx ON catalog_page_hotspots (store_id, product_id);
