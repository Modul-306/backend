ALTER TABLE tenants ADD COLUMN allows_online_payment BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE tenants ADD COLUMN allows_cash_payment BOOLEAN NOT NULL DEFAULT TRUE;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check CHECK (status IN ('pending', 'pending_payment', 'completed', 'cancelled'));

ALTER TABLE orders ADD COLUMN payment_method TEXT NOT NULL DEFAULT 'cash' CHECK (payment_method IN ('online', 'cash'));
ALTER TABLE orders ADD COLUMN payrexx_gateway_id INT UNIQUE;
ALTER TABLE orders ADD COLUMN payment_status TEXT NOT NULL DEFAULT 'unpaid';
