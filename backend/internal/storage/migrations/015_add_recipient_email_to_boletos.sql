-- Add recipient_email column to boletos for proposal boleto support
-- Make customer_id nullable to allow proposal boletos without customer

-- Add recipient_email column (initially nullable, will be populated)
ALTER TABLE boletos ADD COLUMN IF NOT EXISTS recipient_email TEXT;

-- Make customer_id nullable (drop NOT NULL constraint)
-- Note: In PostgreSQL, we need to alter the column definition
ALTER TABLE boletos ALTER COLUMN customer_id DROP NOT NULL;

-- Add index for efficient tenant + recipient_email queries
CREATE INDEX IF NOT EXISTS idx_boletos_tenant_recipient_email
    ON boletos(tenant_id, recipient_email)
    WHERE recipient_email IS NOT NULL;

-- Add index for external_id query (helps with idempotency)
CREATE INDEX IF NOT EXISTS idx_boletos_external_id_tenant
    ON boletos(tenant_id, external_id)
    WHERE external_id IS NOT NULL;
