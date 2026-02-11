-- Revert to BIGINT (only numeric strings convert; order names like #1033 become NULL).
ALTER TABLE supplier_orders
  ALTER COLUMN shopify_order_id TYPE BIGINT USING (
    CASE WHEN shopify_order_id ~ '^\d+$' THEN shopify_order_id::bigint ELSE NULL END
  );
