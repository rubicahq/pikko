-- Phase 5: Merchant Efficiency — notifications, CSV import, analytics foundation (PRD §18, §25, §28–29, §32, §40)

-- Allow CSV catalog uploads
ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop',
    'payment_evidence',
    'preorder_cover',
    'catalog_csv'
));

-- Extend catalog import jobs for CSV (PDF import remains unchanged)
ALTER TABLE catalog_import_jobs
    ALTER COLUMN pdf_file_id DROP NOT NULL;

ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS source_type TEXT NOT NULL DEFAULT 'pdf'
        CHECK (source_type IN ('pdf', 'csv'));

ALTER TABLE catalog_import_jobs
    ADD COLUMN IF NOT EXISTS csv_file_id UUID REFERENCES stored_files(id) ON DELETE RESTRICT;

ALTER TABLE catalog_import_jobs DROP CONSTRAINT IF EXISTS catalog_import_jobs_source_file_check;
ALTER TABLE catalog_import_jobs ADD CONSTRAINT catalog_import_jobs_source_file_check CHECK (
    (source_type = 'pdf' AND pdf_file_id IS NOT NULL) OR
    (source_type = 'csv' AND csv_file_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS catalog_import_jobs_source_idx
    ON catalog_import_jobs (store_id, source_type, created_at DESC);

-- In-app merchant notifications (store-scoped; PRD §29)
CREATE TABLE IF NOT EXISTS notifications (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    type                TEXT NOT NULL,
    title               TEXT NOT NULL,
    body                TEXT,
    resource_type       TEXT,
    resource_id         UUID,
    actor_user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    read_at             TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS notifications_store_created_idx
    ON notifications (store_id, created_at DESC);

CREATE INDEX IF NOT EXISTS notifications_store_unread_idx
    ON notifications (store_id, created_at DESC)
    WHERE read_at IS NULL;

-- Optional operational notification preferences (essential security emails cannot be disabled)
CREATE TABLE IF NOT EXISTS notification_preferences (
    store_id            UUID PRIMARY KEY REFERENCES stores(id) ON DELETE CASCADE,
    email_new_order     BOOLEAN NOT NULL DEFAULT true,
    email_payment_evidence BOOLEAN NOT NULL DEFAULT true,
    email_batch_closed  BOOLEAN NOT NULL DEFAULT true,
    email_purchasing_issues BOOLEAN NOT NULL DEFAULT true,
    inapp_new_order     BOOLEAN NOT NULL DEFAULT true,
    inapp_payment_evidence BOOLEAN NOT NULL DEFAULT true,
    inapp_batch_closed  BOOLEAN NOT NULL DEFAULT true,
    inapp_purchasing_issues BOOLEAN NOT NULL DEFAULT true,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Lightweight analytics event sink (optional; dashboards primarily query orders)
CREATE TABLE IF NOT EXISTS analytics_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id            UUID REFERENCES stores(id) ON DELETE CASCADE,
    event_name          TEXT NOT NULL,
    session_id          TEXT,
    user_id             UUID REFERENCES users(id) ON DELETE SET NULL,
    properties          JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS analytics_events_store_time_idx
    ON analytics_events (store_id, occurred_at DESC)
    WHERE store_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS analytics_events_name_idx
    ON analytics_events (event_name, occurred_at DESC);
