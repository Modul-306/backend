ALTER TABLE users DROP CONSTRAINT IF EXISTS users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('platform_admin', 'farmer_admin', 'customer'));
DROP TABLE IF EXISTS product_reviews;
