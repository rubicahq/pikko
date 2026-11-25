-- Revert only when no paid_by_admin rows remain.
UPDATE payments SET method_type = 'bank_transfer' WHERE method_type = 'paid_by_admin';

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_type_check;
ALTER TABLE payments ADD CONSTRAINT payments_method_type_check
    CHECK (method_type IN ('bank_transfer'));
