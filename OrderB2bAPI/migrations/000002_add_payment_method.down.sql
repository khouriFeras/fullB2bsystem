-- Remove payment_method column
DROP INDEX IF EXISTS idx_supplier_orders_payment_method;
ALTER TABLE supplier_orders DROP COLUMN IF EXISTS payment_method;
