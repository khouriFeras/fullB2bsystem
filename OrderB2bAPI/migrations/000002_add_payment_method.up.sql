-- Add payment_method column to supplier_orders table
ALTER TABLE supplier_orders 
ADD COLUMN payment_method VARCHAR(50);

-- Add index for payment method queries
CREATE INDEX idx_supplier_orders_payment_method ON supplier_orders(payment_method);
