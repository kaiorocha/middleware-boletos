-- Add UNIQUE constraint for external_id to ensure idempotency
-- This constraint is partial: only applies when external_id is NOT NULL and deleted_at IS NULL

CREATE UNIQUE INDEX IF NOT EXISTS idx_boletos_tenant_external_id_unique
ON boletos (tenant_id, external_id)
WHERE external_id IS NOT NULL
  AND external_id <> ''
  AND deleted_at IS NULL;

-- Note: This index prevents duplicate (tenant_id, external_id) combinations
-- within the same tenant for active (non-deleted) boletos.
-- Multiple boletos can have external_id = NULL (soft-deleted or without external_id).
