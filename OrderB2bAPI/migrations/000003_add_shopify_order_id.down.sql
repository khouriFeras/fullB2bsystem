-- Remove shopify_order_id column
DROP INDEX IF EXISTS idx_supplier_orders_shopify_order_id;
ALTER TABLE supplier_orders DROP COLUMN IF EXISTS shopify_order_id;

