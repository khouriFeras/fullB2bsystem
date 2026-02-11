-- Store Shopify order number (e.g. #1033) instead of numeric ID.
-- Existing numeric values become string; delete orders if you need only names.
ALTER TABLE supplier_orders
  ALTER COLUMN shopify_order_id TYPE VARCHAR(50) USING (
    CASE WHEN shopify_order_id IS NULL THEN NULL ELSE shopify_order_id::text END
  );
