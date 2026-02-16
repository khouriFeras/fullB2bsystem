-- Store last delivery status from Wassel webhook so we can show it even when partner has no webhook
ALTER TABLE supplier_orders
ADD COLUMN last_delivery_status INTEGER,
ADD COLUMN last_delivery_status_label VARCHAR(255),
ADD COLUMN last_delivery_waybill VARCHAR(255),
ADD COLUMN last_delivery_image_url TEXT,
ADD COLUMN last_delivery_at TIMESTAMPTZ;
