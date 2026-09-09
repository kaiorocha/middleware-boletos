-- Store payer identity supplied by public integrations. Both fields are optional
-- so existing email-only proposal boletos remain compatible.
ALTER TABLE boletos ADD COLUMN IF NOT EXISTS payer_name TEXT;
ALTER TABLE boletos ADD COLUMN IF NOT EXISTS payer_document TEXT;

CREATE INDEX IF NOT EXISTS idx_boletos_tenant_payer_document
    ON boletos(tenant_id, payer_document)
    WHERE payer_document IS NOT NULL;
