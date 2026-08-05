-- Evolve blacklist to support multiple entry types (DOCUMENT and EMAIL)
-- Add support for email-based blacklist entries

-- Add new columns for the enhanced blacklist schema
ALTER TABLE blacklist ADD COLUMN IF NOT EXISTS entry_type TEXT DEFAULT 'DOCUMENT';
ALTER TABLE blacklist ADD COLUMN IF NOT EXISTS value TEXT;
ALTER TABLE blacklist ADD COLUMN IF NOT EXISTS value_normalized TEXT;

-- Migrate existing document data to new schema
UPDATE blacklist 
SET 
    value = document,
    value_normalized = LOWER(TRIM(document))
WHERE value IS NULL AND document IS NOT NULL;

-- Create new unique index for (tenant_id, entry_type, value_normalized)
-- This ensures no duplicate entries within a tenant for the same type+value
CREATE UNIQUE INDEX IF NOT EXISTS idx_blacklist_tenant_type_value_unique
    ON blacklist(tenant_id, entry_type, value_normalized)
    WHERE active = true AND deleted_at IS NULL;

-- Create index for efficient queries by type
CREATE INDEX IF NOT EXISTS idx_blacklist_entry_type
    ON blacklist(entry_type, value_normalized);

-- Create index for efficient queries by tenant and type
CREATE INDEX IF NOT EXISTS idx_blacklist_tenant_type
    ON blacklist(tenant_id, entry_type)
    WHERE active = true;

-- Drop old unique index (it will be replaced by the new one)
DROP INDEX IF EXISTS idx_blacklist_tenant_document_active_unique;

-- Drop old document index (value_normalized index is more flexible)
DROP INDEX IF EXISTS idx_blacklist_document;

-- Create a better document index using normalized value
CREATE INDEX IF NOT EXISTS idx_blacklist_value_normalized
    ON blacklist(value_normalized);
