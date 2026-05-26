ALTER TABLE users DROP COLUMN loyalty_tier;
DROP TABLE IF EXISTS loyalty_discounts;
ALTER TABLE tenants DROP COLUMN category;
ALTER TABLE products DROP COLUMN category;
