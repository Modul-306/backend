-- Make tenant_id nullable for global/customer users
ALTER TABLE users ALTER COLUMN tenant_id DROP NOT NULL;

-- Add owner_id to tenants (nullable = existing tenants have no owner)
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS owner_id UUID REFERENCES users(id) ON DELETE SET NULL;
