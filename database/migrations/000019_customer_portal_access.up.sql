-- Customer portal access: durable 4-char secret + phone URL (no email magic-link).
-- Format: /{store_slug}/customer/{secret}-{phone}

ALTER TABLE customers
    ADD COLUMN IF NOT EXISTS portal_access_secret TEXT;

-- Backfill existing rows (hex from md5 is [0-9a-f], valid lowercase alphanumeric).
UPDATE customers
SET portal_access_secret = lower(substr(md5(id::text || coalesce(phone, '') || random()::text), 1, 4))
WHERE portal_access_secret IS NULL OR portal_access_secret = '';

ALTER TABLE customers
    ALTER COLUMN portal_access_secret SET NOT NULL;

ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS customers_portal_access_secret_format;

ALTER TABLE customers
    ADD CONSTRAINT customers_portal_access_secret_format
        CHECK (portal_access_secret ~ '^[a-z0-9]{4}$');

CREATE INDEX IF NOT EXISTS customers_store_portal_access_idx
    ON customers (store_id, phone, portal_access_secret)
    WHERE phone IS NOT NULL;
