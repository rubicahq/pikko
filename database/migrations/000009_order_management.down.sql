DROP TABLE IF EXISTS admin_order_drafts;
DROP TABLE IF EXISTS order_item_events;
DROP TABLE IF EXISTS order_revisions;

ALTER TABLE order_items
    DROP COLUMN IF EXISTS availability_status,
    DROP COLUMN IF EXISTS allocated_qty,
    DROP COLUMN IF EXISTS packed_qty,
    DROP COLUMN IF EXISTS adjusted_line_total_idr,
    DROP COLUMN IF EXISTS original_quantity;

ALTER TABLE orders
    DROP COLUMN IF EXISTS source,
    DROP COLUMN IF EXISTS created_by_user_id,
    DROP COLUMN IF EXISTS campaign_id;
