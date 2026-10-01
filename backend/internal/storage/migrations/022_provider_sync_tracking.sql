ALTER TABLE boletos
    ADD COLUMN IF NOT EXISTS last_provider_sync_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_boletos_provider_sync_due
    ON boletos (COALESCE(last_provider_sync_at, updated_at))
    WHERE deleted_at IS NULL
      AND our_number IS NOT NULL
      AND status IN ('PROCESSING', 'ISSUED', 'PARTIAL');
