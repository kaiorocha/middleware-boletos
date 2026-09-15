ALTER TABLE providers
    ADD COLUMN IF NOT EXISTS webhook_token_hash TEXT;
