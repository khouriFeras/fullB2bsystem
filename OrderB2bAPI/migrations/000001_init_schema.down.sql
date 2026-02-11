-- Drop triggers
DROP TRIGGER IF EXISTS update_supplier_orders_updated_at ON supplier_orders;
DROP TRIGGER IF EXISTS update_sku_mappings_updated_at ON sku_mappings;
DROP TRIGGER IF EXISTS update_partners_updated_at ON partners;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables (in reverse order due to foreign keys)
DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS supplier_order_items;
DROP TABLE IF EXISTS supplier_orders;
DROP TABLE IF EXISTS sku_mappings;
DROP TABLE IF EXISTS partners;
