DROP TABLE IF EXISTS payment_attempts;
DROP TABLE IF EXISTS store_doku_credentials;
ALTER TABLE store_payment_methods DROP CONSTRAINT IF EXISTS store_payment_methods_method_type_check;
ALTER TABLE store_payment_methods ADD CONSTRAINT store_payment_methods_method_type_check
    CHECK (method_type IN ('bank_transfer'));
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_type_check;
ALTER TABLE payments ADD CONSTRAINT payments_method_type_check
    CHECK (method_type IN ('bank_transfer', 'paid_by_admin'));
