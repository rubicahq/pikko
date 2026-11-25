DROP INDEX IF EXISTS customer_portal_tokens_customer_idx;

ALTER TABLE customer_portal_tokens
    DROP COLUMN IF EXISTS customer_id;

DROP INDEX IF EXISTS customers_store_phone_unique;

UPDATE customers SET email = 'unknown+' || id::text || '@customers.local' WHERE email IS NULL;

ALTER TABLE customers
    ALTER COLUMN email SET NOT NULL;
