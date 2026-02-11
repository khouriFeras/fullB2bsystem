-- Add shopify_order_id column to supplier_orders table (finalized Shopify order id)
ALTER TABLE supplier_orders
ADD COLUMN shopify_order_id BIGINT;

CREATE INDEX idx_supplier_orders_shopify_order_id ON supplier_orders(shopify_order_id);

