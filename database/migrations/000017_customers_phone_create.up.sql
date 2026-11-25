-- Allow store customers without an order: name + phone/WA only (email optional).
-- Postgres UNIQUE allows multiple NULLs, so existing ON CONFLICT (store_id, email)
-- upserts from checkout/admin orders keep working.
-- Portal access for phone-only customers uses customer_id on portal tokens.

ALTER TABLE customers
    ALTER COLUMN email DROP NOT NULL;

-- Keep customers_store_email_unique (store_id, email) as-is; NULL emails are distinct.

CREATE UNIQUE INDEX IF NOT EXISTS customers_store_phone_unique
    ON customers (store_id, phone)
    WHERE phone IS NOT NULL;

ALTER TABLE customer_portal_tokens
    ADD COLUMN IF NOT EXISTS customer_id UUID REFERENCES customers(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS customer_portal_tokens_customer_idx
    ON customer_portal_tokens (store_id, customer_id, created_at DESC)
    WHERE customer_id IS NOT NULL;
