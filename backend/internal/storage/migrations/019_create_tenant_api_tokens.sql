CREATE TABLE IF NOT EXISTS tenant_api_tokens (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    environment TEXT NOT NULL CHECK (environment IN ('HML', 'PRODUCTION')),
    token_hash TEXT NOT NULL UNIQUE,
    token_prefix TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'ACTIVE' CHECK (status IN ('ACTIVE', 'REVOKED')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_tenant_api_tokens_active_environment
    ON tenant_api_tokens (tenant_id, environment)
    WHERE status = 'ACTIVE';
