CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    status TEXT NOT NULL DEFAULT 'DRAFT',
    created_by UUID REFERENCES users(id),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT campaigns_status_check CHECK (status IN ('DRAFT','READY','PROCESSING','COMPLETED','PARTIAL','FAILED','CANCELLED'))
);

ALTER TABLE boletos ADD COLUMN IF NOT EXISTS campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL;
ALTER TABLE boletos ADD COLUMN IF NOT EXISTS campaign_claimed_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_campaigns_tenant_created ON campaigns (tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_campaigns_tenant_status ON campaigns (tenant_id, status);
CREATE INDEX IF NOT EXISTS idx_boletos_campaign_status ON boletos (campaign_id, status) WHERE campaign_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_boletos_campaign_due_date ON boletos (campaign_id, due_date) WHERE campaign_id IS NOT NULL AND deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS campaign_imports (
    id UUID PRIMARY KEY,
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    checksum TEXT NOT NULL,
    total_rows INTEGER NOT NULL,
    valid_rows INTEGER NOT NULL,
    invalid_rows INTEGER NOT NULL,
    total_amount_cents BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'PREVIEWED',
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (campaign_id, checksum)
);

CREATE TABLE IF NOT EXISTS campaign_import_rows (
    import_id UUID NOT NULL REFERENCES campaign_imports(id) ON DELETE CASCADE,
    row_number INTEGER NOT NULL,
    recipient_email TEXT,
    amount_cents BIGINT,
    due_date DATE,
    external_id TEXT,
    field TEXT,
    error_code TEXT,
    error_message TEXT,
    PRIMARY KEY (import_id, row_number)
);

CREATE INDEX IF NOT EXISTS idx_campaign_import_rows_errors ON campaign_import_rows (import_id, row_number) WHERE error_code IS NOT NULL;
