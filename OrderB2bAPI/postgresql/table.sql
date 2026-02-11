SELECT id, partner_order_id, status, shopify_draft_order_id, created_at
FROM supplier_orders
ORDER BY created_at DESC
LIMIT 50;



SELECT sku, shopify_product_id, shopify_variant_id, is_active, created_at
FROM sku_mappings
ORDER BY created_at DESC;



