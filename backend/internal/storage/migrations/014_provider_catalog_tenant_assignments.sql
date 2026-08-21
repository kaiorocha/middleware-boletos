CREATE EXTENSION IF NOT EXISTS pgcrypto;

ALTER TABLE providers
    ALTER COLUMN tenant_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS type TEXT,
    ADD COLUMN IF NOT EXISTS metadata TEXT;

CREATE TABLE IF NOT EXISTS tenant_providers (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES providers(id) ON DELETE CASCADE,
    active BOOLEAN NOT NULL DEFAULT true,
    config TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    UNIQUE (tenant_id, provider_id)
);

INSERT INTO tenant_providers (id, tenant_id, provider_id, active, config, created_at, updated_at)
SELECT gen_random_uuid(), tenant_id, id, status = 'ACTIVE', config, created_at, updated_at
FROM providers
WHERE tenant_id IS NOT NULL
ON CONFLICT (tenant_id, provider_id) DO NOTHING;

