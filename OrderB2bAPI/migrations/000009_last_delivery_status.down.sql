ALTER TABLE supplier_orders
DROP COLUMN IF EXISTS last_delivery_status,
DROP COLUMN IF EXISTS last_delivery_status_label,
DROP COLUMN IF EXISTS last_delivery_waybill,
DROP COLUMN IF EXISTS last_delivery_image_url,
DROP COLUMN IF EXISTS last_delivery_at;
