ALTER TABLE webhook_events
    ADD COLUMN IF NOT EXISTS provider_id UUID REFERENCES providers(id),
    ADD COLUMN IF NOT EXISTS external_event_id TEXT,
    ADD COLUMN IF NOT EXISTS item_sequence INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS delivered_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS delivery_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_delivery_error TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS webhook_events_provider_event_item_uidx
    ON webhook_events(provider_id, external_event_id, item_sequence)
    WHERE provider_id IS NOT NULL AND external_event_id IS NOT NULL;
