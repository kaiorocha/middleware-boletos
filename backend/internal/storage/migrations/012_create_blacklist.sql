CREATE TABLE IF NOT EXISTS blacklist (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    document TEXT NOT NULL,
    name TEXT,
    reason TEXT,
    notes TEXT,
    source TEXT NOT NULL DEFAULT 'MANUAL',
    created_by UUID,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_blacklist_tenant_id
    ON blacklist (tenant_id);

CREATE INDEX IF NOT EXISTS idx_blacklist_document
    ON blacklist (document);

CREATE INDEX IF NOT EXISTS idx_blacklist_active
    ON blacklist (active);

CREATE UNIQUE INDEX IF NOT EXISTS idx_blacklist_tenant_document_active_unique
    ON blacklist (tenant_id, document)
    WHERE active = true AND deleted_at IS NULL;
