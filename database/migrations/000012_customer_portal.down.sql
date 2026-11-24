DROP INDEX IF EXISTS portal_address_change_requests_order_pending_unique;
DROP TABLE IF EXISTS portal_address_change_requests CASCADE;
ALTER TABLE IF EXISTS customer_addresses DROP COLUMN IF EXISTS updated_at;
DROP TABLE IF EXISTS customer_refund_preferences CASCADE;
DROP TABLE IF EXISTS customer_portal_sessions CASCADE;
DROP TABLE IF EXISTS customer_portal_tokens CASCADE;
