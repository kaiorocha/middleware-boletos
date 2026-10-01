ALTER TABLE webhook_events
    ADD COLUMN IF NOT EXISTS provider_payload TEXT,
    ADD COLUMN IF NOT EXISTS last_delivery_attempt_at TIMESTAMPTZ;

-- Preserve the original provider callback separately before payload becomes
-- the tenant-facing notification used by the delivery worker.
UPDATE webhook_events
SET provider_payload = payload
WHERE provider_id IS NOT NULL
  AND type = 'REGISTRO'
  AND provider_payload IS NULL;

CREATE INDEX IF NOT EXISTS idx_webhook_events_pending_delivery
    ON webhook_events(last_delivery_attempt_at, created_at)
    WHERE delivered_at IS NULL
      AND type IN ('BOLETO_PROVIDER_SYNC', 'BOLETO_TENANT_WEBHOOK');
