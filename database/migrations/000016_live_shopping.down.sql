-- Dev/local rollback only. Do not run against production with live data.

DROP TABLE IF EXISTS live_quick_adds;

DROP INDEX IF EXISTS orders_store_campaign_created_idx;

ALTER TABLE order_item_events DROP CONSTRAINT IF EXISTS order_item_events_event_type_check;
ALTER TABLE order_item_events ADD CONSTRAINT order_item_events_event_type_check
    CHECK (event_type IN (
        'created', 'qty_adjusted', 'availability_changed', 'allocated', 'packed'
    ));

ALTER TABLE order_revisions DROP CONSTRAINT IF EXISTS order_revisions_change_type_check;
ALTER TABLE order_revisions ADD CONSTRAINT order_revisions_change_type_check
    CHECK (change_type IN (
        'created', 'status', 'item_qty', 'item_unavailable', 'notes', 'import', 'bulk'
    ));

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_source_check;
ALTER TABLE orders ADD CONSTRAINT orders_source_check
    CHECK (source IN ('customer', 'admin', 'csv_import'));

DROP INDEX IF EXISTS campaign_products_featured_idx;
ALTER TABLE campaign_products DROP COLUMN IF EXISTS featured;
