-- Add Category to Products
ALTER TABLE products ADD COLUMN category VARCHAR(50) DEFAULT 'General';

-- Add Category/Specialty to Tenants
ALTER TABLE tenants ADD COLUMN category VARCHAR(50) DEFAULT 'General';

-- Add Discounts table for Loyalty Tiers
CREATE TABLE loyalty_discounts (
    tier_name VARCHAR(20) PRIMARY KEY,
    discount_percent DECIMAL(5,2) NOT NULL
);

INSERT INTO loyalty_discounts (tier_name, discount_percent) VALUES
('Seedling', 0.00),
('Sprout', 2.00),
('Harvester', 5.00),
('Harvest Elite', 10.00);

-- Add tier to users
ALTER TABLE users ADD COLUMN loyalty_tier VARCHAR(20) DEFAULT 'Seedling' REFERENCES loyalty_discounts(tier_name);
