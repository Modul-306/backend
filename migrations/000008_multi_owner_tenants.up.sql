CREATE TABLE tenant_owners (
    tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (tenant_id, user_id)
);

-- Seed from existing owner_id in tenants
INSERT INTO tenant_owners (tenant_id, user_id)
SELECT id, owner_id FROM tenants WHERE owner_id IS NOT NULL;

-- Seed from users who have tenant_id
INSERT INTO tenant_owners (tenant_id, user_id)
SELECT tenant_id, id FROM users WHERE tenant_id IS NOT NULL
ON CONFLICT DO NOTHING;
