DROP INDEX IF EXISTS customers_store_portal_access_idx;

ALTER TABLE customers
    DROP CONSTRAINT IF EXISTS customers_portal_access_secret_format;

ALTER TABLE customers
    DROP COLUMN IF EXISTS portal_access_secret;
