-- Revert to original status values
UPDATE supplier_orders SET status = 'PENDING_CONFIRMATION' WHERE status = 'INCOMPLETE_CAUTION';
UPDATE supplier_orders SET status = 'CONFIRMED' WHERE status = 'UNFULFILLED';
UPDATE supplier_orders SET status = 'SHIPPED' WHERE status = 'FULFILLED';
UPDATE supplier_orders SET status = 'DELIVERED' WHERE status = 'COMPLETE';
UPDATE supplier_orders SET status = 'CANCELLED' WHERE status = 'CANCELED';

ALTER TABLE supplier_orders ALTER COLUMN status SET DEFAULT 'PENDING_CONFIRMATION';
