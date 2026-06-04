ALTER TABLE tenants DROP COLUMN IF EXISTS allows_online_payment;
ALTER TABLE tenants DROP COLUMN IF EXISTS allows_cash_payment;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('pending', 'completed', 'cancelled'));

ALTER TABLE orders DROP COLUMN IF EXISTS payment_method;
ALTER TABLE orders DROP COLUMN IF EXISTS payrexx_gateway_id;
ALTER TABLE orders DROP COLUMN IF EXISTS payment_status;
