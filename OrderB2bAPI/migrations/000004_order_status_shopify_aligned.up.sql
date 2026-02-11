-- Migrate order statuses to Shopify-aligned values
UPDATE supplier_orders SET status = 'INCOMPLETE_CAUTION' WHERE status = 'PENDING_CONFIRMATION';
UPDATE supplier_orders SET status = 'UNFULFILLED' WHERE status = 'CONFIRMED';
UPDATE supplier_orders SET status = 'FULFILLED' WHERE status = 'SHIPPED';
UPDATE supplier_orders SET status = 'COMPLETE' WHERE status = 'DELIVERED';
UPDATE supplier_orders SET status = 'CANCELED' WHERE status = 'CANCELLED';
-- REJECTED stays as REJECTED

-- Update default for new orders
ALTER TABLE supplier_orders ALTER COLUMN status SET DEFAULT 'INCOMPLETE_CAUTION';
