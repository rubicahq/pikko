-- Paid by admin: merchant-marked payment path (no bank evidence required).
-- Soft-adds method_type; bank_transfer remains the storefront default.

ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_method_type_check;
ALTER TABLE payments ADD CONSTRAINT payments_method_type_check
    CHECK (method_type IN ('bank_transfer', 'paid_by_admin'));
