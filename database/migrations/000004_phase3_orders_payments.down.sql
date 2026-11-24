DROP TABLE IF EXISTS store_order_counters;
DROP TABLE IF EXISTS public_rate_limits;
DROP TABLE IF EXISTS inventory_adjustments;
DROP TABLE IF EXISTS inventory_reservations;
DROP TABLE IF EXISTS payment_evidence;
DROP TABLE IF EXISTS payments;
DROP TABLE IF EXISTS order_tracking_tokens;
DROP TABLE IF EXISTS order_status_history;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS customer_addresses;
DROP TABLE IF EXISTS customers;
DROP TABLE IF EXISTS store_fulfillment_settings;
DROP TABLE IF EXISTS store_payment_methods;

ALTER TABLE stored_files DROP CONSTRAINT IF EXISTS stored_files_purpose_check;
ALTER TABLE stored_files ADD CONSTRAINT stored_files_purpose_check CHECK (purpose IN (
    'logo', 'favicon', 'store_icon', 'product', 'banner', 'collection', 'other',
    'catalog_pdf', 'catalog_page', 'import_crop'
));
